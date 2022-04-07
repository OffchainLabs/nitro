// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package validator

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
)

const txTimeout time.Duration = 5 * time.Minute

type StakerStrategy uint8

const (
	// Watchtower: don't do anything on L1, but log if there's a bad assertion
	WatchtowerStrategy StakerStrategy = iota
	// Defensive: stake if there's a bad assertion
	DefensiveStrategy
	// Stake latest: stay staked on the latest node, challenging bad assertions
	StakeLatestStrategy
	// Make nodes: continually create new nodes, challenging bad assertions
	MakeNodesStrategy
)

type L1PostingStrategy struct {
	HighGasThreshold   float64 `koanf:"high-gas-threshold"`
	HighGasDelayBlocks int64   `koanf:"high-gas-delay-blocks"`
}

var DefaultL1PostingStrategy = L1PostingStrategy{
	HighGasThreshold:   0,
	HighGasDelayBlocks: 0,
}

func L1PostingStrategyAddOptions(prefix string, f *flag.FlagSet) {
	f.Float64(prefix+".high-gas-threshold", DefaultL1PostingStrategy.HighGasThreshold, "high gas threshold")
	f.Int64(prefix+".high-gas-delay-blocks", DefaultL1PostingStrategy.HighGasDelayBlocks, "high gas delay blocks")
}

type L1ValidatorConfig struct {
	Enable              bool              `koanf:"enable"`
	Strategy            string            `koanf:"strategy"`
	StakerInterval      time.Duration     `koanf:"staker-interval"`
	L1PostingStrategy   L1PostingStrategy `koanf:"posting-strategy"`
	DisableChallenge    bool              `koanf:"disable-challenge"`
	WithdrawDestination string            `koanf:"withdraw-destination"`
	TargetMachineCount  int               `koanf:"target-machine-count"`
	ConfirmationBlocks  int64             `koanf:"confirmation-blocks"`
	Dangerous           DangerousConfig   `koanf:"dangerous"`
}

var DefaultL1ValidatorConfig = L1ValidatorConfig{
	Enable:              false,
	Strategy:            "Watchtower",
	StakerInterval:      time.Minute,
	L1PostingStrategy:   L1PostingStrategy{},
	DisableChallenge:    false,
	WithdrawDestination: "",
	TargetMachineCount:  4,
	ConfirmationBlocks:  12,
	Dangerous:           DangerousConfig{},
}

func L1ValidatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultL1ValidatorConfig.Enable, "enable validator")
	f.String(prefix+".strategy", DefaultL1ValidatorConfig.Strategy, "L1 validator strategy, either watchtower, defensive, stakeLatest, or makeNodes")
	f.Duration(prefix+".staker-interval", DefaultL1ValidatorConfig.StakerInterval, "how often the L1 validator should check the status of the L1 rollup and maybe take action with its stake")
	L1PostingStrategyAddOptions(prefix+".posting-strategy", f)
	f.Bool(prefix+".disable-challenge", DefaultL1ValidatorConfig.DisableChallenge, "disable validator challenge")
	f.String(prefix+".withdraw-destination", DefaultL1ValidatorConfig.WithdrawDestination, "validator withdraw destination")
	f.Int(prefix+".target-machine-count", DefaultL1ValidatorConfig.TargetMachineCount, "target machine count")
	f.Int64(prefix+".confirmation-blocks", DefaultL1ValidatorConfig.ConfirmationBlocks, "confirmation blocks")
	DangerousConfigAddOptions(prefix+".dangerous", f)
}

type DangerousConfig struct {
	WithoutBlockValidator bool `koanf:"without-block-validator"`
}

var DefaultDangerousConfig = DangerousConfig{
	WithoutBlockValidator: false,
}

func DangerousConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".without-block-validator", DefaultL1ValidatorConfig.Dangerous.WithoutBlockValidator, "DANGEROUS! allows running an L1 validator without a block validator")
}

type nodeAndHash struct {
	id   uint64
	hash common.Hash
}

type Staker struct {
	*L1Validator
	util.StopWaiter
	activeChallenge         *ChallengeManager
	strategy                StakerStrategy
	baseCallOpts            bind.CallOpts
	config                  L1ValidatorConfig
	highGasBlocksBuffer     *big.Int
	lastActCalledBlock      *big.Int
	inactiveLastCheckedNode *nodeAndHash
	bringActiveUntilNode    uint64
	withdrawDestination     common.Address
	inboxReader             InboxReaderInterface
}

