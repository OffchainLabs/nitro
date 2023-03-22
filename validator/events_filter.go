package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/OffchainLabs/challenge-protocol-v2/solgen/go/challengeV2gen"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// Subscribes to events fired by the rollup contracts in order to listen to
// challenge start events from the protocol.
// TODO: Brittle - should be based on querying the chain instead.
func (v *Validator) handleChallengeEvents(ctx context.Context) {
	challengeCreatedChan := make(chan *challengeV2gen.ChallengeManagerImplChallengeCreated, 1)
	chalSub, err := v.chalManager.WatchChallengeCreated(&bind.WatchOpts{}, challengeCreatedChan)
	if err != nil {
		log.Error(err)
		return
	}
	defer chalSub.Unsubscribe()

	for {
		select {
		case err := <-chalSub.Err():
			log.Fatal(err)
		case chalCreated := <-challengeCreatedChan:
			var challenge protocol.Challenge
			if err := v.chain.Call(func(tx protocol.ActiveTx) error {
				manager, err := v.chain.CurrentChallengeManager(ctx, tx)
				if err != nil {
					return err
				}
				retrieved, err := manager.GetChallenge(ctx, tx, chalCreated.ChallengeId)
				if err != nil {
					return err
				}
				if retrieved.IsNone() {
					return fmt.Errorf("no challenge with id %#x", chalCreated.ChallengeId)
				}
				challenge = retrieved.Unwrap()
				return nil
			}); err != nil {
				log.Error(err)
				continue
			}
			// Ignore challenges from self.
			if isFromSelf(v.address, challenge.Challenger()) {
				continue
			}
			if err := v.onChallengeStarted(ctx, challenge); err != nil {
				log.Error(err)
			}
		}
	}
}

func (v *Validator) handleAssertions(ctx context.Context) time.Duration {
	var numberOfAssertions uint64
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		retrieved, err := v.chain.NumAssertions(ctx, tx)
		if err != nil {
			return err
		}
		numberOfAssertions = retrieved
		return nil
	}); err != nil {
		log.Error(err)
		return v.newAssertionCheckInterval
	}
	var latestConfirmedAssertion uint64
	if err := v.chain.Call(func(tx protocol.ActiveTx) error {
		retrieved, err := v.chain.LatestConfirmed(ctx, tx)
		if err != nil {
			return err
		}
		latestConfirmedAssertion = uint64(retrieved.SeqNum())
		return nil
	}); err != nil {
		log.Error(err)
		return v.newAssertionCheckInterval
	}

	for i := latestConfirmedAssertion; i < numberOfAssertions; i++ {
		v.assertionsLock.RLock()
		_, ok := v.assertions[protocol.AssertionSequenceNumber(i)]
		v.assertionsLock.RUnlock()
		if ok {
			continue
		}
		var assertion protocol.Assertion
		if err := v.chain.Call(func(tx protocol.ActiveTx) error {
			retrieved, err := v.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(i))
			if err != nil {
				return err
			}
			assertion = retrieved
			return nil
		}); err != nil {
			log.Error(err)
			continue
		}
		v.assertions[assertion.SeqNum()] = assertion
		selfStakedAssertion, err := v.rollup.AssertionHasStaker(&bind.CallOpts{}, i, v.address)
		if err != nil {
			log.Error(err)
			continue
		}
		// Ignore assertions from self.
		if selfStakedAssertion {
			continue
		}
		if err := v.onLeafCreated(ctx, assertion); err != nil {
			log.Error(err)
		}
	}
	return v.newAssertionCheckInterval
}
