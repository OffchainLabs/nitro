package validator

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/OffchainLabs/new-rollup-exploration/protocol"
	statemanager "github.com/OffchainLabs/new-rollup-exploration/state-manager"
	"github.com/OffchainLabs/new-rollup-exploration/testing/mocks"
	"github.com/OffchainLabs/new-rollup-exploration/util"
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

		parentSeqNum := protocol.SequenceNum(1)
		prevRoot := common.BytesToHash([]byte("foo"))
		parentAssertion := &protocol.Assertion{
			StateCommitment: protocol.StateCommitment{
				StateRoot: prevRoot,
				Height:    uint64(parentSeqNum),
			},
		}
		seqNum := parentSeqNum + 1
		newlyCreatedAssertion := &protocol.Assertion{
			Prev:            util.Some[*protocol.Assertion](parentAssertion),
			SequenceNum:     seqNum,
			StateCommitment: protocol.StateCommitment{},
			Staker:          util.Some[common.Address](common.BytesToAddress([]byte("foo"))),
		}
		ev := &protocol.CreateLeafEvent{
			PrevSeqNum:          parentAssertion.SequenceNum,
			PrevStateCommitment: parentAssertion.StateCommitment,
			SeqNum:              newlyCreatedAssertion.SequenceNum,
			StateCommitment:     newlyCreatedAssertion.StateCommitment,
			Validator:           newlyCreatedAssertion.Staker.Unwrap(),
		}

		s.On("HasStateCommitment", ctx, protocol.StateCommitment{}).Return(false)

		err := v.onLeafCreated(ctx, ev)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "No fork detected in assertion tree")
	})
	t.Run("fork leads validator to challenge leaf", func(t *testing.T) {
		logsHook := test.NewGlobal()
		leaf1, leaf2, validator := createTwoValidatorFork(t, context.Background())

		manager := &mocks.MockStateManager{}
		manager.On("HasStateCommitment", ctx, leaf1.StateCommitment).Return(false)
		manager.On("HasStateCommitment", ctx, leaf2.StateCommitment).Return(false)
		manager.On(
			"LatestHistoryCommitment",
			ctx,
		).Return(util.HistoryCommitment{
			Height: 1,
			Merkle: common.BytesToHash([]byte{1}),
		}, nil)
		validator.stateManager = manager

		err := validator.onLeafCreated(ctx, leaf1)
		require.NoError(t, err)
		err = validator.onLeafCreated(ctx, leaf2)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New leaf appended")
		AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")
	})
}

func Test_onChallengeStarted(t *testing.T) {
	ctx := context.Background()
	logsHook := test.NewGlobal()
	leaf1, leaf2, validator := createTwoValidatorFork(t, context.Background())

	manager := &mocks.MockStateManager{}

	manager.On("HasStateCommitment", ctx, leaf1.StateCommitment).Return(false)
	manager.On("HasStateCommitment", ctx, leaf2.StateCommitment).Return(false)
	manager.On(
		"LatestHistoryCommitment",
		ctx,
	).Return(util.HistoryCommitment{
		Height: 1,
		Merkle: common.BytesToHash([]byte{1}),
	}, nil)

	validator.stateManager = manager

	err := validator.onLeafCreated(ctx, leaf1)
	require.NoError(t, err)

	manager = &mocks.MockStateManager{}
	manager.On("HasStateCommitment", ctx, leaf1.StateCommitment).Return(false)
	manager.On("HasStateCommitment", ctx, leaf2.StateCommitment).Return(false)
	manager.On("LatestHistoryCommitment", ctx).Return(util.HistoryCommitment{
		Height: 1,
		Merkle: common.BytesToHash([]byte{1}),
	}, nil)
	validator.stateManager = manager

	err = validator.onLeafCreated(ctx, leaf2)
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "New leaf appended")
	AssertLogsContain(t, logsHook, "Successfully created challenge and added leaf")

	var challenge *protocol.Challenge
	err = validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		commit := protocol.StateCommitment{}
		id := protocol.AssertionStateCommitHash(commit.Hash())
		challenge, err = p.ChallengeByAssertionStateCommit(tx, id)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, challenge)

	manager = &mocks.MockStateManager{}
	manager.On("HasStateCommitment", ctx, leaf1.StateCommitment).Return(false)
	manager.On("HasStateCommitment", ctx, leaf2.StateCommitment).Return(false)
	manager.On("LatestHistoryCommitment", ctx).Return(util.HistoryCommitment{
		Height: 2,
		Merkle: common.BytesToHash([]byte{2}),
	}, nil)
	validator.stateManager = manager

	err = validator.onChallengeStarted(ctx, &protocol.StartChallengeEvent{
		ParentSeqNum:          0,
		ParentStateCommitment: challenge.ParentStateCommitment(),
		ParentStaker:          common.Address{},
		Validator:             common.BytesToAddress([]byte("other validator")),
	})
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "Received challenge for a created leaf, added own leaf")

	err = validator.onChallengeStarted(ctx, &protocol.StartChallengeEvent{
		ParentSeqNum:          0,
		ParentStateCommitment: challenge.ParentStateCommitment(),
		ParentStaker:          common.Address{},
		Validator:             common.BytesToAddress([]byte("other validator")),
	})
	require.NoError(t, err)
	AssertLogsContain(t, logsHook, "Attempted to add a challenge leaf that already exists")
}

