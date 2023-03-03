package validator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_onLeafCreation(t *testing.T) {
	ctx := context.Background()
	_ = ctx
	t.Run("no fork detected", func(t *testing.T) {
		logsHook := test.NewGlobal()
		v, _, s := setupValidator(t)

		parentSeqNum := protocol.AssertionSequenceNumber(1)
		seqNum := parentSeqNum + 1
		prev := &mocks.MockAssertion{
			MockPrevSeqNum: 0,
			MockSeqNum:     parentSeqNum,
			MockStateHash:  common.Hash{},
		}
		ev := &mocks.MockAssertion{
			MockPrevSeqNum: parentSeqNum,
			MockSeqNum:     seqNum,
			MockStateHash:  common.BytesToHash([]byte("bar")),
		}

		p := &mocks.MockProtocol{}
		s.On("HasStateCommitment", ctx, util.StateCommitment{}).Return(false)
		p.On("CurrentChallengeManager", ctx, &mocks.MockActiveTx{}).Return(&mocks.MockChallengeManager{}, nil)
		p.On("AssertionBySequenceNum", ctx, &mocks.MockActiveTx{}, prev.SeqNum()).Return(prev, nil)
		v.chain = p

		err := v.onLeafCreated(ctx, ev)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "No fork detected in assertion tree")
	})
	t.Run("fork leads validator to challenge leaf", func(t *testing.T) {
		logsHook := test.NewGlobal()
		ctx := context.Background()
		createdData := createTwoValidatorFork(t, ctx, &createForkConfig{
			divergeHeight: 10,
			numBlocks:     100,
		})

		manager := statemanager.New(createdData.honestValidatorStateRoots)

		validator, err := New(
			ctx,
			createdData.assertionChains[1],
			createdData.backend,
			manager,
			createdData.addrs.Rollup,
		)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, createdData.leaf1)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, createdData.leaf2)
		require.NoError(t, err)

		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

		err = validator.onLeafCreated(ctx, createdData.leaf2)
		require.ErrorContains(t, err, "Vertex already exists")
	})
}

func Test_onChallengeStarted(t *testing.T) {
	ctx := context.Background()
	logsHook := test.NewGlobal()

	createdData := createTwoValidatorFork(t, ctx, &createForkConfig{
		divergeHeight: 10,
		numBlocks:     100,
	})
	manager := statemanager.New(createdData.honestValidatorStateRoots)

	validator, err := New(
		ctx,
		createdData.assertionChains[1],
		createdData.backend,
		manager,
		createdData.addrs.Rollup,
	)
	require.NoError(t, err)

	err = validator.onLeafCreated(ctx, createdData.leaf1)
	require.NoError(t, err)

	err = validator.onLeafCreated(ctx, createdData.leaf2)
	require.NoError(t, err)

	AssertLogsContain(t, logsHook, "New assertion appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")
}

func Test_submitAndFetchProtocolChallenge(t *testing.T) {
	ctx := context.Background()
	createdData := createTwoValidatorFork(t, ctx, &createForkConfig{
		divergeHeight: 10,
		numBlocks:     100,
	})

	var genesis protocol.Assertion
	err := createdData.assertionChains[1].Call(func(tx protocol.ActiveTx) error {
		conf, err := createdData.assertionChains[1].LatestConfirmed(ctx, tx)
		if err != nil {
			return err
		}
		genesis = conf
		return nil
	})
	require.NoError(t, err)

	// Setup our mock state manager to agree on leaf1 but disagree on leaf2.
	manager := &mocks.MockStateManager{}
	validator, err := New(
		ctx,
		createdData.assertionChains[1],
		createdData.backend,
		manager,
		createdData.addrs.Rollup,
	)
	require.NoError(t, err)

	wantedChallenge, err := validator.submitProtocolChallenge(ctx, genesis.SeqNum())
	require.NoError(t, err)
	gotChallenge, err := validator.fetchProtocolChallenge(ctx, genesis.SeqNum())
	require.NoError(t, err)
	require.Equal(t, wantedChallenge, gotChallenge)
}

type createdValidatorFork struct {
	leaf1                     protocol.Assertion
	leaf2                     protocol.Assertion
	assertionChains           []*solimpl.AssertionChain
	accounts                  []*testAccount
	backend                   *backends.SimulatedBackend
	honestValidatorStateRoots []common.Hash
	evilValidatorStateRoots   []common.Hash
	addrs                     *rollupAddresses
}

type createForkConfig struct {
	numBlocks     uint64
	divergeHeight uint64
}