func stakerStrategyFromString(s string) (StakerStrategy, error) {
	if strings.ToLower(s) == "watchtower" {
		return WatchtowerStrategy, nil
	} else if strings.ToLower(s) == "defensive" {
		return DefensiveStrategy, nil
	} else if strings.ToLower(s) == "stakelatest" {
		return StakeLatestStrategy, nil
	} else if strings.ToLower(s) == "makenodes" {
		return MakeNodesStrategy, nil
	} else {
		return WatchtowerStrategy, fmt.Errorf("unknown staker strategy \"%v\"", s)
	}
}

func NewStaker(
	client arbutil.L1Interface,
	wallet *ValidatorWallet,
	callOpts bind.CallOpts,
	config L1ValidatorConfig,
	l2Blockchain *core.BlockChain,
	inboxReader InboxReaderInterface,
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
	validatorUtilsAddress common.Address,
) (*Staker, error) {
	strategy, err := stakerStrategyFromString(config.Strategy)
	if err != nil {
		return nil, err
	}
	val, err := NewL1Validator(client, wallet, validatorUtilsAddress, callOpts, l2Blockchain, inboxTracker, txStreamer, blockValidator)
	if err != nil {
		return nil, err
	}
	withdrawDestination := wallet.From()
	if common.IsHexAddress(config.WithdrawDestination) {
		withdrawDestination = common.HexToAddress(config.WithdrawDestination)
	}
	return &Staker{
		L1Validator:         val,
		strategy:            strategy,
		baseCallOpts:        callOpts,
		config:              config,
		highGasBlocksBuffer: big.NewInt(config.L1PostingStrategy.HighGasDelayBlocks),
		lastActCalledBlock:  nil,
		withdrawDestination: withdrawDestination,
		inboxReader:         inboxReader,
	}, nil
}

func (s *Staker) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn)
	backoff := time.Second
	s.CallIteratively(func(ctx context.Context) time.Duration {
		arbTx, err := s.Act(ctx)
		if err == nil && arbTx != nil {
			_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, s.client, arbTx, txTimeout)
			err = errors.Wrap(err, "error waiting for tx receipt")
			if err == nil {
				log.Info("successfully executed staker transaction", "hash", arbTx.Hash())
			}
		}
		if err == nil {
			backoff = time.Second
			return s.config.StakerInterval
		}
		log.Warn("error acting as staker", "err", err)
		if backoff < 60*time.Second {
			backoff *= 2
		}
		return backoff
	})
}

func (s *Staker) shouldAct(ctx context.Context) bool {
	var gasPriceHigh = false
	var gasPriceFloat float64
	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		log.Warn("error getting gas price", "err", err)
	} else {
		gasPriceFloat = float64(gasPrice.Int64()) / 1e9
		if gasPriceFloat >= s.config.L1PostingStrategy.HighGasThreshold {
			gasPriceHigh = true
		}
	}
	latestBlockInfo, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Warn("error getting latest block", "err", err)
		return true
	}
	latestBlockNum := latestBlockInfo.Number
	if s.lastActCalledBlock == nil {
		s.lastActCalledBlock = latestBlockNum
	}
	blocksSinceActCalled := new(big.Int).Sub(latestBlockNum, s.lastActCalledBlock)
	s.lastActCalledBlock = latestBlockNum
	if gasPriceHigh {
		// We're eating into the high gas buffer to delay our tx
		s.highGasBlocksBuffer.Sub(s.highGasBlocksBuffer, blocksSinceActCalled)
	} else {
		// We'll try to make a tx if necessary, so we can add to the buffer for future high gas
		s.highGasBlocksBuffer.Add(s.highGasBlocksBuffer, blocksSinceActCalled)
	}
	// Clamp `s.highGasBlocksBuffer` to between 0 and HighGasDelayBlocks
	if s.highGasBlocksBuffer.Sign() < 0 {
		s.highGasBlocksBuffer.SetInt64(0)
	} else if s.highGasBlocksBuffer.Cmp(big.NewInt(s.config.L1PostingStrategy.HighGasDelayBlocks)) > 0 {
		s.highGasBlocksBuffer.SetInt64(s.config.L1PostingStrategy.HighGasDelayBlocks)
	}
	if gasPriceHigh && s.highGasBlocksBuffer.Sign() > 0 {
		log.Warn(
			"not acting yet as gas price is high",
			"gasPrice", gasPriceFloat,
			"highGasPriceConfig", s.config.L1PostingStrategy.HighGasThreshold,
			"highGasBuffer", s.highGasBlocksBuffer,
		)
		return false
	} else {
		return true
	}
}

