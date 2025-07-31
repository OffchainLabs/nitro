package arbnode

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/bold/solgen/go/node_interfacegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	ForceInclusionErr = errors.New("force inclusion is going to happen")
)

type ForceInclusionCheckerConfig struct {
	RetryTime                time.Duration `koanf:"retry-time"`
	PollingInterval          time.Duration `koanf:"polling-interval"`
	BlockThresholdTolerance  uint64        `koanf:"block-threshold-tolerance"`
	SecondThresholdTolerance uint64        `koanf:"second-threshold-tolerance"`

	ErrorToleranceDuration time.Duration `koanf:"error-tolerance-duration"`
}

var DefaultEspressoForceInclusionCheckerConfig = ForceInclusionCheckerConfig{
	RetryTime:                time.Second * 2,
	PollingInterval:          time.Minute * 8,
	BlockThresholdTolerance:  20,
	SecondThresholdTolerance: 200,
	ErrorToleranceDuration:   time.Minute * 8,
}

func EspressoForceInclusionConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".retry-time", DefaultEspressoForceInclusionCheckerConfig.RetryTime, "retry time after a failure")
	f.Duration(prefix+".polling-interval", DefaultEspressoForceInclusionCheckerConfig.PollingInterval, "time after a success")
	f.Uint64(prefix+".block-threshold-tolerance", DefaultEspressoForceInclusionCheckerConfig.BlockThresholdTolerance, "block threshold tolerance")
	f.Uint64(prefix+".second-threshold-tolerance", DefaultEspressoForceInclusionCheckerConfig.SecondThresholdTolerance, "second threshold tolerance")
	f.Duration(prefix+".error-tolerance-duration", DefaultEspressoForceInclusionCheckerConfig.ErrorToleranceDuration, "error tolerance duration")
}

// SeqInboxInterface defines an interface for interacting with the sequencer inbox contract.
// Note: When `deployBold` is disabled, the [MaxTimeVariation](arbnode/espresso_force_inclusion_checker.go:14:1-14:78) values are hardcoded,
// which makes this interface difficult to mock in tests.
type SeqInboxInterface interface {
	MaxTimeVariation(context.Context) (*big.Int, *big.Int, *big.Int, *big.Int, error)
	TotalDelayedMessagesRead(context.Context) (*big.Int, error)
}

type SeqInbox struct {
	seqInbox *bridgegen.SequencerInbox
}

func (s *SeqInbox) MaxTimeVariation(ctx context.Context) (*big.Int, *big.Int, *big.Int, *big.Int, error) {
	return s.seqInbox.MaxTimeVariation(&bind.CallOpts{Context: ctx})
}

func (s *SeqInbox) TotalDelayedMessagesRead(ctx context.Context) (*big.Int, error) {
	return s.seqInbox.TotalDelayedMessagesRead(&bind.CallOpts{Context: ctx})
}

type ForceInclusionChecker struct {
	stopwaiter.StopWaiter

	seqInbox              SeqInboxInterface
	config                ForceInclusionCheckerConfig
	l1Reader              *headerreader.HeaderReader
	delayedMessageFetcher *DelayedMessageFetcher
	fatalErrChan          chan error
}

func NewForceInclusionChecker(
	seqInbox SeqInboxInterface,
	config ForceInclusionCheckerConfig,
	l1Reader *headerreader.HeaderReader,
	delayedMessageFetcher *DelayedMessageFetcher,
	fatalErrChan chan error,
) *ForceInclusionChecker {
	return &ForceInclusionChecker{
		seqInbox:              seqInbox,
		config:                config,
		l1Reader:              l1Reader,
		delayedMessageFetcher: delayedMessageFetcher,
		fatalErrChan:          fatalErrChan,
	}
}

func (f *ForceInclusionChecker) checkIfMessageCanBeForceIncluded(ctx context.Context) error {
	// Get the total number of delayed messages read in the sequencer inbox
	totalDelayedMessagesRead, err := f.seqInbox.TotalDelayedMessagesRead(ctx)
	if err != nil {
		return fmt.Errorf("error getting total delayed messages read: %w", err)
	}
	log.Debug("Total delayed messages read", "totalDelayedMessagesRead", totalDelayedMessagesRead)

	// Get the earliest block number that is without the force inclusion tolerance
	badBlockNumber, err := f.getForceInclusionToleranceBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("error getting force inclusion tolerance block number: %w", err)
	}
	log.Debug("Force inclusion tolerance bad block number", "blockNumber", badBlockNumber)
	// Check the delayed message count at this block number
	count, err := f.delayedMessageFetcher.getDelayedMessageLatestIndexAtBlock(badBlockNumber)
	if err != nil {
		return fmt.Errorf("error getting delayed message count at block %d: %w", badBlockNumber, err)
	}
	log.Debug("Delayed message count at block", "count", count, "blockNumber", badBlockNumber)
	// If the message count in delay inbox is less than or equal to the total delayed messages read
	// then no force inclusion is going to happen.
	if count <= arbmath.BigToUintSaturating(totalDelayedMessagesRead) {
		log.Debug("force inclusion wont happen")
		return nil
	}
	log.Debug("Force inclusion is going to happen")
	// Force inclusion is going to happen, panic the node.
	return ForceInclusionErr
}