func createTwoValidatorFork(
	t *testing.T,
	ctx context.Context,
	cfg *createForkConfig,
) *createdValidatorFork {
	divergenceHeight := cfg.divergeHeight
	numBlocks := cfg.numBlocks

	chains, accs, addrs, backend := setupAssertionChains(t, 3)
	prevInboxMaxCount := big.NewInt(1)

	// Advance the backend by some blocks to get over time delta failures when
	// using the assertion chain.
	for i := 0; i < 100; i++ {
		backend.Commit()
	}

	var genesis protocol.Assertion
	var assertion protocol.Assertion
	var forkedAssertion protocol.Assertion
	err := chains[1].Call(func(tx protocol.ActiveTx) error {
		genesisAssertion, err := chains[1].AssertionBySequenceNum(ctx, tx, 0)
		if err != nil {
			return err
		}
		genesis = genesisAssertion
		return nil
	})
	require.NoError(t, err)

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	genesisStateHash := protocol.ComputeStateHash(genesisState, big.NewInt(1))

	require.Equal(t, genesisStateHash, genesis.StateHash(), "Genesis state hash unequal")

	height := uint64(0)
	honestValidatorStateRoots := make([]common.Hash, 0)
	evilValidatorStateRoots := make([]common.Hash, 0)
	honestValidatorStateRoots = append(honestValidatorStateRoots, genesisStateHash)
	evilValidatorStateRoots = append(evilValidatorStateRoots, genesisStateHash)

	honestBlockHash := common.Hash{}
	for i := uint64(1); i < numBlocks; i++ {
		height += 1
		honestBlockHash = backend.Commit()

		state := &protocol.ExecutionState{
			GlobalState: protocol.GoGlobalState{
				BlockHash: honestBlockHash,
				Batch:     1,
			},
			MachineStatus: protocol.MachineStatusFinished,
		}

		honestValidatorStateRoots = append(honestValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))

		// Before the divergence height, the evil validator agrees.
		if uint64(i) < divergenceHeight {
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		} else {
			junkRoot := make([]byte, 32)
			_, err := rand.Read(junkRoot)
			require.NoError(t, err)
			blockHash := crypto.Keccak256Hash(junkRoot)
			state.GlobalState.BlockHash = blockHash
			evilValidatorStateRoots = append(evilValidatorStateRoots, protocol.ComputeStateHash(state, big.NewInt(1)))
		}

	}

	height += 1
	honestBlockHash = backend.Commit()
	err = chains[1].Tx(func(tx protocol.ActiveTx) error {
		assertion, err = chains[1].CreateAssertion(
			ctx,
			tx,
			height, // Height.
			genesis.SeqNum(),
			genesisState,
			&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash: honestBlockHash,
					Batch:     1,
				},
				MachineStatus: protocol.MachineStatusFinished,
			},
			prevInboxMaxCount,
		)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	honestValidatorStateRoots = append(honestValidatorStateRoots, assertion.StateHash())

	evilPostState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.BytesToHash([]byte("evilcommit")),
			Batch:     1,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	err = chains[2].Tx(func(tx protocol.ActiveTx) error {
		forkedAssertion, err = chains[2].CreateAssertion(
			ctx,
			tx,
			height, // Height.
			genesis.SeqNum(),
			genesisState,
			evilPostState,
			prevInboxMaxCount,
		)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	evilValidatorStateRoots = append(evilValidatorStateRoots, forkedAssertion.StateHash())

	return &createdValidatorFork{
		leaf1:                     assertion,
		leaf2:                     forkedAssertion,
		assertionChains:           chains,
		accounts:                  accs,
		backend:                   backend,
		addrs:                     addrs,
		honestValidatorStateRoots: honestValidatorStateRoots,
		evilValidatorStateRoots:   evilValidatorStateRoots,
	}
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	tx := &mocks.MockActiveTx{}
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		v, p, _ := setupValidator(t)
		genesis := &mocks.MockAssertion{
			MockSeqNum:    0,
			MockHeight:    0,
			MockStateHash: common.Hash{},
			Prev:          util.None[*mocks.MockAssertion](),
		}
		p.On("LatestConfirmed", ctx, tx).Return(genesis, nil)
		p.On("NumAssertions", ctx, tx).Return(uint64(100), nil)
		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, genesis.SeqNum(), latestValid)
	})
	t.Run("all are valid, latest one is picked", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for _, a := range assertions {
			v.assertions[a.SeqNum()] = a
			s.On("HasStateCommitment", ctx, util.StateCommitment{
				Height:    a.Height(),
				StateRoot: a.StateHash(),
			}).Return(true)
		}
		p.On("LatestConfirmed", ctx, tx).Return(assertions[0], nil)
		p.On("NumAssertions", ctx, tx).Return(uint64(len(assertions)), nil)

		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertions[len(assertions)-1].SeqNum(), latestValid)
	})
	t.Run("latest valid is behind", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for i, a := range assertions {
			v.assertions[a.SeqNum()] = a
			if i <= 5 {
				s.On("HasStateCommitment", ctx, util.StateCommitment{
					Height:    a.Height(),
					StateRoot: a.StateHash(),
				}).Return(true)
			} else {
				s.On("HasStateCommitment", ctx, util.StateCommitment{
					Height:    a.Height(),
					StateRoot: a.StateHash(),
				}).Return(false)
			}
		}
		p.On("LatestConfirmed", ctx, tx).Return(assertions[0], nil)
		p.On("NumAssertions", ctx, tx).Return(uint64(len(assertions)), nil)
		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertions[5].SeqNum(), latestValid)
	})
}

