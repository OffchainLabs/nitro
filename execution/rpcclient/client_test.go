// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	utilrpc "github.com/offchainlabs/nitro/util/rpcclient"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// mockExecutionService implements a minimal execution RPC service for testing.
type mockExecutionService struct {
	err error
}

func (s *mockExecutionService) HeadMessageIndex(_ context.Context) (arbutil.MessageIndex, error) {
	return 0, s.err
}

func createMockExecutionNode(t *testing.T, errToReturn error) *node.Node {
	t.Helper()
	stackConf := node.DefaultConfig
	stackConf.HTTPPort = 0
	stackConf.DataDir = ""
	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 0
	stackConf.WSModules = []string{execution.RPCNamespace}
	stackConf.P2P.NoDiscovery = true
	stackConf.P2P.ListenAddr = ""

	stack, err := node.New(&stackConf)
	testhelpers.RequireImpl(t, err)

	stack.RegisterAPIs([]rpc.API{{
		Namespace: execution.RPCNamespace,
		Service:   &mockExecutionService{err: errToReturn},
		Public:    true,
	}})

	testhelpers.RequireImpl(t, stack.Start())
	t.Cleanup(func() { _ = stack.Close() })
	return stack
}

var allSentinels = []error{
	execution.ErrResultNotFound,
	execution.ErrRetrySequencer,
	execution.ErrSequencerInsertLockTaken,
}

func TestClientErrorHandling(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	tests := []struct {
		name        string
		serverErr   error
		expectedErr error
	}{
		{
			name:        "ResultNotFound",
			serverErr:   execution.ErrResultNotFound,
			expectedErr: execution.ErrResultNotFound,
		},
		{
			name:        "ResultNotFound wrapped",
			serverErr:   fmt.Errorf("execution context: %w", execution.ErrResultNotFound),
			expectedErr: execution.ErrResultNotFound,
		},
		{
			name:        "ErrRetrySequencer",
			serverErr:   execution.ErrRetrySequencer,
			expectedErr: execution.ErrRetrySequencer,
		},
		{
			name:        "ErrRetrySequencer wrapped",
			serverErr:   fmt.Errorf("rpc context: %w", execution.ErrRetrySequencer),
			expectedErr: execution.ErrRetrySequencer,
		},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stack := createMockExecutionNode(t, tc.serverErr)

			config := &utilrpc.ClientConfig{
				URL:     "self",
				Timeout: 5 * time.Second,
			}
			testhelpers.RequireImpl(t, config.Validate())

			client := NewClient(func() *utilrpc.ClientConfig { return config }, stack)
			testhelpers.RequireImpl(t, client.Start(ctx))
			defer client.StopAndWait()

			_, err := client.HeadMessageIndex().Await(ctx)

			if err == nil {
				t.Fatal("expected an error from server, got nil")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

// TestClientErrorNoFalsePositives verifies that a plain server error (which
// arrives with the default JSON-RPC code -32000) does not match any sentinel.
func TestClientErrorNoFalsePositives(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	stack := createMockExecutionNode(t, errors.New("some unrelated failure"))

	config := &utilrpc.ClientConfig{
		URL:     "self",
		Timeout: 5 * time.Second,
	}
	testhelpers.RequireImpl(t, config.Validate())

	client := NewClient(func() *utilrpc.ClientConfig { return config }, stack)
	testhelpers.RequireImpl(t, client.Start(ctx))
	defer client.StopAndWait()

	_, err := client.HeadMessageIndex().Await(ctx)
	if err == nil {
		t.Fatal("expected an error from server, got nil")
	}
	for _, sentinel := range allSentinels {
		if errors.Is(err, sentinel) {
			t.Errorf("plain error should not match sentinel %v", sentinel)
		}
	}
}
