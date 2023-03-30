package solimpl

import (
	"context"
	"testing"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/stretchr/testify/require"
	"math/big"
)

var (
	_ = protocol.SpecEdge(&SpecEdge{})
	_ = protocol.SpecChallengeManager(&SpecChallengeManager{})
)

func TestEdgeChallengeManager(t *testing.T) {
	ctx := context.Background()
	height1 := protocol.Height(3)
	height2 := protocol.Height(3)
	a1, a2, chain1, _ := setupEdgeBasedFork(
		t, ctx, height1, height2,
	)
	_ = a1
	_ = a2
	manager, err := chain1.SpecChallengeManager(ctx)
	require.NoError(t, err)
	edge, err := manager.GetEdge(ctx, [32]byte{})
	require.NoError(t, err)
	t.Logf("%+v", edge)
}

func setupEdgeBasedFork(
	t *testing.T, ctx context.Context, h1, h2 protocol.Height,
) (protocol.Assertion, protocol.Assertion, protocol.AssertionChain, protocol.AssertionChain) {
	chain1, accs, addresses, backend, headerReader := setupChainWithEdgeChallengeManager(t)
	prev := uint64(0)

	minAssertionPeriod, err := chain1.userLogic.MinimumAssertionPeriod(chain1.callOpts)
	require.NoError(t, err)

	latestBlockHash := common.Hash{}
	for i := uint64(0); i < minAssertionPeriod.Uint64(); i++ {
		latestBlockHash = backend.Commit()
	}

	prevState := &protocol.ExecutionState{
		GlobalState:   protocol.GoGlobalState{},
		MachineStatus: protocol.MachineStatusFinished,
	}
	postState := &protocol.ExecutionState{
		GlobalState: protocol.GoGlobalState{
			BlockHash:  latestBlockHash,
			SendRoot:   common.Hash{},
			Batch:      1,
			PosInBatch: 0,
		},
		MachineStatus: protocol.MachineStatusFinished,
	}
	prevInboxMaxCount := big.NewInt(1)
	a1, err := chain1.CreateAssertion(
		ctx,
		uint64(h1),
		protocol.AssertionSequenceNumber(prev),
		prevState,
		postState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)

	chain2, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[2].txOpts,
		&bind.CallOpts{},
		accs[2].accountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
	)
	require.NoError(t, err)

	postState.GlobalState.BlockHash = common.BytesToHash([]byte("evil"))
	a2, err := chain2.CreateAssertion(
		ctx,
		uint64(h2),
		protocol.AssertionSequenceNumber(prev),
		prevState,
		postState,
		prevInboxMaxCount,
	)
	require.NoError(t, err)
	return a1, a2, chain1, chain2
}

func setupChainWithEdgeChallengeManager(t *testing.T) (
	*AssertionChain, []*testAccount, *rollupAddresses, *backends.SimulatedBackend, *headerreader.HeaderReader,
) {
	t.Helper()
	ctx := context.Background()
	accs, backend := setupAccounts(t, 3)
	prod := false
	wasmModuleRoot := common.Hash{}
	rollupOwner := accs[0].accountAddr
	chainId := big.NewInt(1337)
	loserStakeEscrow := common.Address{}
	challengePeriodSeconds := big.NewInt(100)
	miniStake := big.NewInt(1)
	cfg := generateRollupConfig(
		prod,
		wasmModuleRoot,
		rollupOwner,
		chainId,
		loserStakeEscrow,
		challengePeriodSeconds,
		miniStake,
	)
	addresses := deployFullRollupStack(
		t,
		ctx,
		backend,
		accs[0].txOpts,
		common.Address{}, // Sequencer addr.
		cfg,
	)
	headerReader := headerreader.New(util.SimulatedBackendWrapper{SimulatedBackend: backend}, func() *headerreader.Config { return &headerreader.TestConfig })
	headerReader.Start(ctx)
	chain, err := NewAssertionChain(
		ctx,
		addresses.Rollup,
		accs[1].txOpts,
		&bind.CallOpts{},
		accs[1].accountAddr,
		backend,
		headerReader,
		addresses.EdgeChallengeManager,
	)
	require.NoError(t, err)
	return chain, accs, addresses, backend, headerReader
}
