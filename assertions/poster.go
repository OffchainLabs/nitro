// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions

import (
	"context"
	"fmt"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	"github.com/OffchainLabs/bold/containers"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// Poster defines a service which frequently posts assertions onchain at some intervals,
// given the latest execution state it can find in its local state manager.
type Poster struct {
	validatorName string
	chain         protocol.Protocol
	stateManager  l2stateprovider.ExecutionProvider
	postInterval  time.Duration
}

// NewPoster creates a poster from required dependencies.
func NewPoster(
	chain protocol.Protocol,
	stateManager l2stateprovider.ExecutionProvider,
	validatorName string,
	postInterval time.Duration,
) (*Poster, error) {
	if postInterval == 0 {
		return nil, errors.New("assertion posting interval must be greater than 0")
	}
	return &Poster{
		chain:         chain,
		stateManager:  stateManager,
		validatorName: validatorName,
		postInterval:  postInterval,
	}, nil
}

func (p *Poster) Start(ctx context.Context) {
	if _, err := p.PostAssertion(ctx); err != nil {
		srvlog.Error("Could not submit latest assertion to L1", log.Ctx{"err": err})
	}
	ticker := time.NewTicker(p.postInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := p.PostAssertion(ctx); err != nil {
				srvlog.Error("Could not submit latest assertion to L1", log.Ctx{"err": err})
			}
		case <-ctx.Done():
			return
		}
	}
}

// PostAssertion differs depending on whether or not the validator is currently staked.
func (p *Poster) PostAssertion(ctx context.Context) (protocol.Assertion, error) {
	staked, err := p.chain.IsStaked(ctx)
	if err != nil {
		return nil, err
	}
	if staked {
		return p.PostAssertionAndMoveStake(ctx)
	}
	return p.PostAssertionAndNewStake(ctx)
}

// PostAssertionAndNewStake posts the latest assertion and adds a new stake on it.
func (p *Poster) PostAssertionAndNewStake(ctx context.Context) (protocol.Assertion, error) {
	return p.postAssertionImpl(ctx, p.chain.NewStakeOnNewAssertion)
}

// PostAssertionAndMoveStake posts the latest assertion and moves an existing stake to it.
func (p *Poster) PostAssertionAndMoveStake(ctx context.Context) (protocol.Assertion, error) {
	return p.postAssertionImpl(ctx, p.chain.StakeOnNewAssertion)
}

func (p *Poster) postAssertionImpl(
	ctx context.Context,
	submitFn func(
		ctx context.Context,
		parentCreationInfo *protocol.AssertionCreatedInfo,
		newState *protocol.ExecutionState,
	) (protocol.Assertion, error),
) (protocol.Assertion, error) {
	// Ensure that we only build on a valid parent from this validator's perspective.
	// the validator should also have ready access to historical commitments to make sure it can select
	// the valid parent based on its commitment state root.
	parentAssertionSeq, err := p.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not find latest valid assertion")
	}
	parentAssertionCreationInfo, err := p.chain.ReadAssertionCreationInfo(ctx, parentAssertionSeq)
	if err != nil {
		return nil, err
	}
	if !parentAssertionCreationInfo.InboxMaxCount.IsUint64() {
		return nil, errors.New("inbox max count not a uint64")
	}
	// The parent assertion tells us what the next posted assertion's batch should be.
	// We read this value and use it to compute the required execution state we must post.
	batchCount := parentAssertionCreationInfo.InboxMaxCount.Uint64()
	newState, err := p.stateManager.ExecutionStateAfterBatchCount(ctx, batchCount)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get execution state at message count %d", batchCount)
	}
	srvlog.Info("Required batch for assertion and retrieved state", log.Ctx{"batch": batchCount, "newState": fmt.Sprintf("%+v", newState)})
	assertion, err := submitFn(
		ctx,
		parentAssertionCreationInfo,
		newState,
	)
	switch {
	case errors.Is(err, solimpl.ErrAlreadyExists):
		return nil, errors.Wrap(err, "assertion already exists, was unable to post")
	case err != nil:
		return nil, err
	}
	srvlog.Info("Submitted latest L2 state claim as an assertion to L1", log.Ctx{
		"validatorName":         p.validatorName,
		"layer2BlockHash":       containers.Trunc(newState.GlobalState.BlockHash[:]),
		"requiredInboxMaxCount": batchCount,
		"postedExectionState":   fmt.Sprintf("%+v", newState),
	})

	return assertion, nil
}

// Finds the latest valid assertion sequence num a validator should build their new leaves upon.
// It retrieves the latest assertion hashes posted to the rollup contract since the last confirmed assertion block.
// This walks down the list of assertions in the protocol down until it finds
// the latest assertion that we have a state commitment for.
func (p *Poster) findLatestValidAssertion(ctx context.Context) (protocol.AssertionHash, error) {
	latestCreatedAssertionHashes, err := p.chain.LatestCreatedAssertionHashes(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	// Loop over latestCreatedAssertionHashes in reverse order to find the latest valid assertion.
	for i := len(latestCreatedAssertionHashes) - 1; i >= 0; i-- {
		var info *protocol.AssertionCreatedInfo
		info, err = p.chain.ReadAssertionCreationInfo(ctx, latestCreatedAssertionHashes[i])
		if err != nil {
			return protocol.AssertionHash{}, err
		}
		if err = p.stateManager.AgreesWithExecutionState(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState)); err == nil {
			return latestCreatedAssertionHashes[i], nil
		}
	}
	latestConfirmed, err := p.chain.LatestConfirmed(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	return latestConfirmed.Id(), nil
}
