package validator

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	solimpl "github.com/OffchainLabs/challenge-protocol-v2/protocol/sol-implementation"
	"github.com/OffchainLabs/challenge-protocol-v2/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Processes new challenge creation events from the protocol that were not created by self.
// This will fetch the challenge, its parent assertion, and create a challenge leaf that is
// relevant towards resolving the challenge. We then spawn a challenge tracker in the background.
func (v *Validator) onChallengeStarted(
	ctx context.Context, ev protocol.Challenge,
) error {
	var challengedAssertion protocol.Assertion
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		rootAssertion, err := ev.RootAssertion(ctx, tx)
		if err != nil {
			return err
		}
		challengedAssertion = rootAssertion
		return nil
	}); err != nil {
		return err
	}

	challenge, err := v.fetchProtocolChallenge(
		ctx,
		challengedAssertion.SeqNum(),
	)
	if err != nil {
		return err
	}

	// We then add a challenge vertex to the challenge.
	challengeVertex, err := v.addChallengeVertex(ctx, challenge)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists on challenge with id %#x",
				ev.Id(),
			)
			return nil
		}
		return err
	}

	challengerName := "unknown-name"
	staker := challengeVertex.MiniStaker()
	if name, ok := v.knownValidatorNames[staker]; ok {
		challengerName = name
	}
	log.WithFields(logrus.Fields{
		"name":                 v.name,
		"challenger":           challengerName,
		"challengingAssertion": fmt.Sprintf("%d", challengedAssertion.SeqNum()),
	}).Warn("Received challenge for a created leaf, added own leaf with history commitment")

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	return nil
}

// Initiates a challenge on an assertion added to the protocol by finding its parent assertion
// and starting a challenge transaction. If the challenge creation is successful, we add a leaf
// with an associated history commitment to it and spawn a challenge tracker in the background.
func (v *Validator) challengeAssertion(ctx context.Context, assertion protocol.Assertion) error {
	var challenge protocol.Challenge
	var err error
	challenge, err = v.submitProtocolChallenge(ctx, assertion.PrevSeqNum())
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			existingChallenge, fetchErr := v.fetchProtocolChallenge(ctx, assertion.PrevSeqNum())
			if fetchErr != nil {
				return fetchErr
			}
			challenge = existingChallenge
		} else {
			return err
		}
	}

	// We then add a challenge vertex to the challenge.
	challengeVertex, err := v.addChallengeVertex(ctx, challenge)
	if err != nil {
		if errors.Is(err, solimpl.ErrAlreadyExists) {
			// TODO: Should we return error here instead of a log and nil?
			log.Infof(
				"Attempted to add a challenge leaf that already exists with challenge hash %#x",
				challenge.Id(),
			)
			return nil
		}
		return err
	}

	// Start tracking the challenge.
	go newVertexTracker(v.timeRef, v.challengeVertexWakeInterval, challenge, challengeVertex, v.chain, v.stateManager, v.name, v.address).track(ctx)

	logFields := logrus.Fields{}
	logFields["name"] = v.name
	logFields["parentAssertionSeqNum"] = assertion.PrevSeqNum()
	log.WithFields(logFields).Info("Successfully created challenge and added leaf, now tracking events")
	return nil
}

func (v *Validator) addChallengeVertex(
	ctx context.Context,
	challenge protocol.Challenge,
) (protocol.ChallengeVertex, error) {
	latestValidAssertionSeq, err := v.findLatestValidAssertion(ctx)
	if err != nil {
		return nil, err
	}
	var createdVertex protocol.ChallengeVertex
	if err := v.chain.Tx(func(tx protocol.ActiveTx) error {
		assertion, err := v.chain.AssertionBySequenceNum(ctx, tx, latestValidAssertionSeq)
		if err != nil {
			return err
		}
		historyCommit, err := v.stateManager.HistoryCommitmentUpTo(ctx, assertion.Height())
		if err != nil {
			return err
		}
		leaf, err := challenge.AddBlockChallengeLeaf(ctx, tx, assertion, historyCommit)
		if err != nil {
			return errors.Wrap(err, "could not add challenge leaf to challenge")
		}
		createdVertex = leaf
		return nil
	}); err != nil {
		return nil, err
	}
	return createdVertex, nil
}

func (v *Validator) submitProtocolChallenge(
	ctx context.Context,
	parentAssertionSeqNum protocol.AssertionSequenceNumber,
) (protocol.Challenge, error) {
	var challenge protocol.Challenge
	var err error
	if err = v.chain.Tx(func(tx protocol.ActiveTx) error {
		challenge, err = v.chain.CreateSuccessionChallenge(ctx, tx, parentAssertionSeqNum)
		if err != nil {
			return errors.Wrap(err, "could not submit challenge")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return challenge, nil
}

// Tries to retrieve a challenge from the protocol on-chain
// based on the parent assertion's state commitment hash.
func (v *Validator) fetchProtocolChallenge(
	ctx context.Context,
	parentAssertionSeqNum protocol.AssertionSequenceNumber,
) (protocol.Challenge, error) {
	var err error
	var challenge util.Option[protocol.Challenge]
	if err = v.chain.Call(func(tx protocol.ActiveTx) error {
		assertionId, err2 := v.chain.GetAssertionId(ctx, tx, parentAssertionSeqNum)
		if err2 != nil {
			return err2
		}
		manager, err3 := v.chain.CurrentChallengeManager(ctx, tx)
		if err3 != nil {
			return err3
		}
		chalHash, err4 := manager.CalculateChallengeHash(ctx, tx, common.Hash(assertionId), protocol.BlockChallenge)
		if err4 != nil {
			return err4
		}
		challenge, err = manager.GetChallenge(
			ctx,
			tx,
			chalHash,
		)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "could not get challenge from protocol")
	}
	if challenge.IsNone() {
		return nil, errors.New("got nil challenge from protocol")
	}
	return challenge.Unwrap(), nil
}
