// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package endtoend

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/testing"
	"github.com/offchainlabs/nitro/bold/testing/endtoend/backend"
	"github.com/offchainlabs/nitro/bold/testing/mocks/state-provider"
	"github.com/offchainlabs/nitro/bold/testing/setup"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
)

type backendKind uint8

const (
	simulated backendKind = iota
	anvil
)

func (b backendKind) String() string {
	switch b {
	case simulated:
		return "simulated"
	case anvil:
		return "anvil"
	default:
		return "unknown"
	}
}

// Defines the configuration for an end-to-end test, with different
// parameters for the various parts of the system.
type e2eConfig struct {
	backend      backendKind
	protocol     protocolParams
	timings      timeParams
	inbox        inboxParams
	actors       actorParams
	expectations []expect
}

// Defines parameters related to the actors participating in the test.
type actorParams struct {
	numEvilValidators uint64
}

// Configures intervals related to timings in the system.
type timeParams struct {
	blockTime                            time.Duration
	assertionPostingInterval             time.Duration
	assertionScanningInterval            time.Duration
	assertionConfirmationAttemptInterval time.Duration
}

func defaultTimeParams() timeParams {
	return timeParams{
		// Fast block time.
		blockTime: time.Second,
		// Go very fast.
		assertionPostingInterval:             time.Hour,
		assertionScanningInterval:            time.Second,
		assertionConfirmationAttemptInterval: time.Second,
	}
}

// Configures info about the state of the Arbitrum Inbox when a test runs,
// useful to set up things such as the number of batches posted.
type inboxParams struct {
	numBatchesPosted uint64
}

func defaultInboxParams() inboxParams {
	return inboxParams{
		numBatchesPosted: 5,
	}
}

// Defines constants and other parameters related to the protocol itself,
// such as the number of challenge levels or the confirmation period.
type protocolParams struct {
	numBigStepLevels      uint8
	challengePeriodBlocks uint64
	layerZeroHeights      protocol.LayerZeroHeights
}

func defaultProtocolParams() protocolParams {
	return protocolParams{
		numBigStepLevels:      1,
		challengePeriodBlocks: 25,
		layerZeroHeights: protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 4,
			BigStepChallengeHeight:   1 << 4,
			SmallStepChallengeHeight: 1 << 4,
		},
	}
}

func TestEndToEnd_SmokeTest(t *testing.T) {
	timeCfg := defaultTimeParams()
	timeCfg.blockTime = time.Second
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: defaultProtocolParams(),
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 1,
		},
		timings: timeCfg,
		expectations: []expect{
			expectChallengeWinWithAllHonestEssentialEdgesConfirmed,
		},
	})
}

func TestEndToEnd_TwoEvilValidators(t *testing.T) {
	protocolCfg := defaultProtocolParams()
	timeCfg := defaultTimeParams()
	timeCfg.assertionPostingInterval = time.Hour
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: protocolCfg,
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 2,
		},
		timings: timeCfg,
		expectations: []expect{
			expectChallengeWinWithAllHonestEssentialEdgesConfirmed,
		},
	})
}

func TestEndToEnd_ManyEvilValidators(t *testing.T) {
	protocolCfg := defaultProtocolParams()
	timeCfg := defaultTimeParams()
	timeCfg.assertionPostingInterval = time.Hour
	protocolCfg.challengePeriodBlocks = 50
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: protocolCfg,
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 5,
		},
		timings: timeCfg,
		expectations: []expect{
			expectChallengeWinWithAllHonestEssentialEdgesConfirmed,
		},
	})
}

