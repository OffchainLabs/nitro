package assertions

import (
	"context"
	"testing"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/OffchainLabs/bold/solgen/go/rollupgen"
	"github.com/OffchainLabs/bold/testing/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFindLastAgreedWithAncestor(t *testing.T) {
	ctx := context.Background()

	t.Run("Error reading assertion creation info", func(t *testing.T) {
		chain := &mocks.MockProtocol{}
		genesis := common.BytesToHash([]byte("genesis"))
		assertion := &mocks.MockAssertion{MockId: protocol.AssertionHash{Hash: genesis}}
		chain.On("LatestConfirmed", ctx).Return(assertion, nil)
		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: genesis}).Return(&protocol.AssertionCreatedInfo{}, errors.New("error"))
		manager := &Manager{chain: chain}

		_, err := manager.findLastAgreedWithAncestor(ctx, &protocol.AssertionCreatedInfo{})
		assert.Error(t, err)
	})

	t.Run("Reaches latest confirmed without finding agreed upon ancestor", func(t *testing.T) {
		chain := &mocks.MockProtocol{}
		stateProvider := &mocks.MockStateManager{}
		genesis := common.BytesToHash([]byte("genesis"))
		parent := common.BytesToHash([]byte("parent-of-latest"))
		latest := common.BytesToHash([]byte("latest"))
		genesisAssertion := &mocks.MockAssertion{MockId: protocol.AssertionHash{Hash: genesis}}
		chain.On("LatestConfirmed", ctx).Return(genesisAssertion, nil)

		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: genesis}).Return(
			&protocol.AssertionCreatedInfo{AssertionHash: genesis}, nil,
		)
		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: parent}).Return(
			&protocol.AssertionCreatedInfo{AssertionHash: parent, ParentAssertionHash: genesis}, nil,
		)

		stateProvider.On("AgreesWithExecutionState", ctx, mock.Anything).Return(l2stateprovider.ErrNoExecutionState)
		manager := &Manager{chain: chain, stateProvider: stateProvider}

		ancestor, err := manager.findLastAgreedWithAncestor(ctx, &protocol.AssertionCreatedInfo{
			AssertionHash:       latest,
			ParentAssertionHash: parent,
		})
		assert.NoError(t, err)
		assert.Equal(t, genesis, ancestor.AssertionHash)
	})

	t.Run("Finds agreed upon ancestor before latest confirmed", func(t *testing.T) {
		chain := &mocks.MockProtocol{}
		stateProvider := &mocks.MockStateManager{}
		genesis := common.BytesToHash([]byte("genesis"))
		parent := common.BytesToHash([]byte("parent-of-latest"))
		latest := common.BytesToHash([]byte("latest"))
		genesisAssertion := &mocks.MockAssertion{MockId: protocol.AssertionHash{Hash: genesis}}
		chain.On("LatestConfirmed", ctx).Return(genesisAssertion, nil)

		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: genesis}).Return(
			&protocol.AssertionCreatedInfo{AssertionHash: genesis}, nil,
		)
		execState := rollupgen.ExecutionState{
			GlobalState: rollupgen.GlobalState{
				U64Vals: [2]uint64{1, 0},
			},
			MachineStatus: uint8(protocol.MachineStatusFinished),
		}
		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: parent}).Return(
			&protocol.AssertionCreatedInfo{
				AssertionHash:       parent,
				ParentAssertionHash: genesis,
				AfterState:          execState,
			}, nil,
		)

		goExec := protocol.GoExecutionStateFromSolidity(execState)
		stateProvider.On("AgreesWithExecutionState", ctx, goExec).Return(nil)
		manager := &Manager{chain: chain, stateProvider: stateProvider}

		ancestor, err := manager.findLastAgreedWithAncestor(ctx, &protocol.AssertionCreatedInfo{
			AssertionHash:       latest,
			ParentAssertionHash: parent,
		})
		assert.NoError(t, err)
		assert.Equal(t, parent, ancestor.AssertionHash)
	})

	t.Run("State provider returns any other error", func(t *testing.T) {
		chain := &mocks.MockProtocol{}
		stateProvider := &mocks.MockStateManager{}
		genesis := common.BytesToHash([]byte("genesis"))
		parent := common.BytesToHash([]byte("parent-of-latest"))
		latest := common.BytesToHash([]byte("latest"))
		genesisAssertion := &mocks.MockAssertion{MockId: protocol.AssertionHash{Hash: genesis}}
		chain.On("LatestConfirmed", ctx).Return(genesisAssertion, nil)

		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: genesis}).Return(
			&protocol.AssertionCreatedInfo{AssertionHash: genesis}, nil,
		)
		chain.On("ReadAssertionCreationInfo", ctx, protocol.AssertionHash{Hash: parent}).Return(
			&protocol.AssertionCreatedInfo{AssertionHash: parent, ParentAssertionHash: genesis}, nil,
		)

		stateProvider.On("AgreesWithExecutionState", ctx, mock.Anything).Return(errors.New("errored"))
		manager := &Manager{chain: chain, stateProvider: stateProvider}

		_, err := manager.findLastAgreedWithAncestor(ctx, &protocol.AssertionCreatedInfo{
			AssertionHash:       latest,
			ParentAssertionHash: parent,
		})
		assert.ErrorContains(t, err, "errored")
	})
}