func (s *Staker) Act(ctx context.Context) (*types.Transaction, error) {
	if !s.shouldAct(ctx) {
		// The fact that we're delaying acting is alreay logged in `shouldAct`
		return nil, nil
	}
	callOpts := s.getCallOpts(ctx)
	s.builder.ClearTransactions()
	var rawInfo *StakerInfo
	walletAddress := s.wallet.Address()
	var walletAddressOrZero common.Address
	if walletAddress != nil {
		walletAddressOrZero = *walletAddress
	}
	if walletAddress != nil {
		var err error
		rawInfo, err = s.rollup.StakerInfo(ctx, walletAddressOrZero)
		if err != nil {
			return nil, err
		}
	}
	// If the wallet address is zero, or the wallet address isn't staked,
	// this will return the latest node and its hash (atomically).
	latestStakedNodeNum, latestStakedNodeInfo, err := s.validatorUtils.LatestStaked(callOpts, s.rollupAddress, walletAddressOrZero)
	if err != nil {
		return nil, err
	}
	if rawInfo != nil {
		rawInfo.LatestStakedNode = latestStakedNodeNum
	}
	info := OurStakerInfo{
		CanProgress:          true,
		LatestStakedNode:     latestStakedNodeNum,
		LatestStakedNodeHash: latestStakedNodeInfo.NodeHash,
		StakerInfo:           rawInfo,
		StakeExists:          rawInfo != nil,
	}

	effectiveStrategy := s.strategy
	nodesLinear, err := s.validatorUtils.AreUnresolvedNodesLinear(callOpts, s.rollupAddress)
	if err != nil {
		return nil, err
	}
	if !nodesLinear {
		log.Warn("rollup assertion fork detected")
		if effectiveStrategy == DefensiveStrategy {
			effectiveStrategy = StakeLatestStrategy
		}
		s.inactiveLastCheckedNode = nil
	}
	if s.bringActiveUntilNode != 0 {
		if info.LatestStakedNode < s.bringActiveUntilNode {
			if effectiveStrategy == DefensiveStrategy {
				effectiveStrategy = StakeLatestStrategy
			}
		} else {
			log.Info("defensive validator staked past incorrect node; waiting here")
			s.bringActiveUntilNode = 0
		}
		s.inactiveLastCheckedNode = nil
	}
	if effectiveStrategy <= DefensiveStrategy && s.inactiveLastCheckedNode != nil {
		info.LatestStakedNode = s.inactiveLastCheckedNode.id
		info.LatestStakedNodeHash = s.inactiveLastCheckedNode.hash
	}

	latestConfirmedNode, err := s.rollup.LatestConfirmed(callOpts)
	if err != nil {
		return nil, err
	}

	// If we have an old stake, remove it
	if rawInfo != nil && rawInfo.LatestStakedNode <= latestConfirmedNode {
		// At this point it'd be better to just create a new stake on the latest confirmed node
		stakeIsTooOutdated := rawInfo.LatestStakedNode < latestConfirmedNode
		// We're not trying to stake anyways
		stakeIsUnwanted := effectiveStrategy < StakeLatestStrategy
		if stakeIsTooOutdated || stakeIsUnwanted {
			// Note: we must have an address if rawInfo != nil
			_, err = s.rollup.ReturnOldDeposit(s.builder.Auth(ctx), *walletAddress)
			if err != nil {
				return nil, err
			}
			if stakeIsUnwanted {
				_, err = s.rollup.WithdrawStakerFunds(s.builder.Auth(ctx), s.withdrawDestination)
				if err != nil {
					return nil, err
				}
				log.Info("removing old stake and withdrawing funds")
			} else {
				log.Info("removing old stake to re-place stake on latest confirmed node")
			}
			return s.wallet.ExecuteTransactions(ctx, s.builder)
		}
	}

	// Resolve nodes if either we're on the make nodes strategy,
	// or we're on the stake latest strategy but don't have a stake
	// (attempt to reduce the current required stake).
	shouldResolveNodes := effectiveStrategy >= MakeNodesStrategy
	if !shouldResolveNodes && effectiveStrategy >= StakeLatestStrategy && rawInfo == nil {
		shouldResolveNodes, err = s.isRequiredStakeElevated(ctx)
		if err != nil {
			return nil, err
		}
	}
	resolvingNode := false
	if shouldResolveNodes {
		arbTx, err := s.resolveTimedOutChallenges(ctx)
		if err != nil || arbTx != nil {
			return arbTx, err
		}
		resolvingNode, err = s.resolveNextNode(ctx, rawInfo)
		if err != nil {
			return nil, err
		}
	}

	if walletAddress != nil {
		withdrawable, err := s.rollup.WithdrawableFunds(callOpts, *walletAddress)
		if err != nil {
			return nil, err
		}
		if withdrawable.Sign() > 0 && s.withdrawDestination != (common.Address{}) {
			_, err = s.rollup.WithdrawStakerFunds(s.builder.Auth(ctx), s.withdrawDestination)
			if err != nil {
				return nil, err
			}
		}
	}

	if rawInfo != nil {
		if err = s.handleConflict(ctx, rawInfo); err != nil {
			return nil, err
		}
	}

	// Don't attempt to create a new stake if we're resolving a node,
	// as that might affect the current required stake.
	if rawInfo != nil || !resolvingNode {
		// Advance stake up to 20 times in one transaction
		for i := 0; info.CanProgress && i < 20; i++ {
			if err := s.advanceStake(ctx, &info, effectiveStrategy); err != nil {
				return nil, err
			}
		}
	}

	if rawInfo != nil && s.builder.BuildingTransactionCount() == 0 {
		if err := s.createConflict(ctx, rawInfo); err != nil {
			return nil, err
		}
	}

	if s.builder.BuildingTransactionCount() == 0 {
		return nil, nil
	}

	if info.StakerInfo == nil && info.StakeExists {
		log.Info("staking to execute transactions")
	}
	return s.wallet.ExecuteTransactions(ctx, s.builder)
}

