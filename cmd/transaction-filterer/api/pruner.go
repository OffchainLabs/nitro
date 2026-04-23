// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package api

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos"
)

type PruneConfig struct {
	Enable               bool          `koanf:"enable"`
	StartDelayedMsgIdx   uint64        `koanf:"start-delayed-msg-idx"`
	StartParentBlock     uint64        `koanf:"start-parent-block"`
	PollInterval         time.Duration `koanf:"poll-interval"`
	ParentBlockChunkSize uint64        `koanf:"parent-block-chunk-size"`
}

var DefaultPruneConfig = PruneConfig{
	Enable:               false,
	PollInterval:         time.Minute,
	ParentBlockChunkSize: 1000,
}

func (c *PruneConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.PollInterval <= 0 {
		return errors.New("pruning.poll-interval must be positive")
	}
	if c.ParentBlockChunkSize == 0 {
		return errors.New("pruning.parent-block-chunk-size must be positive")
	}
	return nil
}

func PruneConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultPruneConfig.Enable, "enable background pruning of entries from the on-chain filter set once the corresponding delayed messages are sequenced")
	f.Uint64(prefix+".start-delayed-msg-idx", DefaultPruneConfig.StartDelayedMsgIdx, "delayed message index from which the pruner starts processing")
	f.Uint64(prefix+".start-parent-block", DefaultPruneConfig.StartParentBlock, "parent chain block from which the pruner starts scanning")
	f.Duration(prefix+".poll-interval", DefaultPruneConfig.PollInterval, "interval between parent chain scans")
	f.Uint64(prefix+".parent-block-chunk-size", DefaultPruneConfig.ParentBlockChunkSize, "max parent chain block range per LookupMessagesInRange call")
}

// PruneOptions bundles the runtime dependencies the pruner needs.
// All fields are required when Config.Enable is true.
type PruneOptions struct {
	Config            PruneConfig
	ChainId           *big.Int
	ParentChainClient *ethclient.Client
	ChildChainClient  *ethclient.Client
	DelayedBridge     *arbnode.DelayedBridge
}

func validatePruneOptions(p *PruneOptions) error {
	if p == nil || !p.Config.Enable {
		return nil
	}
	if err := p.Config.Validate(); err != nil {
		return err
	}
	if p.ChainId == nil {
		return errors.New("pruning enabled but chain ID not provided")
	}
	if p.ParentChainClient == nil {
		return errors.New("pruning enabled but parent chain client not provided")
	}
	if p.ChildChainClient == nil {
		return errors.New("pruning enabled but child chain client not provided")
	}
	if p.DelayedBridge == nil {
		return errors.New("pruning enabled but delayed bridge not provided")
	}
	return nil
}

// pruner scans parent-chain delayed messages in order, gates on child-chain finality,
// and yields the transaction hashes contained in each finalized-and-sequenced message.
// Consumers decide what to do with the hashes (typically: IsTransactionFiltered + unfilter).
type pruner struct {
	config            PruneConfig
	chainId           *big.Int
	parentChainClient *ethclient.Client
	childChainClient  *ethclient.Client
	delayedBridge     *arbnode.DelayedBridge
	nextParentBlock   uint64
	nextDelayedMsgIdx uint64
}

func newPruner(opts *PruneOptions) (*pruner, error) {
	if err := validatePruneOptions(opts); err != nil {
		return nil, err
	}
	p := &pruner{}
	if opts == nil || !opts.Config.Enable {
		return p, nil
	}
	p.config = opts.Config
	p.chainId = opts.ChainId
	p.parentChainClient = opts.ParentChainClient
	p.childChainClient = opts.ChildChainClient
	p.delayedBridge = opts.DelayedBridge
	p.nextParentBlock = opts.Config.StartParentBlock
	p.nextDelayedMsgIdx = opts.Config.StartDelayedMsgIdx
	return p, nil
}

// pruneResult carries the output of a single scan tick.
// FinalizedChildNumber is the child-chain block the caller should use for state-read CallOpts.
type pruneResult struct {
	Hashes               []common.Hash
	FinalizedChildNumber *big.Int
}

func (p *pruner) step(ctx context.Context) (pruneResult, error) {
	finalizedChild, err := p.childChainClient.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		return pruneResult{}, fmt.Errorf("child chain finalized header: %w", err)
	}
	// L2 block header Nonce holds the cumulative count of delayed messages sequenced up to and including this block.
	cumulativeDelayed := finalizedChild.Nonce.Uint64()
	if p.nextDelayedMsgIdx >= cumulativeDelayed {
		return pruneResult{}, nil
	}

	finalizedParent, err := p.parentChainClient.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		return pruneResult{}, fmt.Errorf("parent chain finalized header: %w", err)
	}
	finalizedParentNum := finalizedParent.Number.Uint64()
	if p.nextParentBlock > finalizedParentNum {
		return pruneResult{}, nil
	}

	toBlock := min(p.nextParentBlock+p.config.ParentBlockChunkSize-1, finalizedParentNum)
	msgs, err := p.delayedBridge.LookupMessagesInRange(
		ctx,
		new(big.Int).SetUint64(p.nextParentBlock),
		new(big.Int).SetUint64(toBlock),
		nil,
	)
	if err != nil {
		return pruneResult{}, fmt.Errorf("lookup delayed messages: %w", err)
	}

	hashes, stopParentBlock, stopped := p.collectHashes(msgs, cumulativeDelayed)
	if stopped {
		p.nextParentBlock = stopParentBlock
	} else {
		p.nextParentBlock = toBlock + 1
	}
	return pruneResult{Hashes: hashes, FinalizedChildNumber: finalizedChild.Number}, nil
}

// collectHashes iterates sorted-by-idx delayed messages, parses each in order, and accumulates
// the contained tx hashes. Returns (hashes, stopParentBlock, stopped). stopped=true means the
// scan halted mid-range because the next msg is past child-chain finality or has an idx gap;
// the caller should resume from stopParentBlock next tick.
func (p *pruner) collectHashes(msgs []*mel.DelayedInboxMessage, cumulativeDelayed uint64) ([]common.Hash, uint64, bool) {
	var hashes []common.Hash
	for _, msg := range msgs {
		if msg.Message.Header.RequestId == nil {
			continue
		}
		idx := new(big.Int).SetBytes(msg.Message.Header.RequestId[:]).Uint64()
		if idx < p.nextDelayedMsgIdx {
			continue
		}
		if idx != p.nextDelayedMsgIdx {
			log.Error("delayed message index gap", "expected", p.nextDelayedMsgIdx, "got", idx, "parentBlock", msg.ParentChainBlockNumber)
			return hashes, msg.ParentChainBlockNumber, true
		}
		if idx >= cumulativeDelayed {
			return hashes, msg.ParentChainBlockNumber, true
		}
		// ArbOS version only matters for L1MessageType_BatchPostingReport (internal tx, never filtered),
		// so the max supported version parses every user-facing message kind correctly.
		txs, err := arbos.ParseL2Transactions(msg.Message, p.chainId, params.MaxDebugArbosVersionSupported)
		if err != nil {
			// Same log-and-continue handling as arbos/block_processor.go and execution/gethexec/executionengine.go.
			log.Warn("error parsing incoming delayed message", "idx", idx, "kind", msg.Message.Header.Kind, "err", err)
		} else {
			for _, tx := range txs {
				hashes = append(hashes, tx.Hash())
			}
		}
		p.nextDelayedMsgIdx = idx + 1
	}
	return hashes, 0, false
}
