package assertions

import (
	"context"
	"time"

	protocol "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/chain-abstraction/sol-implementation"
	l2stateprovider "github.com/OffchainLabs/challenge-protocol-v2/layer2-state-provider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	if _, err := p.PostLatestAssertion(ctx); err != nil {
		log.WithError(err).Error("Could not submit latest assertion to L1")
	}
	ticker := time.NewTicker(p.postInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := p.PostLatestAssertion(ctx); err != nil {
				log.WithError(err).Error("Could not submit latest assertion to L1")
			}
		case <-ctx.Done():
			return
		}
	}
}

// Posts the latest claim of the Node's L2 state as an assertion to the L1 protocol smart contracts.
// TODO: Include leaf creation validity conditions which are more complex than this.
// For example, a validator must include messages from the inbox that were not included
// by the last validator in the last leaf's creation.
func (p *Poster) PostLatestAssertion(ctx context.Context) (protocol.Assertion, error) {
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
	// TODO: this should really only go up to the prevInboxMaxCount batch state
	newState, err := p.stateManager.LatestExecutionState(ctx)
	if err != nil {
		return nil, err
	}
	assertion, err := p.chain.CreateAssertion(
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
	logFields := logrus.Fields{
		"validatorName": p.validatorName,
	}
	log.WithFields(logFields).Info("Submitted latest L2 state claim as an assertion to L1")

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
		_, hasState := p.stateManager.ExecutionStateBlockHeight(ctx, protocol.GoExecutionStateFromSolidity(info.AfterState))
		if hasState {
			return curr.Id(), nil
		}
		prevId, err := curr.PrevId(ctx)
		if err != nil {
			return protocol.AssertionHash{}, err
		}
		prev, err := p.chain.GetAssertion(ctx, prevId)
		if err != nil {
			return protocol.AssertionHash{}, err
		}
		curr = prev
	}
	return latestConfirmed.Id(), nil
}
