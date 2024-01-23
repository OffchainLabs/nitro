package endtoend

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	challengemanager "github.com/OffchainLabs/bold/challenge-manager"
	"github.com/OffchainLabs/bold/challenge-manager/types"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	challenge_testing "github.com/OffchainLabs/bold/testing"
	"github.com/OffchainLabs/bold/testing/endtoend/backend"
	statemanager "github.com/OffchainLabs/bold/testing/mocks/state-provider"
	"github.com/OffchainLabs/bold/testing/setup"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
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
// TODO: Support concurrent challenges at the assertion chain level.
// TODO: Many evil parties, each with their own claim.
// TODO: Many evil parties, all supporting the same claim.
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
	challengeMoveInterval                time.Duration
	assertionPostingInterval             time.Duration
	assertionScanningInterval            time.Duration
	assertionConfirmationAttemptInterval time.Duration
}

func defaultTimeParams() timeParams {
	return timeParams{
		// Fast block time.
		blockTime: time.Millisecond * 100,
		// Go very fast.
		challengeMoveInterval:     time.Millisecond * 250,
		assertionPostingInterval:  time.Second,
		assertionScanningInterval: time.Second,
		// An extremely high number so that we never attempt to confirm an assertion by time.
		// We instead expect the assertion to be confirmed by challenge win.
		assertionConfirmationAttemptInterval: time.Hour,
	}
}

// Configures info about the state of the Arbitrum Inbox
// when a test runs, useful to set up things such as the number of batches posted.
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
		challengePeriodBlocks: 150,
		layerZeroHeights: protocol.LayerZeroHeights{
			BlockChallengeHeight:     1 << 5,
			BigStepChallengeHeight:   1 << 5,
			SmallStepChallengeHeight: 1 << 5,
		},
	}
}

func TestEndToEnd_SmokeTest(t *testing.T) {
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: defaultProtocolParams(),
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 1,
		},
		timings: defaultTimeParams(),
		expectations: []expect{
			// Expect one assertion is confirmed by challenge win.
			expectAssertionConfirmedByChallengeWin,
			// Other ideas:
			// All validators are staked at top-level
			// All subchallenges have mini-stakes
		},
	})
}

func TestEndToEnd_MaxWavmOpcodes(t *testing.T) {
	protocolCfg := defaultProtocolParams()
	protocolCfg.numBigStepLevels = 3
	// A block can take a max of 2^43 wavm opcodes to validate.
	// With three big step levels of 2^10 each, and one small step level of
	// 2^13 (adding the exponents), we can have full coverage of a block dispute
	// in a challenge game.
	protocolCfg.layerZeroHeights = protocol.LayerZeroHeights{
		BlockChallengeHeight:     1 << 6,
		BigStepChallengeHeight:   1 << 10,
		SmallStepChallengeHeight: 1 << 13,
	}
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: protocolCfg,
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 1,
		},
		timings: defaultTimeParams(),
		expectations: []expect{
			// Expect one assertion is confirmed by challenge win.
			expectAssertionConfirmedByChallengeWin,
		},
	})
}

func TestEndToEnd_TwoEvilValidators(t *testing.T) {
	protocolCfg := defaultProtocolParams()
	timeCfg := defaultTimeParams()
	timeCfg.blockTime = time.Millisecond * 500
	timeCfg.challengeMoveInterval = time.Millisecond * 500
	timeCfg.assertionPostingInterval = time.Hour
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: protocolCfg,
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 2,
		},
		timings: defaultTimeParams(),
		expectations: []expect{
			// Expect one assertion is confirmed by challenge win.
			expectAssertionConfirmedByChallengeWin,
		},
	})
}

func TestEndToEnd_ManyEvilValidators(t *testing.T) {
	protocolCfg := defaultProtocolParams()
	protocolCfg.challengePeriodBlocks = 2000
	timeCfg := defaultTimeParams()
	timeCfg.blockTime = time.Millisecond * 500
	timeCfg.challengeMoveInterval = time.Millisecond * 500
	timeCfg.assertionPostingInterval = time.Hour
	runEndToEndTest(t, &e2eConfig{
		backend:  simulated,
		protocol: protocolCfg,
		inbox:    defaultInboxParams(),
		actors: actorParams{
			numEvilValidators: 5,
		},
		timings: defaultTimeParams(),
		expectations: []expect{
			// Expect one assertion is confirmed by challenge win.
			expectAssertionConfirmedByChallengeWin,
		},
	})
}

