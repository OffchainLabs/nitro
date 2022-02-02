//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/pkg/errors"
)

const txTimeout time.Duration = 5 * time.Minute

type StakerStrategy uint8

const (
	WatchtowerStrategy StakerStrategy = iota
	DefensiveStrategy
	StakeLatestStrategy
	MakeNodesStrategy
)

type L1PostingStrategy struct {
	HighGasThreshold   float64
	HighGasDelayBlocks int64
}

type ValidatorConfig struct {
	Strategy             string
	UtilsAddress         string
	StakerDelay          time.Duration
	WalletFactoryAddress string
	L1PostingStrategy    L1PostingStrategy
	DontChallenge        bool
	WithdrawDestination  string
	TargetNumMachines    int
}

type nodeAndHash struct {
	id   uint64
	hash common.Hash
}

type Staker struct {
	*Validator
	activeChallenge         *ChallengeManager
	strategy                StakerStrategy
	fromBlock               int64
	baseCallOpts            bind.CallOpts
	auth                    *bind.TransactOpts
	config                  ValidatorConfig
	highGasBlocksBuffer     *big.Int
	lastActCalledBlock      *big.Int
	inactiveLastCheckedNode *nodeAndHash
	bringActiveUntilNode    uint64
	withdrawDestination     common.Address

	l2Blockchain *core.BlockChain
	inboxReader  InboxReaderInterface
	inboxTracker InboxTrackerInterface
	txStreamer   TransactionStreamerInterface
}

func NewStaker(
	ctx context.Context,
	client *ethclient.Client,
	wallet *ValidatorWallet,
	fromBlock int64,
	validatorUtilsAddress common.Address,
	strategy StakerStrategy,
	callOpts bind.CallOpts,
	auth *bind.TransactOpts,
	config ValidatorConfig,
	l2Blockchain *core.BlockChain,
	inboxReader InboxReaderInterface,
	inboxTracker InboxTrackerInterface,
	txStreamer TransactionStreamerInterface,
	blockValidator *BlockValidator,
) (*Staker, error) {
	val, err := NewValidator(ctx, client, wallet, fromBlock, validatorUtilsAddress, callOpts, l2Blockchain, inboxReader, inboxTracker, txStreamer, blockValidator)
	if err != nil {
		return nil, err
	}
	withdrawDestination := wallet.From()
	if common.IsHexAddress(config.WithdrawDestination) {
		withdrawDestination = common.HexToAddress(config.WithdrawDestination)
	}
	return &Staker{
		Validator:           val,
		strategy:            strategy,
		fromBlock:           fromBlock,
		baseCallOpts:        callOpts,
		auth:                auth,
		config:              config,
		highGasBlocksBuffer: big.NewInt(config.L1PostingStrategy.HighGasDelayBlocks),
		lastActCalledBlock:  nil,
		withdrawDestination: withdrawDestination,

		l2Blockchain: l2Blockchain,
		inboxReader:  inboxReader,
		inboxTracker: inboxTracker,
		txStreamer:   txStreamer,
	}, nil
}

