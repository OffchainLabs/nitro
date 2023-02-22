package validator

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"math/big"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
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
		prevRoot := common.BytesToHash([]byte("foo"))
		seqNum := parentSeqNum + 1
		ev := &protocol.CreateLeafEvent{
			PrevSeqNum:    parentSeqNum,
			PrevStateHash: prevRoot,
			SeqNum:        seqNum,
			StateHash:     common.BytesToHash([]byte("bar")),
			Validator:     common.BytesToAddress([]byte("alice")),
		}

		s.On("HasStateCommitment", ctx, util.StateCommitment{}).Return(false)

		err := v.onLeafCreated(ctx, ev)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "No fork detected in assertion tree")
	})
	t.Run("fork leads validator to challenge leaf", func(t *testing.T) {
		logsHook := test.NewGlobal()
		ctx := context.Background()
		leaf1, leaf2, chains, _, _ := createTwoValidatorFork(t, ctx)

		// Setup our mock state manager to agree on leaf1 but disagree on leaf2.
		manager := &mocks.MockStateManager{}
		manager.On("HasStateCommitment", ctx, util.StateCommitment{
			Height:    leaf1.Height,
			StateRoot: leaf1.StateHash,
		}).Return(true)
		manager.On("HasStateCommitment", ctx, util.StateCommitment{
			Height:    leaf2.Height,
			StateRoot: leaf2.StateHash,
		}).Return(false)

		manager.On(
			"HistoryCommitmentUpTo",
			ctx,
			uint64(leaf1.Height),
		).Return(util.HistoryCommitment{
			Height: leaf1.Height,
			Merkle: leaf1.StateHash, // TODO: Change
		}, nil)

		validator, err := New(ctx, chains[1], manager)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, leaf1)
		require.NoError(t, err)
		err = validator.onLeafCreated(ctx, leaf2)
		require.NoError(t, err)

		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

		err = validator.onLeafCreated(ctx, leaf2)
		require.ErrorContains(t, err, "Vertex already exists")
	})
}

// func Test_onChallengeStarted(t *testing.T) {
// 	ctx := context.Background()
// 	logsHook := test.NewGlobal()

// 	stateRoots := generateStateRoots(10)
// 	manager := &mocks.MockStateManager{}
// 	manager.On("HasStateCommitment", ctx, util.StateCommitment{
// 		Height:    5,
// 		StateRoot: stateRoots[5],
// 	}).Return(false)
// 	manager.On("HasStateCommitment", ctx, util.StateCommitment{
// 		Height:    6,
// 		StateRoot: stateRoots[6],
// 	}).Return(true)

// 	commit6, err := util.NewHistoryCommitment(
// 		6,
// 		stateRoots[:7],
// 		util.WithLastElementProof(stateRoots[:7]),
// 	)
// 	require.NoError(t, err)

// 	manager.On(
// 		"HistoryCommitmentUpTo",
// 		ctx,
// 		uint64(6),
// 	).Return(commit6, nil)

// 	commit4, err := util.NewHistoryCommitment(
// 		4,
// 		stateRoots[:5],
// 		util.WithLastElementProof(stateRoots[:5]),
// 	)
// 	require.NoError(t, err)

// 	manager.On(
// 		"HistoryCommitmentUpTo",
// 		ctx,
// 		uint64(4),
// 	).Return(commit4, nil)
// 	leaf1, leaf2, validator := createTwoValidatorFork(t, context.Background(), manager, stateRoots)

// 	err = validator.onLeafCreated(ctx, leaf1)
// 	require.NoError(t, err)
// 	err = validator.onLeafCreated(ctx, leaf2)
// 	require.NoError(t, err)
// 	AssertLogsContain(t, logsHook, "New leaf appended")
// 	AssertLogsContain(t, logsHook, "New leaf appended")
// 	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

// 	var challenge protocol.Challenge
// 	err = validator.chain.Call(func(tx protocol.ActiveTx) error {
// 		commit := util.StateCommitment{}
// 		id := protocol.ChallengeHash(commit.Hash())
// 		challenge, err = validator.chain.ChallengeByCommitHash(tx, id)
// 		if err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// 	require.NoError(t, err)
// 	require.NotNil(t, challenge)

// 	manager = &mocks.MockStateManager{}
// 	manager.On("HasStateCommitment", ctx, leaf1.StateCommitment).Return(false)
// 	manager.On("HasStateCommitment", ctx, leaf2.StateCommitment).Return(true)

