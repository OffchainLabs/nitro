package validator

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	"github.com/OffchainLabs/challenge-protocol-v2/containers/option"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/rollupgen"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/mocks"
	"github.com/OffchainLabs/challenge-protocol-v2/testing/setup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func Test_onLeafCreation(t *testing.T) {
	ctx := context.Background()
	t.Run("no fork detected", func(t *testing.T) {
		logsHook := test.NewGlobal()
		v, _, _ := setupValidator(t)

		prev := &mocks.MockAssertion{
			MockPrevId:       mockId(1),
			MockId:           mockId(1),
			MockStateHash:    common.Hash{},
			MockIsFirstChild: true,
		}
		ev := &mocks.MockAssertion{
			MockPrevId:       mockId(1),
			MockId:           mockId(2),
			MockStateHash:    common.BytesToHash([]byte("bar")),
			MockIsFirstChild: true,
		}

		p := &mocks.MockProtocol{}
		p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
		p.On("GetAssertion", ctx, mockId(0)).Return(prev, nil)
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
			DivergeBlockHeight: 5,
		})
		require.NoError(t, err)

		validator, err := New(
			ctx,
			createdData.Chains[1],
			createdData.Backend,
			createdData.HonestStateManager,
			createdData.Addrs.Rollup,
		)
		require.NoError(t, err)

		err = validator.onLeafCreated(ctx, createdData.Leaf1)
		require.NoError(t, err)

		anotherValidator, err := New(
			ctx,
			createdData.Chains[0],
			createdData.Backend,
			createdData.EvilStateManager,
			createdData.Addrs.Rollup,
		)
		require.NoError(t, err)

		err = anotherValidator.onLeafCreated(ctx, createdData.Leaf2)
		require.NoError(t, err)

		AssertLogsContain(t, logsHook, "New assertion appended")
		AssertLogsContain(t, logsHook, "Successfully created level zero edge")

		err = anotherValidator.onLeafCreated(ctx, createdData.Leaf2)
		require.ErrorContains(t, err, "Edge already exists")
	})
}

func Test_findLatestValidAssertion(t *testing.T) {
	ctx := context.Background()
	t.Run("only valid latest assertion is genesis", func(t *testing.T) {
		v, p, s := setupValidator(t)
		setupAssertions(ctx, p, s, 10, func(int) bool { return false })
		p.On("LatestConfirmed", ctx).Return(0, nil)
		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(0), latestValid)
	})
	t.Run("all are valid, latest one is picked", func(t *testing.T) {
		v, p, s := setupValidator(t)
		numAssertions := 10
		setupAssertions(ctx, p, s, numAssertions, func(int) bool { return true })

		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(10), latestValid)
	})
	t.Run("latest valid is behind", func(t *testing.T) {
		v, p, s := setupValidator(t)
		setupAssertions(ctx, p, s, 10, func(i int) bool { return i <= 5 })
		p.On("LatestConfirmed", ctx).Return(1, nil)

		latestValid, err := v.findLatestValidAssertion(ctx)
		require.NoError(t, err)
		require.Equal(t, mockId(5), latestValid)
	})
}

func mockId(x uint64) protocol.AssertionId {
	return protocol.AssertionId(common.BytesToHash([]byte(fmt.Sprintf("%d", x))))
}

func setupAssertions(ctx context.Context, p *mocks.MockProtocol, s *mocks.MockStateManager, num int, validity func(int) bool) []protocol.Assertion {
	if num == 0 {
		return make([]protocol.Assertion, 0)
	}
	genesis := &mocks.MockAssertion{
		MockId:        mockId(0),
		MockPrevId:    mockId(0),
		MockHeight:    0,
		MockStateHash: common.Hash{},
		Prev:          option.None[*mocks.MockAssertion](),
	}
	p.On(
		"GetAssertion",
		ctx,
		mockId(uint64(0)),
	).Return(genesis, nil)
	assertions := []protocol.Assertion{genesis}
	for i := 1; i <= num; i++ {
		mockHash := common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		assertion := protocol.Assertion(&mocks.MockAssertion{
			MockId:        mockId(uint64(i)),
			MockPrevId:    mockId(uint64(i - 1)),
			MockHeight:    uint64(i),
			MockStateHash: mockHash,
			Prev:          option.Some(assertions[i-1].(*mocks.MockAssertion)),
		})
		assertions = append(assertions, assertion)
		p.On(
			"GetAssertion",
			ctx,
			mockId(uint64(i)),
		).Return(assertion, nil)
		mockState := rollupgen.ExecutionState{
			MachineStatus: uint8(protocol.MachineStatusFinished),
			GlobalState: rollupgen.GlobalState(protocol.GoGlobalState{
				BlockHash: mockHash,
			}.AsSolidityStruct()),
		}
		mockAssertionCreationInfo := &protocol.AssertionCreatedInfo{
			AfterState: mockState,
		}
		p.On(
			"ReadAssertionCreationInfo",
			ctx,
			mockId(uint64(i)),
		).Return(mockAssertionCreationInfo, nil)
		valid := validity(i)
		s.On("ExecutionStateBlockHeight", ctx, protocol.GoExecutionStateFromSolidity(mockState)).Return(uint64(i), valid)

		if i == 1 {
			var firstValid protocol.Assertion = genesis
			if valid {
				firstValid = assertion
			}
			p.On("LatestConfirmed", ctx).Return(firstValid, nil)
		}
	}
	p.On("LatestConfirmed", ctx).Return(assertions[0], nil)
	p.On("LatestCreatedAssertion", ctx).Return(assertions[len(assertions)-1], nil)
	return assertions
}

func setupValidator(t *testing.T) (*Manager, *mocks.MockProtocol, *mocks.MockStateManager) {
	t.Helper()
	p := &mocks.MockProtocol{}
	ctx := context.Background()
	p.On("CurrentChallengeManager", ctx).Return(&mocks.MockChallengeManager{}, nil)
	p.On("SpecChallengeManager", ctx).Return(&mocks.MockSpecChallengeManager{}, nil)
	s := &mocks.MockStateManager{}
	cfg, err := setup.ChainsWithEdgeChallengeManager()
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