func runEndToEndTest(t *testing.T, cfg *e2eConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Validators include a chain admin, a single honest validators, and any number of evil entities.
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

	rollupAddr, err := bk.DeployRollup(ctx, challengeTestingOpts...)
	require.NoError(t, err)

	require.NoError(t, bk.Start(ctx))

	rollupAdminBindings, err := rollupgen.NewRollupAdminLogic(rollupAddr, bk.Client())
	require.NoError(t, err)
	accounts := bk.Accounts()
	_, err = rollupAdminBindings.SetMinimumAssertionPeriod(accounts[0], big.NewInt(1))
	require.NoError(t, err)

	bk.Commit()

	baseStateManagerOpts := []statemanager.Opt{
		statemanager.WithNumBatchesRead(cfg.inbox.numBatchesPosted),
		statemanager.WithLayerZeroHeights(&cfg.protocol.layerZeroHeights, cfg.protocol.numBigStepLevels),
	}
	honestStateManager, err := statemanager.NewForSimpleMachine(baseStateManagerOpts...)
	require.NoError(t, err)

	baseChallengeManagerOpts := []challengemanager.Opt{
		challengemanager.WithEdgeTrackerWakeInterval(cfg.timings.challengeMoveInterval),
		challengemanager.WithMode(types.MakeMode),
		challengemanager.WithAssertionPostingInterval(cfg.timings.assertionPostingInterval),
		challengemanager.WithAssertionScanningInterval(cfg.timings.assertionScanningInterval),
		challengemanager.WithAssertionConfirmingInterval(cfg.timings.assertionConfirmationAttemptInterval),
	}

	name := "honest"
	txOpts := accounts[1]
	honestOpts := append(
		baseChallengeManagerOpts,
		challengemanager.WithAddress(txOpts.From),
		challengemanager.WithName(name),
	)
	honestManager := setupChallengeManager(
		t, ctx, bk.Client(), rollupAddr, honestStateManager, txOpts, name, honestOpts...,
	)

	// Diverge exactly at the last opcode within the block.
	totalOpcodes := totalWasmOpcodes(&cfg.protocol.layerZeroHeights, cfg.protocol.numBigStepLevels)
	t.Logf("Total wasm opcodes in test: %d", totalOpcodes)

	assertionDivergenceHeight := uint64(1)
	assertionBlockHeightDifference := int64(1)

	evilChallengeManagers := make([]*challengemanager.Manager, cfg.actors.numEvilValidators)
	for i := uint64(0); i < cfg.actors.numEvilValidators; i++ {
		machineDivergenceStep := randUint64(totalOpcodes)
		evilStateManagerOpts := append(
			baseStateManagerOpts,
			statemanager.WithMachineDivergenceStep(machineDivergenceStep),
			statemanager.WithBlockDivergenceHeight(assertionDivergenceHeight),
			statemanager.WithDivergentBlockHeightOffset(assertionBlockHeightDifference),
		)
		evilStateManager, err := statemanager.NewForSimpleMachine(evilStateManagerOpts...)
		require.NoError(t, err)

		// Honest validator has index 1 in the accounts slice, as 0 is admin, so evil ones should start at 2.
		txOpts = accounts[2+i]
		name = fmt.Sprintf("evil-%d", i)
		evilOpts := append(
			baseChallengeManagerOpts,
			challengemanager.WithAddress(txOpts.From),
			challengemanager.WithName(name),
		)
		evilManager := setupChallengeManager(
			t, ctx, bk.Client(), rollupAddr, evilStateManager, txOpts, name, evilOpts...,
		)
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
			return fn(t, ctx, bk.ContractAddresses(), bk.Client())
		})
	}
	require.NoError(t, g.Wait())
}