// 	commit6.Merkle = common.BytesToHash([]byte("forked commit"))
// 	commit4.Merkle = common.BytesToHash([]byte("forked commit"))
// 	manager.On("HistoryCommitmentUpTo", ctx, uint64(6)).Return(commit6, nil)
// 	manager.On("HistoryCommitmentUpTo", ctx, uint64(4)).Return(commit4, nil)
// 	validator.stateManager = manager

// 	parentStateCommitment, err := challenge.ParentStateCommitment(ctx, &mocks.MockActiveTx{ReadWriteTx: false})
// 	require.NoError(t, err)
// 	err = validator.onChallengeStarted(ctx, &protocol.StartChallengeEvent{
// 		ParentSeqNum:    0,
// 		ParentStateHash: parentStateCommitment.StateRoot,
// 		ParentStaker:    common.Address{},
// 		Validator:       common.BytesToAddress([]byte("other validator")),
// 	})
// 	require.NoError(t, err)
// 	AssertLogsContain(t, logsHook, "Received challenge for a created leaf, added own leaf")

// 	err = validator.onChallengeStarted(ctx, &protocol.StartChallengeEvent{
// 		ParentSeqNum:    0,
// 		ParentStateHash: parentStateCommitment.StateRoot,
// 		ParentStaker:    common.Address{},
// 		Validator:       common.BytesToAddress([]byte("other validator")),
// 	})
// 	require.NoError(t, err)
// 	AssertLogsContain(t, logsHook, "Attempted to add a challenge leaf that already exists")
// }

// func Test_submitAndFetchProtocolChallenge(t *testing.T) {
// 	ctx := context.Background()
// 	stateRoots := generateStateRoots(10)
// 	_, _, validator := createTwoValidatorFork(t, ctx, &mocks.MockStateManager{}, stateRoots)
// 	var genesis *goimpl.Assertion
// 	var err error
// 	err = validator.chain.Call(func(tx *goimpl.ActiveTx) error {
// 		genesis = validator.chain.LatestConfirmed(tx)
// 		return nil
// 	})
// 	require.NoError(t, err)
// 	wantedChallenge, err := validator.submitProtocolChallenge(ctx, genesis.SequenceNum)
// 	require.NoError(t, err)
// 	gotChallenge, err := validator.fetchProtocolChallenge(ctx, genesis.SequenceNum, genesis.StateCommitment)
// 	require.NoError(t, err)
// 	require.Equal(t, wantedChallenge, gotChallenge)
// }

