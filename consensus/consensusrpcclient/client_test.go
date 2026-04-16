// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package consensusrpcclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/consensus"
	"github.com/offchainlabs/nitro/consensus/consensusrpcserver"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	utilrpc "github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// mockConsensusService implements consensus.FullConsensusClient for testing.
type mockConsensusService struct {
	err error
}

func (m *mockConsensusService) GetL1Confirmations(_ arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](0, m.err)
}

func (m *mockConsensusService) FindBatchContainingMessage(_ arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](0, m.err)
}

func (m *mockConsensusService) BlockMetadataAtMessageIndex(_ arbutil.MessageIndex) containers.PromiseInterface[common.BlockMetadata] {
	return containers.NewReadyPromise[common.BlockMetadata](nil, m.err)
}

func (m *mockConsensusService) WriteMessageFromSequencer(_ arbutil.MessageIndex, _ arbostypes.MessageWithMetadata, _ execution.MessageResult, _ common.BlockMetadata) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise[struct{}](struct{}{}, m.err)
}

func (m *mockConsensusService) ExpectChosenSequencer() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise[struct{}](struct{}{}, m.err)
}

func createMockConsensusNode(t *testing.T, errToReturn error) *node.Node {
	t.Helper()
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{consensus.RPCNamespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	testhelpers.RequireImpl(t, err)

	stack.RegisterAPIs([]rpc.API{{
		Namespace: consensus.RPCNamespace,
		Service:   consensusrpcserver.NewConsensusRPCServer(&mockConsensusService{err: errToReturn}),
		Public:    true,
	}})

	testhelpers.RequireImpl(t, stack.Start())
	t.Cleanup(func() { _ = stack.Close() })
	return stack
}

func TestConsensusClientErrorHandling(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	tests := []struct {
		name        string
		serverErr   error
		expectedErr error
	}{
		{
			name:        "ErrSequencerInsertLockTaken",
			serverErr:   execution.ErrSequencerInsertLockTaken,
			expectedErr: execution.ErrSequencerInsertLockTaken,
		},
		{
			name:        "ErrSequencerInsertLockTaken wrapped",
			serverErr:   fmt.Errorf("consensus context: %w", execution.ErrSequencerInsertLockTaken),
			expectedErr: execution.ErrSequencerInsertLockTaken,
		},
		{
			name:        "ErrRetrySequencer",
			serverErr:   execution.ErrRetrySequencer,
			expectedErr: execution.ErrRetrySequencer,
		},
		{
			name:        "ErrRetrySequencer wrapped",
			serverErr:   fmt.Errorf("consensus context: %w", execution.ErrRetrySequencer),
			expectedErr: execution.ErrRetrySequencer,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stack := createMockConsensusNode(t, tc.serverErr)

			config := &utilrpc.ClientConfig{
				URL:     "self",
				Timeout: 5 * time.Second,
			}
			testhelpers.RequireImpl(t, config.Validate())

			client := NewConsensusRPCClient(func() *utilrpc.ClientConfig { return config }, stack)
			testhelpers.RequireImpl(t, client.Start(ctx))
			defer client.StopAndWait()

			_, err := client.ExpectChosenSequencer().Await(ctx)

			if err == nil {
				t.Fatal("expected an error from server, got nil")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

// TestConsensusClientErrorNoFalsePositives verifies that a plain server error
// (arriving with the default JSON-RPC code -32000) does not match any sentinel.
func TestConsensusClientErrorNoFalsePositives(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	stack := createMockConsensusNode(t, errors.New("some unrelated failure"))

	config := &utilrpc.ClientConfig{
		URL:     "self",
		Timeout: 5 * time.Second,
	}
	testhelpers.RequireImpl(t, config.Validate())

	client := NewConsensusRPCClient(func() *utilrpc.ClientConfig { return config }, stack)
	testhelpers.RequireImpl(t, client.Start(ctx))
	defer client.StopAndWait()

	_, err := client.ExpectChosenSequencer().Await(ctx)
	if err == nil {
		t.Fatal("expected an error from server, got nil")
	}

	allSentinels := []error{
		execution.ErrResultNotFound,
		execution.ErrRetrySequencer,
		execution.ErrSequencerInsertLockTaken,
	}
	for _, sentinel := range allSentinels {
		if errors.Is(err, sentinel) {
			t.Errorf("plain error should not match sentinel %v", sentinel)
		}
	}
}