func (s *Staker) RunInBackground(ctx context.Context, stakerDelay time.Duration) chan bool {
	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		backoff := time.Second
		for {
			arbTx, err := s.Act(ctx)
			if err == nil && arbTx != nil {
				// Note: methodName isn't accurate, it's just used for logging
				_, err = arbutil.EnsureTxSucceededWithTimeout(ctx, s.client, arbTx, txTimeout)
				err = errors.Wrap(err, "error waiting for tx receipt")
				if err == nil {
					log.Info("successfully executed staker transaction", "hash", arbTx.Hash())
				}
			}
			if err != nil {
				log.Warn("error acting as staker", "err", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 60*time.Second {
					backoff *= 2
				}
				continue
			} else {
				backoff = time.Second
			}
			time.Sleep(stakerDelay)
		}
	}()
	return done
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
		// We'll make a tx if necessary, so we can add to the buffer for future high gas
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
	latestStakedNode, latestStakedNodeHash, err := s.validatorUtils.LatestStaked(callOpts, s.rollupAddress, walletAddressOrZero)
	if err != nil {
		return nil, err
	}
	if rawInfo != nil {
		rawInfo.LatestStakedNode = latestStakedNode
	}
	info := OurStakerInfo{
		CanProgress:          true,
		LatestStakedNode:     latestStakedNode,
		LatestStakedNodeHash: latestStakedNodeHash,
		StakerInfo:           rawInfo,
	}

	effectiveStrategy := s.strategy
	nodesLinear, err := s.validatorUtils.AreUnresolvedNodesLinear(callOpts, s.rollupAddress)
	if err != nil {
		return nil, err
	}
	if !nodesLinear {
		log.Warn("fork detected")
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
	if shouldResolveNodes {
		// Keep the stake of this validator placed if we plan on staking further
		arbTx, err := s.removeOldStakers(ctx, effectiveStrategy >= StakeLatestStrategy)
		if err != nil || arbTx != nil {
			return arbTx, err
		}
		arbTx, err = s.resolveTimedOutChallenges(ctx)
		if err != nil || arbTx != nil {
			return arbTx, err
		}
		if err := s.resolveNextNode(ctx, rawInfo, s.fromBlock); err != nil {
			return nil, err
		}
	}

	addr := s.wallet.Address()
	if addr != nil {
		withdrawable, err := s.rollup.WithdrawableFunds(callOpts, *addr)
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

	// Don't attempt to create a new stake if we're resolving a node,
	// as that might affect the current required stake.
	creatingNewStake := rawInfo == nil && s.builder.BuilderTransactionCount() == 0
	if creatingNewStake {
		if err := s.newStake(ctx); err != nil {
			return nil, err
		}
	}

	if rawInfo != nil {
		if err = s.handleConflict(ctx, rawInfo); err != nil {
			return nil, err
		}
	}
	if rawInfo != nil || creatingNewStake {
		// Advance stake up to 20 times in one transaction
		for i := 0; info.CanProgress && i < 20; i++ {
			if err := s.advanceStake(ctx, &info, effectiveStrategy); err != nil {
				return nil, err
			}
		}
	}
	if rawInfo != nil && s.builder.BuilderTransactionCount() == 0 {
		if err := s.createConflict(ctx, rawInfo); err != nil {
			return nil, err
		}
	}

	txCount := s.builder.BuilderTransactionCount()
	if creatingNewStake {
		// Ignore our stake creation, as it's useless by itself
		txCount--
	}
	if txCount == 0 {
		return nil, nil
	}
	if creatingNewStake {
		log.Info("staking to execute transactions")
	}
	return s.wallet.ExecuteTransactions(ctx, s.builder)
}

func (s *Staker) handleConflict(ctx context.Context, info *StakerInfo) error {
	if info.CurrentChallenge == nil {
		s.activeChallenge = nil
		return nil
	}

	if s.activeChallenge == nil || s.activeChallenge.RootChallengeAddress() != *info.CurrentChallenge {
		log.Warn("entered challenge", "challenge", info.CurrentChallenge)

		newChallengeManager, err := NewChallengeManager(ctx, s.client, s.auth, *info.CurrentChallenge, s.l2Blockchain, s.inboxReader, s.inboxTracker, s.txStreamer, uint64(s.fromBlock), s.config.TargetNumMachines)
		if err != nil {
			return err
		}

		s.activeChallenge = newChallengeManager
	}

	_, err := s.activeChallenge.Act(ctx)
	return err
}

func (s *Staker) newStake(ctx context.Context) error {
	var addr = s.wallet.Address()
	if addr != nil {
		info, err := s.rollup.StakerInfo(ctx, *addr)
		if err != nil {
			return err
		}
		if info != nil {
			return nil
		}
	}
	stakeAmount, err := s.rollup.CurrentRequiredStake(s.getCallOpts(ctx))
	if err != nil {
		return err
	}
	_, err = s.rollup.NewStake(s.builder.AuthWithAmount(ctx, stakeAmount))
	if err != nil {
		return err
	}
	return nil
}

func (s *Staker) advanceStake(ctx context.Context, info *OurStakerInfo, effectiveStrategy StakerStrategy) error {
	active := effectiveStrategy >= StakeLatestStrategy
	action, wrongNodesExist, err := s.generateNodeAction(ctx, info, effectiveStrategy, s.fromBlock)
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
		if wrongNodesExist && s.config.DontChallenge {
			log.Error("refusing to challenge assertion as config disables challenges")
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
		_, err = s.rollup.StakeOnNewNode(s.builder.Auth(ctx), action.hash, action.assertion.BytesFields(), action.assertion.IntFields(), action.prevInboxMaxCount, action.assertion.NumBlocks)
		return err
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
		_, err = s.rollup.StakeOnExistingNode(s.builder.Auth(ctx), action.number, action.hash)
		return err
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
		conflictType, node1, node2, err := s.validatorUtils.FindStakerConflict(callOpts, s.rollupAddress, walletAddr, staker, big.NewInt(1024))
		if err != nil {
			return err
		}
		if ConflictType(conflictType) != CONFLICT_TYPE_FOUND {
			continue
		}
		staker1 := walletAddr
		staker2 := staker
		if node2 < node1 {
			staker1, staker2 = staker2, staker1
			node1, node2 = node2, node1
		}
		if node1 <= latestNode {
			// removeOldStakers will take care of them
			continue
		}

		node1Info, err := s.rollup.LookupNode(ctx, node1)
		if err != nil {
			return err
		}
		node2Info, err := s.rollup.LookupNode(ctx, node2)
		if err != nil {
			return err
		}
		log.Warn("creating challenge", "ourNode", node1, "otherNode", node2, "otherStaker", staker2)
		_, err = s.rollup.CreateChallenge(
			s.builder.Auth(ctx),
			[2]common.Address{staker1, staker2},
			[2]uint64{node1, node2},
			[2][2]uint8{node1Info.MachineStatuses(), node2Info.MachineStatuses()},
			[2][2]rollupgen.GlobalState{node1Info.GlobalStates(), node2Info.GlobalStates()},
			[2]uint64{node1Info.Assertion.NumBlocks, node2Info.Assertion.NumBlocks},
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