func Test_submitOrFetchProtocolChallenge(t *testing.T) {
	logsHook := test.NewGlobal()
	ctx := context.Background()
	_, _, validator := createTwoValidatorFork(t, ctx)
	var genesis *protocol.Assertion
	var err error
	err = validator.chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		genesis = p.LatestConfirmed(tx)
		return nil
	})
	require.NoError(t, err)
	wantedChallenge, err := validator.submitOrFetchProtocolChallenge(ctx, genesis.SequenceNum, genesis.StateCommitment)
	require.NoError(t, err)
	gotChallenge, err := validator.submitOrFetchProtocolChallenge(ctx, genesis.SequenceNum, genesis.StateCommitment)
	require.NoError(t, err)
	require.Equal(t, wantedChallenge, gotChallenge)
	AssertLogsContain(t, logsHook, "Challenge on leaf already exists, reading existing challenge")
}

func createTwoValidatorFork(t *testing.T, ctx context.Context) (
	*protocol.CreateLeafEvent,
	*protocol.CreateLeafEvent,
	*Validator,
) {
	chain := protocol.NewAssertionChain(
		ctx,
		util.NewArtificialTimeReference(),
		time.Second,
	)
	staker1 := common.BytesToAddress([]byte("foo"))
	staker2 := common.BytesToAddress([]byte("bar"))
	staker3 := common.BytesToAddress([]byte("nyan"))
	stateManager := &mocks.MockStateManager{}
	v := setupValidatorWithChain(t, chain, stateManager, staker3)

	// Add balances to the stakers.
	bal := big.NewInt(0).Mul(protocol.Gwei, big.NewInt(100))
	err := chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		chain.AddToBalance(tx, staker1, bal)
		chain.AddToBalance(tx, staker2, bal)
		chain.AddToBalance(tx, staker3, bal)
		return nil
	})
	require.NoError(t, err)

	// Create some commitments.
	commit := protocol.StateCommitment{
		StateRoot: common.BytesToHash([]byte("baz")),
		Height:    1,
	}
	forkedCommit := protocol.StateCommitment{
		StateRoot: common.BytesToHash([]byte("woop")),
		Height:    2,
	}

	var genesis *protocol.Assertion
	var assertion *protocol.Assertion
	var forkedAssertion *protocol.Assertion
	err = chain.Call(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		genesis = chain.LatestConfirmed(tx)
		return nil
	})
	require.NoError(t, err)

	err = chain.Tx(func(tx *protocol.ActiveTx, p protocol.OnChainProtocol) error {
		assertion, err = chain.CreateLeaf(
			tx,
			genesis,
			commit,
			staker1,
		)
		if err != nil {
			return err
		}
		forkedAssertion, err = chain.CreateLeaf(
			tx,
			genesis,
			forkedCommit,
			staker2,
		)
		if err != nil {
			return err
		}
		return nil
	})
	require.NoError(t, err)

	ev1 := &protocol.CreateLeafEvent{
		PrevSeqNum:          genesis.SequenceNum,
		PrevStateCommitment: genesis.StateCommitment,
		SeqNum:              assertion.SequenceNum,
		StateCommitment:     assertion.StateCommitment,
		Validator:           staker1,
	}
	ev2 := &protocol.CreateLeafEvent{
		PrevSeqNum:          genesis.SequenceNum,
		PrevStateCommitment: genesis.StateCommitment,
		SeqNum:              forkedAssertion.SequenceNum,
		StateCommitment:     forkedAssertion.StateCommitment,
		Validator:           staker2,
	}
	return ev1, ev2, v
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	tx := &protocol.ActiveTx{}
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		v, p, _ := setupValidator(t)
		genesis := &protocol.Assertion{
			SequenceNum: 0,
			StateCommitment: protocol.StateCommitment{
				Height:    0,
				StateRoot: common.Hash{},
			},
			Prev:   util.None[*protocol.Assertion](),
			Staker: util.None[common.Address](),
		}
		p.On("LatestConfirmed", tx).Return(genesis)
		p.On("NumAssertions", tx).Return(uint64(100))
		latestValid := v.findLatestValidAssertion(ctx)
		require.Equal(t, genesis.SequenceNum, latestValid)
	})
	t.Run("all are valid, latest one is picked", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for _, a := range assertions {
			v.assertions[a.SequenceNum] = &protocol.CreateLeafEvent{
				StateCommitment: a.StateCommitment,
				SeqNum:          a.SequenceNum,
			}
			s.On("HasStateCommitment", ctx, a.StateCommitment).Return(true)
		}
		p.On("LatestConfirmed", tx).Return(assertions[0])
		p.On("NumAssertions", tx).Return(uint64(len(assertions)))

		latestValid := v.findLatestValidAssertion(ctx)
		require.Equal(t, assertions[len(assertions)-1].SequenceNum, latestValid)
	})
	t.Run("latest valid is behind", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for i, a := range assertions {
			v.assertions[a.SequenceNum] = &protocol.CreateLeafEvent{
				StateCommitment: a.StateCommitment,
				SeqNum:          a.SequenceNum,
			}
			if i <= 5 {
				s.On("HasStateCommitment", ctx, a.StateCommitment).Return(true)
			} else {
				s.On("HasStateCommitment", ctx, a.StateCommitment).Return(false)
			}
		}
		p.On("LatestConfirmed", tx).Return(assertions[0])
		p.On("NumAssertions", tx).Return(uint64(len(assertions)))
		latestValid := v.findLatestValidAssertion(ctx)
		require.Equal(t, assertions[5].SequenceNum, latestValid)
	})
}

func setupAssertions(num int) []*protocol.Assertion {
	if num == 0 {
		return make([]*protocol.Assertion, 0)
	}
	genesis := &protocol.Assertion{
		SequenceNum: 0,
		StateCommitment: protocol.StateCommitment{
			Height:    0,
			StateRoot: common.Hash{},
		},
		Prev:   util.None[*protocol.Assertion](),
		Staker: util.None[common.Address](),
	}
	assertions := []*protocol.Assertion{genesis}
	for i := 1; i < num; i++ {
		assertions = append(assertions, &protocol.Assertion{
			SequenceNum: protocol.SequenceNum(i),
			StateCommitment: protocol.StateCommitment{
				Height:    uint64(i),
				StateRoot: common.BytesToHash([]byte(fmt.Sprintf("%d", i))),
			},
			Prev:   util.Some[*protocol.Assertion](assertions[i-1]),
			Staker: util.None[common.Address](),
		})
	}
	return assertions
}

func setupValidatorWithChain(
	t testing.TB, chain protocol.ChainReadWriter, manager statemanager.Manager, staker common.Address,
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
