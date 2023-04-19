package validator

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	statemanager "github.com/OffchainLabs/challenge-protocol-v2/state-manager"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_onLeafCreation(t *testing.T) {
	ctx := context.Background()
	t.Run("no fork detected", func(t *testing.T) {
		logsHook := test.NewGlobal()
		v, _, s := setupValidator(t)

		parentSeqNum := protocol.AssertionSequenceNumber(1)
		seqNum := parentSeqNum + 1
		prev := &mocks.MockAssertion{
			MockPrevSeqNum: 1,
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
		p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
		p.On("AssertionBySequenceNum", ctx, prev.SeqNum()).Return(prev, nil)
		v.chain = p

		err := v.onLeafCreated(ctx, ev)
		require.NoError(t, err)
		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "No fork detected in assertion tree")
	})
	t.Run("fork leads validator to challenge leaf", func(t *testing.T) {
		logsHook := test.NewGlobal()
		ctx := context.Background()
		createdData, err := setup.CreateTwoValidatorFork(ctx, &setup.CreateForkConfig{
			NumBlocks:     20,
			DivergeHeight: 10,
		})
		require.NoError(t, err)

		manager, err := statemanager.NewWithAssertionStates(createdData.HonestValidatorStates, createdData.HonestValidatorInboxCounts)
		require.NoError(t, err)

		validator, err := New(
			ctx,
			createdData.Chains[1],
			createdData.Backend,
			manager,
			createdData.Addrs.Rollup,
		)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, createdData.Leaf1)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, createdData.Leaf2)
		require.NoError(t, err)

		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "Successfully created level zero edge")

		err = validator.onLeafCreated(ctx, createdData.Leaf2)
		require.ErrorContains(t, err, "Edge already exists")
	})
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		v, p, _ := setupValidator(t)
		genesis := &mocks.MockAssertion{
			MockSeqNum:    0,
			MockHeight:    0,
			MockStateHash: common.Hash{},
			Prev:          util.None[*mocks.MockAssertion](),
		}
		p.On("LatestConfirmed", ctx).Return(genesis, nil)
		p.On("NumAssertions", ctx).Return(uint64(100), nil)
		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, genesis.SeqNum(), latestValid)
	})
	t.Run("all are valid, latest one is picked", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for _, a := range assertions {
			v.assertions[a.SeqNum()] = a
			height, err := a.Height()
			require.NoError(t, err)
			stateHash, err := a.StateHash()
			require.NoError(t, err)
			s.On("HasStateCommitment", ctx, util.StateCommitment{
				Height:    height,
				StateRoot: stateHash,
			}).Return(true)
		}
		p.On("LatestConfirmed", ctx).Return(assertions[0], nil)
		p.On("NumAssertions", ctx).Return(uint64(len(assertions)), nil)

		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, assertions[len(assertions)-1].SeqNum(), latestValid)
	})
	t.Run("latest valid is behind", func(t *testing.T) {
		v, p, s := setupValidator(t)
		assertions := setupAssertions(10)
		for i, a := range assertions {
			v.assertions[a.SeqNum()] = a
			height, err := a.Height()
			require.NoError(t, err)
			stateHash, err := a.StateHash()
			require.NoError(t, err)
			if i <= 5 {
				s.On("HasStateCommitment", ctx, util.StateCommitment{
					Height:    height,
					StateRoot: stateHash,
				}).Return(true)
			} else {
				s.On("HasStateCommitment", ctx, util.StateCommitment{
					Height:    height,
					StateRoot: stateHash,
				}).Return(false)
			}
		}
		p.On("LatestConfirmed", ctx).Return(assertions[0], nil)
		p.On("NumAssertions", ctx).Return(uint64(len(assertions)), nil)
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

func setupValidator(t *testing.T) (*Validator, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	p.On(
		"AssertionBySequenceNum",
		ctx,
		protocol.AssertionSequenceNumber(1),
	).Return(&mocks.MockAssertion{}, nil)
	p.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.SetupChainsWithEdgeChallengeManager()
	require.NoError(t, err)
	v, err := New(context.Background(), p, cfg.Backend, s, cfg.Addrs.Rollup)
	require.NoError(t, err)
	return v, p, s
}

// AssertLogsContain checks that the desired string is a subset of the current log output.
func AssertLogsContain(tb testing.TB, hook *test.Hook, want string, msg ...interface{}) {
	checkLogs(tb, hook, want, true, msg...)
}

// AssertLogsDoNotContain is the inverse check of LogsContain.

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