func runEndToEndTest(t *testing.T, cfg *e2eConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Validators include a chain admin, a single honest validator, and any
	// number of evil entities.
	totalValidators := cfg.actors.numEvilValidators + 2

	challengeTestingOpts := []challenge_testing.Opt{
		challenge_testing.WithConfirmPeriodBlocks(cfg.protocol.challengePeriodBlocks),
		challenge_testing.WithLayerZeroHeights(&cfg.protocol.layerZeroHeights),
		challenge_testing.WithNumBigStepLevels(cfg.protocol.numBigStepLevels),
	}
	deployOpts := []setup.Opt{
		setup.WithMockBridge(),
		setup.WithMockOneStepProver(),
		setup.WithNumAccounts(totalValidators),
		setup.WithChallengeTestingOpts(challengeTestingOpts...),
	}

	var bk backend.Backend
	switch cfg.backend {
	case simulated:
		simBackend, err := backend.NewSimulated(cfg.timings.blockTime, deployOpts...)
		require.NoError(t, err)
		bk = simBackend
	case anvil:
		anvilBackend, err := backend.NewAnvilLocal(ctx)
		require.NoError(t, err)
		bk = anvilBackend
	default:
		t.Fatalf("Backend kind for e2e test not supported: %s", cfg.backend)
	}

	require.NoError(t, bk.Start(ctx))

	rollupAddr, err := bk.DeployRollup(ctx, challengeTestingOpts...)
	require.NoError(t, err)

	accounts := bk.Accounts()
	bk.Commit()

	rollupUserBindings, err := rollupgen.NewRollupUserLogic(rollupAddr.Rollup, bk.Client())
	require.NoError(t, err)
	bridgeAddr, err := rollupUserBindings.Bridge(&bind.CallOpts{})
	require.NoError(t, err)
	dataHash := common.Hash{1}
	enqueueSequencerMessageAsExecutor(
		t, accounts[0], rollupAddr.UpgradeExecutor, bk.Client(), bridgeAddr, seqMessage{
			dataHash:                 dataHash,
			afterDelayedMessagesRead: big.NewInt(1),
			prevMessageCount:         big.NewInt(1),
			newMessageCount:          big.NewInt(2),
		},
	)

	baseStateManagerOpts := []stateprovider.Opt{
		stateprovider.WithNumBatchesRead(cfg.inbox.numBatchesPosted),
		stateprovider.WithLayerZeroHeights(&cfg.protocol.layerZeroHeights, cfg.protocol.numBigStepLevels),
	}
	honestStateManager, err := stateprovider.NewForSimpleMachine(t, baseStateManagerOpts...)
	require.NoError(t, err)

	shp := &simpleHeaderProvider{b: bk, chs: make([]chan<- *gethtypes.Header, 0)}
	shp.Start(ctx)

	baseStackOpts := []challengemanager.StackOpt{
		challengemanager.StackWithMode(types.MakeMode),
		challengemanager.StackWithPollingInterval(cfg.timings.assertionScanningInterval),
		challengemanager.StackWithPostingInterval(cfg.timings.assertionPostingInterval),
		challengemanager.StackWithAverageBlockCreationTime(cfg.timings.blockTime),
		challengemanager.StackWithConfirmationInterval(cfg.timings.assertionConfirmationAttemptInterval),
		challengemanager.StackWithMinimumGapToParentAssertion(0),
		challengemanager.StackWithHeaderProvider(shp),
	}

	name := "honest"
	txOpts := accounts[1]
	//nolint:gocritic
	honestOpts := append(
		baseStackOpts,
		challengemanager.StackWithName(name),
	)
	honestChain := setupAssertionChain(t, ctx, bk.Client(), rollupAddr.Rollup, txOpts)
	honestManager, err := challengemanager.NewChallengeStack(honestChain, honestStateManager, honestOpts...)
	require.NoError(t, err)

	totalOpcodes := totalWasmOpcodes(&cfg.protocol.layerZeroHeights, cfg.protocol.numBigStepLevels)
	t.Logf("Total wasm opcodes in test: %d", totalOpcodes)

	assertionDivergenceHeight := uint64(1)
	assertionBlockHeightDifference := int64(1)

	evilChallengeManagers := make([]*challengemanager.Manager, cfg.actors.numEvilValidators)
	for i := uint64(0); i < cfg.actors.numEvilValidators; i++ {
		// Diverge at a random opcode within the block.
		machineDivergenceStep := randUint64(i)
		if machineDivergenceStep == 0 {
			machineDivergenceStep = 1
		}
		//nolint:gocritic
		evilStateManagerOpts := append(
			baseStateManagerOpts,
			stateprovider.WithMachineDivergenceStep(machineDivergenceStep),
			stateprovider.WithBlockDivergenceHeight(assertionDivergenceHeight),
			stateprovider.WithDivergentBlockHeightOffset(assertionBlockHeightDifference),
		)
		evilStateManager, err := stateprovider.NewForSimpleMachine(t, evilStateManagerOpts...)
		require.NoError(t, err)

		// Honest validator has index 1 in the accounts slice, as 0 is admin, so
		// evil ones should start at 2.
		evilTxOpts := accounts[2+i]
		name = fmt.Sprintf("evil-%d", i)
		//nolint:gocritic
		evilOpts := append(
			baseStackOpts,
			challengemanager.StackWithName(name),
		)
		evilChain := setupAssertionChain(t, ctx, bk.Client(), rollupAddr.Rollup, evilTxOpts)
		evilManager, err := challengemanager.NewChallengeStack(evilChain, evilStateManager, evilOpts...)
		require.NoError(t, err)
		evilChallengeManagers[i] = evilManager
	}

	honestManager.Start(ctx)

	for _, evilManager := range evilChallengeManagers {
		evilManager.Start(ctx)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, e := range cfg.expectations {
		fn := e // loop closure
		g.Go(func() error {
			return fn(t, ctx, bk.ContractAddresses(), bk.Client(), txOpts.From)
		})
	}
	require.NoError(t, g.Wait())
}

type seqMessage struct {
	dataHash                 common.Hash
	afterDelayedMessagesRead *big.Int
	prevMessageCount         *big.Int
	newMessageCount          *big.Int
}

type committer interface {
	Commit() common.Hash
}

func enqueueSequencerMessageAsExecutor(
	t *testing.T,
	opts *bind.TransactOpts,
	executor common.Address,
	backend protocol.ChainBackend,
	bridge common.Address,
	msg seqMessage,
) {
	execBindings, err := mocksgen.NewUpgradeExecutorMock(executor, backend)
	require.NoError(t, err)
	seqInboxABI, err := abi.JSON(strings.NewReader(bridgegen.AbsBridgeABI))
	require.NoError(t, err)

	data, err := seqInboxABI.Pack(
		"setSequencerInbox",
		executor,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, data)
	require.NoError(t, err)
	if comm, ok := backend.(committer); ok {
		comm.Commit()
	}

	seqQueueMsg, err := seqInboxABI.Pack(
		"enqueueSequencerMessage",
		msg.dataHash, msg.afterDelayedMessagesRead, msg.prevMessageCount, msg.newMessageCount,
	)
	require.NoError(t, err)
	_, err = execBindings.ExecuteCall(opts, bridge, seqQueueMsg)
	require.NoError(t, err)
	if comm, ok := backend.(committer); ok {
		comm.Commit()
	}
}
