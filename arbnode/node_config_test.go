// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbnode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/core/rawdb"

	"github.com/offchainlabs/nitro/cmd/chaininfo"
)

const melRPCTestURL = "http://localhost:1234"

// TestConfigValidateMEL covers the MEL cross-field assertions in Config.Validate: each case starts
// from a config that validates cleanly and flips only the fields under test.
func TestConfigValidateMEL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		mutate  func(c *Config)
		wantErr string // empty means the config must validate
	}{
		{
			name:   "default has no MEL at all",
			mutate: func(c *Config) {},
		},
		{
			name: "native mel",
			mutate: func(c *Config) {
				c.MessageExtraction.Enable = true
			},
		},
		{
			name: "mel over rpc",
			mutate: func(c *Config) {
				c.MELRPCClient.URL = melRPCTestURL
				c.RPCServer.Enable = true
			},
		},
		{
			name: "both native mel and mel over rpc",
			mutate: func(c *Config) {
				c.MessageExtraction.Enable = true
				c.MELRPCClient.URL = melRPCTestURL
				c.RPCServer.Enable = true
			},
			wantErr: "cannot enable both message-extraction (native MEL) and mel-rpc-client",
		},
		{
			// The remote provider pushes extracted messages back to this node's
			// nitromelconsumer sink, which only exists when the RPC server is up.
			name: "mel over rpc without rpc server",
			mutate: func(c *Config) {
				c.MELRPCClient.URL = melRPCTestURL
				c.RPCServer.Enable = false
			},
			wantErr: "mel-rpc-client requires rpc-server.enable",
		},
		{
			name: "native mel with always-fallback-to-parent-chain-da",
			mutate: func(c *Config) {
				c.MessageExtraction.Enable = true
				c.Dangerous.AlwaysFallbackToParentChainDA = true
			},
			wantErr: "always-fallback-to-parent-chain-da is not supported with message-extraction.enable=true",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Start from the production defaults rather than a ConfigDefault*Test variant: the
			// test variants only tune timings and disable services so real nodes boot quickly,
			// which is irrelevant to Validate and would make this baseline drift with unrelated
			// harness tweaks.
			config := ConfigDefault
			tc.mutate(&config)

			err := config.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

type staticConfigFetcher struct{ config *Config }

func (f *staticConfigFetcher) Get() *Config          { return f.config }
func (f *staticConfigFetcher) Start(context.Context) {}
func (f *staticConfigFetcher) StopAndWait()          {}
func (f *staticConfigFetcher) Started() bool         { return true }

// TestGetInboxTrackerAndReaderMEL pins the wiring decision that both the legacy inbox tracker and
// reader are skipped whenever messages come from MEL, natively extracted or consumed over RPC.
// createNodeImpl rejects a node that ends up with both, so this is the invariant feeding that check.
func TestGetInboxTrackerAndReaderMEL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		mutate    func(c *Config)
		wantInbox bool
	}{
		{
			name:      "no mel keeps the inbox tracker and reader",
			mutate:    func(c *Config) {},
			wantInbox: true,
		},
		{
			name: "native mel disables the inbox tracker and reader",
			mutate: func(c *Config) {
				c.MessageExtraction.Enable = true
			},
		},
		{
			name: "mel over rpc disables the inbox tracker and reader",
			mutate: func(c *Config) {
				c.MELRPCClient.URL = melRPCTestURL
				c.RPCServer.Enable = true
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			config := ConfigDefault
			tc.mutate(&config)
			require.NoError(t, config.Validate())

			// Only the arguments the MEL branch can reach need to be real: it returns before
			// touching the parent chain, and the two constructors it calls only populate structs.
			tracker, reader, err := getInboxTrackerAndReader(
				&config,
				rawdb.NewMemoryDatabase(),
				nil, // txStreamer
				nil, // dapReaders
				&staticConfigFetcher{config: &config},
				nil, // l1client
				nil, // l1Reader
				&chaininfo.RollupAddresses{DeployedAt: 0},
				nil, // delayedBridge
				nil, // sequencerInbox
				nil, // fatalErrChan
			)
			require.NoError(t, err)

			if tc.wantInbox {
				require.NotNil(t, tracker)
				require.NotNil(t, reader)
			} else {
				require.Nil(t, tracker)
				require.Nil(t, reader)
			}
		})
	}
}
