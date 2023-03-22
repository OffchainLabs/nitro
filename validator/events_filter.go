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
	v.assertionsLock.RLock()
	currentNumberOfAssertions := uint64(len(v.assertions))
	v.assertionsLock.RUnlock()

	for i := currentNumberOfAssertions; i < numberOfAssertions; i++ {
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
		assertionNum, err := v.rollup.LatestStakedAssertion(&bind.CallOpts{}, v.address)
		if err != nil {
			log.Error(err)
			continue
		}
		// Ignore assertions from self.
		if assertionNum == uint64(assertion.SeqNum()) {
			continue
		}
		if err := v.onLeafCreated(ctx, assertion); err != nil {
			log.Error(err)
		}
	}
	return v.newAssertionCheckInterval
}
