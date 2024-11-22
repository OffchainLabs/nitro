// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/bold/blob/main/LICENSE.md

package challengemanager

import (
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/bold/api/backend"
	"github.com/offchainlabs/bold/api/db"
	"github.com/offchainlabs/bold/api/server"
	"github.com/offchainlabs/bold/assertions"
	protocol "github.com/offchainlabs/bold/chain-abstraction"
	watcher "github.com/offchainlabs/bold/challenge-manager/chain-watcher"
	"github.com/offchainlabs/bold/challenge-manager/types"
	l2stateprovider "github.com/offchainlabs/bold/layer2-state-provider"
)

type stackParams struct {
	mode                                types.Mode
	name                                string
	pollInterval                        time.Duration
	postInterval                        time.Duration
	confInterval                        time.Duration
	avgBlockTime                        time.Duration
	trackChallengeParentAssertionHashes []protocol.AssertionHash
	apiAddr                             string
	apiDBPath                           string
	headerProvider                      HeaderProvider
	enableFastConfirmation              bool
	assertionManagerOverride            *assertions.Manager
}

var defaultStackParams = stackParams{
	mode:                                types.MakeMode,
	name:                                "unnamed-challenge-manager",
	pollInterval:                        time.Minute,
	postInterval:                        time.Hour,
	confInterval:                        time.Second * 10,
	avgBlockTime:                        time.Second * 12,
	trackChallengeParentAssertionHashes: nil,
	apiAddr:                             "",
	apiDBPath:                           "",
	headerProvider:                      nil,
	enableFastConfirmation:              false,
	assertionManagerOverride:            nil,
}

// StackOpt is a functional option to configure the stack.
type StackOpt func(*stackParams)

// WithMode sets the mode of the challenge manager.
func StackWithMode(mode types.Mode) StackOpt {
	return func(p *stackParams) {
		p.mode = mode
	}
}

// WithName sets the name of the challenge manager.
func StackWithName(name string) StackOpt {
	return func(p *stackParams) {
		p.name = name
	}
}

// WithPollingInterval sets the polling interval of the challenge manager.
func StackWithPollingInterval(interval time.Duration) StackOpt {
	return func(p *stackParams) {
		p.pollInterval = interval
	}
}

// WithPostingInterval sets the posting interval of the challenge manager.
func StackWithPostingInterval(interval time.Duration) StackOpt {
	return func(p *stackParams) {
		p.postInterval = interval
	}
}

// WithConfirmationInterval sets the confirmation interval of the challenge
// manager.
func StackWithConfirmationInterval(interval time.Duration) StackOpt {
	return func(p *stackParams) {
		p.confInterval = interval
	}
}

// WithAverageBlockCreationTime sets the average block creation time of the
// challenge manager.
func StackWithAverageBlockCreationTime(interval time.Duration) StackOpt {
	return func(p *stackParams) {
		p.avgBlockTime = interval
	}
}

// WithTrackChallengeParentAssertionHashes sets the track challenge parent
// assertion hashes of the challenge manager.
func StackWithTrackChallengeParentAssertionHashes(hashes []string) StackOpt {
	return func(p *stackParams) {
		p.trackChallengeParentAssertionHashes = make([]protocol.AssertionHash, len(hashes))
		for i, h := range hashes {
			p.trackChallengeParentAssertionHashes[i] = protocol.AssertionHash{Hash: common.HexToHash(h)}
		}
	}
}

// WithAPIEnabled sets the API address and database path of the challenge
// manager.
func StackWithAPIEnabled(apiAddr, apiDBPath string) StackOpt {
	return func(p *stackParams) {
		p.apiAddr = apiAddr
		p.apiDBPath = apiDBPath
	}
}

// StackWithHeaderProvider sets the header provider of the challenge manager.
func StackWithHeaderProvider(hp HeaderProvider) StackOpt {
	return func(p *stackParams) {
		p.headerProvider = hp
	}
}

// WithFastConfirmationEnabled
func StackWithFastConfirmationEnabled() StackOpt {
	return func(p *stackParams) {
		p.enableFastConfirmation = true
	}
}

// OverrideAssertionManger can be used in tests to override the assertion
// manager.
func OverrideAssertionManager(asm *assertions.Manager) StackOpt {
	return func(p *stackParams) {
		p.assertionManagerOverride = asm
	}
}

// NewChallengeStack creates a new ChallengeManager and all of the dependencies
// wiring them together.
func NewChallengeStack(
	chain protocol.AssertionChain,
	provider l2stateprovider.Provider,
	opts ...StackOpt,
) (*Manager, error) {
	params := defaultStackParams
	for _, o := range opts {
		o(&params)
	}

	var err error
	// Create the api database.
	var apiDB db.Database
	if params.apiDBPath != "" {
		apiDB, err = db.NewDatabase(params.apiDBPath)
		if err != nil {
			return nil, err
		}
		provider.UpdateAPIDatabase(apiDB)
	}

	// Create the chain watcher.
	watcher, err := watcher.New(
		chain,
		provider,
		params.name,
		apiDB,
		params.confInterval,
		params.avgBlockTime,
		params.trackChallengeParentAssertionHashes,
	)
	if err != nil {
		return nil, err
	}

	// Create the api backend server.
	var api *server.Server
	if params.apiAddr != "" {
		bknd := backend.NewBackend(apiDB, chain, watcher)
		api, err = server.New(params.apiAddr, bknd)
		if err != nil {
			return nil, err
		}
	}

	// Create the assertions manager.
	var asm *assertions.Manager
	if params.assertionManagerOverride == nil {
		// Create the assertions manager.
		amOpts := []assertions.Opt{
			assertions.WithAverageBlockCreationTime(params.avgBlockTime),
			assertions.WithConfirmationInterval(params.confInterval),
			assertions.WithPollingInterval(params.pollInterval),
			assertions.WithPostingInterval(params.postInterval),
		}
		if apiDB != nil {
			amOpts = append(amOpts, assertions.WithAPIDB(apiDB))
		}
		if params.enableFastConfirmation {
			amOpts = append(amOpts, assertions.WithFastConfirmation())
		}
		asm, err = assertions.NewManager(
			chain,
			provider,
			params.name,
			params.mode,
			amOpts...,
		)
		if err != nil {
			return nil, err
		}
	} else {
		asm = params.assertionManagerOverride
	}

	// Create the challenge manager.
	cmOpts := []Opt{
		WithMode(params.mode),
		WithName(params.name),
	}
	if params.headerProvider != nil {
		cmOpts = append(cmOpts, WithHeaderProvider(params.headerProvider))
	}
	if params.apiAddr != "" {
		cmOpts = append(cmOpts, WithAPIServer(api))
	}
	return New(chain, provider, watcher, asm, cmOpts...)
}