func setupAssertions(num int) []protocol.Assertion {
	if num == 0 {
		return make([]protocol.Assertion, 0)
	}
	genesis := &mocks.MockAssertion{
		MockSeqNum:    0,
		MockHeight:    0,
		MockStateHash: common.Hash{},
		Prev:          util.None[*mocks.MockAssertion](),
	}
	assertions := []protocol.Assertion{genesis}
	for i := 1; i < num; i++ {
		assertions = append(assertions, protocol.Assertion(&mocks.MockAssertion{
			MockSeqNum:    protocol.AssertionSequenceNumber(i),
			MockHeight:    uint64(i),
			MockStateHash: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
			Prev:          util.Some(assertions[i-1].(*mocks.MockAssertion)),
		}))
	}
	return assertions
}

func setupValidator(t testing.TB) (*Validator, *mocks.MockProtocol, *mocks.MockStateManager) {
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	p.On(
		"AssertionBySequenceNum",
		ctx,
		&mocks.MockActiveTx{},
		protocol.AssertionSequenceNumber(0),
	).Return(&mocks.MockAssertion{}, nil)
	p.On("CurrentChallengeManager", ctx, &mocks.MockActiveTx{}).Return(&mocks.MockChallengeManager{}, nil)
	s := &mocks.MockStateManager{}
	_, _, addrs, backend := setupAssertionChains(t, 3)
	v, err := New(context.Background(), p, backend, s, addrs.Rollup)
	require.NoError(t, err)
	return v, p, s
}

// AssertLogsContain checks that the desired string is a subset of the current log output.
func AssertLogsContain(tb testing.TB, hook *test.Hook, want string, msg ...interface{}) {
	checkLogs(tb, hook, want, true, msg...)
}

// AssertLogsDoNotContain is the inverse check of LogsContain.
func AssertLogsDoNotContain(tb testing.TB, hook *test.Hook, want string, msg ...interface{}) {
	checkLogs(tb, hook, want, false, msg...)
}

// LogsContain checks whether a given substring is a part of logs. If flag=false, inverse is checked.
func checkLogs(tb testing.TB, hook *test.Hook, want string, flag bool, msg ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	entries := hook.AllEntries()
	logs := make([]string, 0, len(entries))
	match := false
	for _, e := range entries {
		msg, err := e.String()
		if err != nil {
			tb.Errorf("%s:%d Failed to format log entry to string: %v", filepath.Base(file), line, err)
			return
		}
		if strings.Contains(msg, want) {
			match = true
		}
		for _, field := range e.Data {
			fieldStr, ok := field.(string)
			if !ok {
				continue
			}
			if strings.Contains(fieldStr, want) {
				match = true
			}
		}
		logs = append(logs, msg)
	}
	var errMsg string
	if flag && !match {
		errMsg = parseMsg("Expected log not found", msg...)
	} else if !flag && match {
		errMsg = parseMsg("Unexpected log found", msg...)
	}
	if errMsg != "" {
		tb.Errorf("%s:%d %s: %v\nSearched logs:\n%v", filepath.Base(file), line, errMsg, want, logs)
	}
}

func parseMsg(defaultMsg string, msg ...interface{}) string {
	if len(msg) >= 1 {
		msgFormat, ok := msg[0].(string)
		if !ok {
			return defaultMsg
		}
		return fmt.Sprintf(msgFormat, msg[1:]...)
	}
	return defaultMsg
}