func (s *Staker) handleConflict(ctx context.Context, info *StakerInfo) error {
	if info.CurrentChallenge == nil {
		s.activeChallenge = nil
		return nil
	}

	if s.activeChallenge == nil || s.activeChallenge.ChallengeIndex() != *info.CurrentChallenge {
		log.Warn("entered challenge", "challenge", info.CurrentChallenge)

		latestConfirmedCreated, err := s.rollup.LatestConfirmedCreationBlock(ctx)
		if err != nil {
			return err
		}

		newChallengeManager, err := NewChallengeManager(
			ctx,
			s.builder,
			s.builder.builderAuth,
			*s.builder.wallet.Address(),
			s.challengeManagerAddress,
			*info.CurrentChallenge,
			s.l2Blockchain,
			s.inboxReader,
			s.inboxTracker,
			s.txStreamer,
			s.blockValidator.MachineLoader,
			latestConfirmedCreated,
			s.config.TargetMachineCount,
			s.config.ConfirmationBlocks,
		)
		if err != nil {
			return err
		}

		s.activeChallenge = newChallengeManager
	}

	_, err := s.activeChallenge.Act(ctx)
	return err
}

func (s *Staker) advanceStake(ctx context.Context, info *OurStakerInfo, effectiveStrategy StakerStrategy) error {
	active := effectiveStrategy >= StakeLatestStrategy
	action, wrongNodesExist, err := s.generateNodeAction(ctx, info, effectiveStrategy)
	if err != nil {
		return err
	}
	if wrongNodesExist && effectiveStrategy == WatchtowerStrategy {
		log.Error("found incorrect assertion in watchtower mode")
	}
	if action == nil {
		info.CanProgress = false
		return nil
	}

	switch action := action.(type) {
	case createNodeAction:
		if wrongNodesExist && s.config.DisableChallenge {
			log.Error("refusing to challenge assertion as config disables challenges")
			info.CanProgress = false
			return nil
		}
		if !active {
			if wrongNodesExist && effectiveStrategy >= DefensiveStrategy {
				log.Warn("bringing defensive validator online because of incorrect assertion")
				s.bringActiveUntilNode = info.LatestStakedNode + 1
			}
			info.CanProgress = false
			return nil
		}
		// Details are already logged with more details in generateNodeAction
		info.CanProgress = false
		info.LatestStakedNode = 0
		info.LatestStakedNodeHash = action.hash

		// We'll return early if we already havea stake
		if info.StakeExists {
			_, err = s.rollup.StakeOnNewNode(s.builder.Auth(ctx), action.assertion.AsSolidityStruct(), action.hash, action.prevInboxMaxCount)
			return err
		}

		// If we have no stake yet, we'll put one down
		stakeAmount, err := s.rollup.CurrentRequiredStake(s.getCallOpts(ctx))
		if err != nil {
			return err
		}
		_, err = s.rollup.NewStakeOnNewNode(
			s.builder.AuthWithAmount(ctx, stakeAmount),
			action.assertion.AsSolidityStruct(),
			action.hash,
			action.prevInboxMaxCount,
		)
		if err != nil {
			return err
		}
		info.StakeExists = true
		return nil
	case existingNodeAction:
		info.LatestStakedNode = action.number
		info.LatestStakedNodeHash = action.hash
		if !active {
			if wrongNodesExist && effectiveStrategy >= DefensiveStrategy {
				log.Warn("bringing defensive validator online because of incorrect assertion")
				s.bringActiveUntilNode = action.number
				info.CanProgress = false
			} else {
				s.inactiveLastCheckedNode = &nodeAndHash{
					id:   action.number,
					hash: action.hash,
				}
			}
			return nil
		}
		log.Info("staking on existing node", "node", action.number)
		// We'll return early if we already havea stake
		if info.StakeExists {
			_, err = s.rollup.StakeOnExistingNode(s.builder.Auth(ctx), action.number, action.hash)
			return err
		}

		// If we have no stake yet, we'll put one down
		stakeAmount, err := s.rollup.CurrentRequiredStake(s.getCallOpts(ctx))
		if err != nil {
			return err
		}
		_, err = s.rollup.NewStakeOnExistingNode(
			s.builder.AuthWithAmount(ctx, stakeAmount),
			action.number,
			action.hash,
		)
		if err != nil {
			return err
		}
		info.StakeExists = true
		return nil
	default:
		panic("invalid action type")
	}
}

