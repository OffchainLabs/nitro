// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE

package assertions

import (
	"context"
	"time"

	protocol "github.com/OffchainLabs/bold/chain-abstraction"
	solimpl "github.com/OffchainLabs/bold/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/bold/layer2-state-provider"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

// Poster defines a service which frequently posts assertions onchain at some intervals,
// given the latest execution state it can find in its local state manager.
type Poster struct {
	validatorName string
	chain         protocol.Protocol
	stateManager  l2stateprovider.Provider
	postInterval  time.Duration
}

// NewPoster creates a poster from required dependencies.
func NewPoster(
	chain protocol.Protocol,
	stateManager l2stateprovider.Provider,
	validatorName string,
	postInterval time.Duration,
) *Poster {
	return &Poster{
		chain:         chain,
		stateManager:  stateManager,
		validatorName: validatorName,
		postInterval:  postInterval,
	}
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
		return nil, err
	}
	parentAssertionCreationInfo, err := p.chain.ReadAssertionCreationInfo(ctx, parentAssertionSeq)
	if err != nil {
		return nil, err
	}
	if !parentAssertionCreationInfo.InboxMaxCount.IsUint64() {
		return nil, errors.New("inbox max count not a uint64")
	}
	prevInboxMaxCount := parentAssertionCreationInfo.InboxMaxCount.Uint64()
	newState, err := p.stateManager.ExecutionStateAtMessageNumber(ctx, prevInboxMaxCount)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get execution state at message count %d", prevInboxMaxCount)
	}
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
	srvlog.Info("Submitted latest L2 state claim as an assertion to L1", log.Ctx{"validatorName": p.validatorName})

	return assertion, nil
}

// Finds the latest valid assertion sequence num a validator should build their new leaves upon. This walks
// down from the number of assertions in the protocol down until it finds
// an assertion that we have a state commitment for.
func (p *Poster) findLatestValidAssertion(ctx context.Context) (protocol.AssertionHash, error) {
	latestConfirmed, err := p.chain.LatestConfirmed(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	latestCreated, err := p.chain.LatestCreatedAssertion(ctx)
	if err != nil {
		return protocol.AssertionHash{}, err
	}
	if latestConfirmed == latestCreated {
		return latestConfirmed.Id(), nil
	}
	curr := latestCreated
	for curr.Id() != latestConfirmed.Id() {
		info, err := p.chain.ReadAssertionCreationInfo(ctx, curr.Id())
		if err != nil {
			return protocol.AssertionHash{}, err
		}
		_, err = p.stateManager.ExecutionStateMsgCount(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState))
		switch {
		case errors.Is(err, l2stateprovider.ErrNoExecutionState):
			prevId, prevErr := curr.PrevId(ctx)
			if prevErr != nil {
				return protocol.AssertionHash{}, prevErr
			}
			prev, getErr := p.chain.GetAssertion(ctx, prevId)
			if getErr != nil {
				return protocol.AssertionHash{}, getErr
			}
			curr = prev
		case err != nil:
			return protocol.AssertionHash{}, nil
		default:
			return curr.Id(), nil
		}
	}
	return latestConfirmed.Id(), nil
}