func (f *ForceInclusionChecker) Start(ctx context.Context) error {
	f.StopWaiter.Start(ctx, f)
	var firstErrFound time.Time

	return f.CallIterativelySafe(func(ctx context.Context) time.Duration {
		err := f.checkIfMessageCanBeForceIncluded(ctx)
		if err == nil {
			firstErrFound = time.Time{}
			return f.config.PollingInterval
		}
		if errors.Is(err, ForceInclusionErr) {
			log.Error("rorce inclusion error", "err", err)
			f.fatalErrChan <- err
			return 0
		}
		if firstErrFound.IsZero() {
			firstErrFound = time.Now()
		} else if time.Since(firstErrFound) > f.config.ErrorToleranceDuration {
			log.Error("error tolerance duration exceeded", "firstErrFound", firstErrFound, "timeSinceFirstErrFound", time.Since(firstErrFound))
			f.fatalErrChan <- err
		}
		log.Error("error checking force inclusion", "err", err)
		return f.config.RetryTime
	})
}

func (f *ForceInclusionChecker) getForceInclusionToleranceBlockNumber(ctx context.Context) (uint64, error) {
	maxTimeVariationDelayBlocks, _, maxTimeVariationDelaySeconds, _, err := f.seqInbox.MaxTimeVariation(ctx)
	if err != nil {
		return 0, err
	}
	log.Debug("Max time variation delay blocks", "maxTimeVariationDelayBlocks", maxTimeVariationDelayBlocks, "maxTimeVariationDelaySeconds", maxTimeVariationDelaySeconds)

	parentLatestHeader, err := f.l1Reader.LastHeader(ctx)
	if err != nil {
		return 0, err
	}

	l1BlockNumber := parentLatestHeader.Number.Uint64()
	l1TimeStamp := parentLatestHeader.Time

	log.Debug("L1 block number", "l1BlockNumber", l1BlockNumber, "l1TimeStamp", l1TimeStamp)

	if f.l1Reader.IsParentChainArbitrum() {
		headerInfo := types.DeserializeHeaderExtraInformation(parentLatestHeader)
		l1BlockNumber = headerInfo.L1BlockNumber
	}

	lastBadBlockNumber := arbmath.SaturatingUSub(f.config.BlockThresholdTolerance+l1BlockNumber, arbmath.BigToUintSaturating(maxTimeVariationDelayBlocks))
	lastBadBlockTime := arbmath.SaturatingUSub(f.config.SecondThresholdTolerance+l1TimeStamp, arbmath.BigToUintSaturating(maxTimeVariationDelaySeconds))

	log.Debug("Last bad block number", "lastBadBlockNumber", lastBadBlockNumber, "lastBadBlockTime", lastBadBlockTime)

	if f.l1Reader.IsParentChainArbitrum() {
		n, err := node_interfacegen.NewNodeInterface(types.NodeInterfaceAddress, f.l1Reader.Client())
		if err != nil {
			return 0, err
		}
		rng, err := n.L2BlockRangeForL1(&bind.CallOpts{Context: ctx}, lastBadBlockNumber)
		if err == nil {
			lastBadBlockNumber = rng.LastBlock
			log.Debug("Last bad block number from L2 block range for L1", "lastBadBlockNumber", lastBadBlockNumber)
		} else {
			// If the L2 block range for L1 call fails, we will use binary search
			// The start block number should be the genesis block
			// Reference: https://github.com/OffchainLabs/arbitrum-sdk/blob/792a7ee3ccf09842653bc49b771671706894cbb4/src/lib/inbox/inbox.ts#L104-L113
			genesis, err := n.NitroGenesisBlock(&bind.CallOpts{Context: ctx})
			if err != nil {
				return 0, err
			}
			target := lastBadBlockNumber
			start := genesis.Uint64()
			end := parentLatestHeader.Number.Uint64()
			lastBadBlockNumber, err = binarySearchForBlockNumber(ctx, start, end, func(ctx context.Context, blockNumber uint64) (int, error) {
				block, err := f.l1Reader.Client().BlockByNumber(ctx, arbmath.UintToBig(blockNumber))
				if err != nil {
					return 0, err
				}
				l1Block := types.DeserializeHeaderExtraInformation(block.Header()).L1BlockNumber
				if l1Block < target {
					return binarySearch_LessThanTarget, nil
				} else if l1Block > target {
					return binarySearch_GreaterThanTarget, nil
				} else {
					return binarySearch_EqualToTarget, nil
				}
			})
			if err != nil {
				log.Error("error in binary search", "err", err)
				return 0, err
			}
			log.Debug("Last bad block number from binary search", "lastBadBlockNumber", lastBadBlockNumber)
		}
	}

	lastBadBlock, err := f.findFirstParentChainBlockBelow(ctx, lastBadBlockNumber, lastBadBlockTime)
	if err != nil {
		log.Error("error finding first parent chain block below", "err", err)
		return 0, err
	}
	return lastBadBlock, nil
}

func (f *ForceInclusionChecker) findFirstParentChainBlockBelow(ctx context.Context, lastBadBlockNumber uint64, lastBadBlockTime uint64) (uint64, error) {
	client := f.l1Reader.Client()
	blockNumber := lastBadBlockNumber

	for blockNumber > 0 {
		block, err := client.BlockByNumber(ctx, arbmath.UintToBig(blockNumber))
		if err != nil {
			log.Error("Error getting block", "blockNumber", blockNumber, "err", err)
			return 0, err
		}
		if block.NumberU64() <= lastBadBlockNumber || block.Time() <= lastBadBlockTime {
			log.Debug("Block number is less than or equal to last bad block number or time", "blockNumber", block.NumberU64(), "lastBadBlockNumber", lastBadBlockNumber, "lastBadBlockTime", lastBadBlockTime)
			return block.NumberU64(), nil
		}
		log.Debug("Block number Decreasing", "blockNumber", blockNumber)
		blockNumber--
	}
	return 0, fmt.Errorf("no parent block found")
}
