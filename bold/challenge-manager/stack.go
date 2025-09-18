// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package challengemanager

import (
	"time"

	"github.com/ccoveille/go-safecast"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/bold/api/backend"
	"github.com/offchainlabs/nitro/bold/api/db"
	"github.com/offchainlabs/nitro/bold/api/server"
	"github.com/offchainlabs/nitro/bold/assertions"
	"github.com/offchainlabs/nitro/bold/chain-abstraction"
	"github.com/offchainlabs/nitro/bold/challenge-manager/chain-watcher"
	"github.com/offchainlabs/nitro/bold/challenge-manager/types"
	"github.com/offchainlabs/nitro/bold/layer2-state-provider"
)

type stackParams struct {
	mode                                types.Mode
	name                                string
	pollInterval                        time.Duration
	postInterval                        time.Duration
	confInterval                        time.Duration
	avgBlockTime                        time.Duration
	minGapToParent                      time.Duration
	trackChallengeParentAssertionHashes []protocol.AssertionHash
	apiAddr                             string
	apiDBPath                           string
	headerProvider                      HeaderProvider
	enableFastConfirmation              bool
	assertionManagerOverride            *assertions.Manager
	maxGetLogBlocks                     int64
	delegatedStaking                    bool
	autoDeposit                         bool
	autoAllowanceApproval               bool
}

var defaultStackParams = stackParams{
	mode:                                types.MakeMode,
	name:                                "unnamed-challenge-manager",
	pollInterval:                        time.Minute,
	postInterval:                        time.Hour,
	confInterval:                        time.Second * 10,
	avgBlockTime:                        time.Second * 12,
	minGapToParent:                      time.Minute * 10,
	trackChallengeParentAssertionHashes: nil,
	apiAddr:                             "",
	apiDBPath:                           "",
	headerProvider:                      nil,
	enableFastConfirmation:              false,
	assertionManagerOverride:            nil,
	maxGetLogBlocks:                     1000,
	delegatedStaking:                    false,
	autoDeposit:                         true,
	autoAllowanceApproval:               true,
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

// StackWithMinimumGapToParentAssertion sets the minimum gap to parent assertion creation time
// of the challenge manager.
func StackWithMinimumGapToParentAssertion(interval time.Duration) StackOpt {
	return func(p *stackParams) {
		p.minGapToParent = interval
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

// StackWithSyncMaxGetLogBlocks specifies the max size chunks of blocks to use when using get logs rpc for
// when syncing the chain watcher.
func StackWithSyncMaxGetLogBlocks(maxGetLog int64) StackOpt {
	return func(p *stackParams) {
		p.maxGetLogBlocks = maxGetLog
	}
}

// StackWithDelegatedStaking specifies that the challenge manager will call
// the `newStake` function in the rollup contract on startup to await funding from another account
// such that it becomes a delegated staker.
func StackWithDelegatedStaking() StackOpt {
	return func(p *stackParams) {
		p.delegatedStaking = true
	}
}

// StackWithoutAutoDeposit specifies that the software will not call
// the stake token's `deposit` function on startup to fund the account.
func StackWithoutAutoDeposit() StackOpt {
	return func(p *stackParams) {
		p.autoDeposit = false
	}
}

// StackWithoutAutoAllowanceApproval specifies that the software will not call
// the stake token's `increaseAllowance` function on startup to approve allowance spending for
// the rollup and challenge manager contracts.
func StackWithoutAutoAllowanceApproval() StackOpt {
	return func(p *stackParams) {
		p.autoAllowanceApproval = false
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
	maxGetLogBlocks, err := safecast.ToUint64(params.maxGetLogBlocks)
	if err != nil {
		return nil, err
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
		maxGetLogBlocks,
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
			assertions.WithMinimumGapToParentAssertion(params.minGapToParent),
			assertions.WithMaxGetLogBlocks(maxGetLogBlocks),
		}
		if apiDB != nil {
			amOpts = append(amOpts, assertions.WithAPIDB(apiDB))
		}
		if params.enableFastConfirmation {
			amOpts = append(amOpts, assertions.WithFastConfirmation())
		}
		if params.delegatedStaking {
			amOpts = append(amOpts, assertions.WithDelegatedStaking())
		}
		if !params.autoDeposit {
			amOpts = append(amOpts, assertions.WithoutAutoDeposit())
		}
		if !params.autoAllowanceApproval {
			amOpts = append(amOpts, assertions.WithoutAutoAllowanceApproval())
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