func (s *Staker) createConflict(ctx context.Context, info *StakerInfo) error {
	if info.CurrentChallenge != nil {
		return nil
	}

	callOpts := s.getCallOpts(ctx)
	stakers, moreStakers, err := s.validatorUtils.GetStakers(callOpts, s.rollupAddress, 0, 1024)
	if err != nil {
		return err
	}
	for moreStakers {
		var newStakers []common.Address
		newStakers, moreStakers, err = s.validatorUtils.GetStakers(callOpts, s.rollupAddress, uint64(len(stakers)), 1024)
		if err != nil {
			return err
		}
		stakers = append(stakers, newStakers...)
	}
	latestNode, err := s.rollup.LatestConfirmed(callOpts)
	if err != nil {
		return err
	}
	// Safe to dereference as createConflict is only called when we have a wallet address
	walletAddr := *s.wallet.Address()
	for _, staker := range stakers {
		stakerInfo, err := s.rollup.StakerInfo(ctx, staker)
		if err != nil {
			return err
		}
		if stakerInfo.CurrentChallenge != nil {
			continue
		}
		conflictInfo, err := s.validatorUtils.FindStakerConflict(callOpts, s.rollupAddress, walletAddr, staker, big.NewInt(1024))
		if err != nil {
			return err
		}
		if ConflictType(conflictInfo.Ty) != CONFLICT_TYPE_FOUND {
			continue
		}
		staker1 := walletAddr
		staker2 := staker
		if conflictInfo.Node2 < conflictInfo.Node1 {
			staker1, staker2 = staker2, staker1
			conflictInfo.Node1, conflictInfo.Node2 = conflictInfo.Node2, conflictInfo.Node1
		}
		if conflictInfo.Node1 <= latestNode {
			// Immaterial as this is past the confirmation point; this must be a zombie
			continue
		}

		node1Info, err := s.rollup.LookupNode(ctx, conflictInfo.Node1)
		if err != nil {
			return err
		}
		node2Info, err := s.rollup.LookupNode(ctx, conflictInfo.Node2)
		if err != nil {
			return err
		}
		log.Warn("creating challenge", "node1", conflictInfo.Node1, "node2", conflictInfo.Node2, "otherStaker", staker2)
		_, err = s.rollup.CreateChallenge(
			s.builder.Auth(ctx),
			[2]common.Address{staker1, staker2},
			[2]uint64{conflictInfo.Node1, conflictInfo.Node2},
			node1Info.MachineStatuses(),
			node1Info.GlobalStates(),
			node1Info.Assertion.NumBlocks,
			node2Info.Assertion.ExecutionHash(),
			[2]*big.Int{new(big.Int).SetUint64(node1Info.BlockProposed), new(big.Int).SetUint64(node2Info.BlockProposed)},
			[2][32]byte{node1Info.WasmModuleRoot, node2Info.WasmModuleRoot},
		)
		if err != nil {
			return err
		}
	}
	// No conflicts exist
	return nil
}