func createTwoValidatorFork(
	t *testing.T,
	ctx context.Context,
) (*protocol.CreateLeafEvent, *protocol.CreateLeafEvent, []*solimpl.AssertionChain, []*testAccount, *backends.SimulatedBackend) {
	chains, accs, _, backend := setupAssertionChains(t, 3)
	prevInboxMaxCount := big.NewInt(1)

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

	latestBlockHash := common.Hash{}
	for i := 0; i < 100; i++ {
		latestBlockHash = backend.Commit()
	}

	genesisState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash: common.Hash{},
		},
		MachineStatus: protocol.MachineStatusFinished,
	}

	err = chains[1].Tx(func(tx protocol.ActiveTx) error {
		assertion, err = chains[1].CreateAssertion(
			ctx,
			tx,
			5, // Height.
			genesis.SeqNum(),
			genesisState,
			&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash: latestBlockHash,
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

	err = chains[2].Tx(func(tx protocol.ActiveTx) error {
		forkedAssertion, err = chains[2].CreateAssertion(
			ctx,
			tx,
			6, // Height.
			genesis.SeqNum(),
			genesisState,
			&protocol.ExecutionState{
				GlobalState: protocol.GoGlobalState{
					BlockHash: common.BytesToHash([]byte("malicious commit")),
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

	ev1 := &protocol.CreateLeafEvent{
		PrevSeqNum:    genesis.PrevSeqNum(),
		PrevStateHash: genesis.StateHash(),
		PrevHeight:    0,
		Height:        assertion.Height(),
		SeqNum:        assertion.SeqNum(),
		StateHash:     assertion.StateHash(),
		Validator:     accs[1].accountAddr,
	}
	ev2 := &protocol.CreateLeafEvent{
		PrevSeqNum:    genesis.PrevSeqNum(),
		PrevStateHash: genesis.StateHash(),
		PrevHeight:    0,
		Height:        forkedAssertion.Height(),
		SeqNum:        forkedAssertion.SeqNum(),
		StateHash:     forkedAssertion.StateHash(),
		Validator:     accs[2].accountAddr,
	}
	return ev1, ev2, chains, accs, backend
}

// func Test_findLatestValidAssertion(t *testing.T) {
// 	ctx := context.Background()
// 	tx := &goimpl.ActiveTx{TxStatus: goimpl.ReadOnlyTxStatus}
// 	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
// 		v, p, _ := setupValidator(t)
// 		genesis := &goimpl.Assertion{
// 			SequenceNum: 0,
// 			StateCommitment: util.StateCommitment{
// 				Height:    0,
// 				StateRoot: common.Hash{},
// 			},
// 			Prev:   util.None[*goimpl.Assertion](),
// 			Staker: util.None[common.Address](),
// 		}
// 		p.On("LatestConfirmed", tx).Return(genesis)
// 		p.On("NumAssertions", tx).Return(uint64(100))
// 		latestValid := v.findLatestValidAssertion(ctx)
// 		require.Equal(t, genesis.SequenceNum, latestValid)
// 	})
// 	t.Run("all are valid, latest one is picked", func(t *testing.T) {
// 		v, p, s := setupValidator(t)
// 		assertions := setupAssertions(10)
// 		for _, a := range assertions {
// 			v.assertions[a.SequenceNum] = &goimpl.CreateLeafEvent{
// 				StateCommitment: a.StateCommitment,
// 				SeqNum:          a.SequenceNum,
// 			}
// 			s.On("HasStateCommitment", ctx, a.StateCommitment).Return(true)
// 		}
// 		p.On("LatestConfirmed", tx).Return(assertions[0])
// 		p.On("NumAssertions", tx).Return(uint64(len(assertions)))

// 		latestValid := v.findLatestValidAssertion(ctx)
// 		require.Equal(t, assertions[len(assertions)-1].SequenceNum, latestValid)
// 	})
// 	t.Run("latest valid is behind", func(t *testing.T) {
// 		v, p, s := setupValidator(t)
// 		assertions := setupAssertions(10)
// 		for i, a := range assertions {
// 			v.assertions[a.SequenceNum] = &goimpl.CreateLeafEvent{
// 				StateCommitment: a.StateCommitment,
// 				SeqNum:          a.SequenceNum,
// 			}
// 			if i <= 5 {
// 				s.On("HasStateCommitment", ctx, a.StateCommitment).Return(true)
// 			} else {
// 				s.On("HasStateCommitment", ctx, a.StateCommitment).Return(false)
// 			}
// 		}
// 		p.On("LatestConfirmed", tx).Return(assertions[0])
// 		p.On("NumAssertions", tx).Return(uint64(len(assertions)))
// 		latestValid := v.findLatestValidAssertion(ctx)
// 		require.Equal(t, assertions[5].SequenceNum, latestValid)
// 	})
// }

// func setupAssertions(num int) []*goimpl.Assertion {
// 	if num == 0 {
// 		return make([]*goimpl.Assertion, 0)
// 	}
// 	genesis := &goimpl.Assertion{
// 		SequenceNum: 0,
// 		StateCommitment: util.StateCommitment{
// 			Height:    0,
// 			StateRoot: common.Hash{},
// 		},
// 		Prev:   util.None[*goimpl.Assertion](),
// 		Staker: util.None[common.Address](),
// 	}
// 	assertions := []*goimpl.Assertion{genesis}
// 	for i := 1; i < num; i++ {
// 		assertions = append(assertions, &goimpl.Assertion{
// 			SequenceNum: goimpl.AssertionSequenceNumber(i),
// 			StateCommitment: util.StateCommitment{
// 				Height:    uint64(i),
// 				StateRoot: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
// 			},
// 			Prev:   util.Some[*goimpl.Assertion](assertions[i-1]),
// 			Staker: util.None[common.Address](),
// 		})
// 	}
// 	return assertions
// }

func setupValidatorWithChain(
	t testing.TB, chain protocol.Protocol, manager statemanager.Manager, staker common.Address,
) *Validator {
	v, err := New(context.Background(), chain, manager, WithAddress(staker))
	require.NoError(t, err)
	return v
}

func setupValidator(t testing.TB) (*Validator, *mocks.MockProtocol, *mocks.MockStateManager) {
	p := &mocks.MockProtocol{}
	s := &mocks.MockStateManager{}
	v, err := New(context.Background(), p, s)
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
