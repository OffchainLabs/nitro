package validator

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

func (v *Validator) pollForChallenges(ctx context.Context) {
	ticker := time.NewTicker(v.newChallengeCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			v.assertionsLock.RLock()
			assertions := v.assertions
			v.assertionsLock.RUnlock()
			for assertionId := range assertions {
				challenge, err := v.chain.HasBlockChallenge(ctx, assertionId)
				if err != nil {
					log.Error(err)
					continue
				}
				v.challengesLock.RLock()
				_, ok := v.challenges[challenge.Id()]
				v.challengesLock.RUnlock()
				if ok {
					continue
				}
				v.challengesLock.Lock()
				v.challenges[challenge.Id()] = challenge
				v.challengesLock.Unlock()
				// Ignore challenges from self.
				challenger := challenge.Challenger()
				if isFromSelf(v.address, challenger) {
					continue
				}
				if err := v.onChallengeStarted(ctx, challenge); err != nil {
					log.Error(err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *Validator) pollForAssertions(ctx context.Context) {
	ticker := time.NewTicker(v.newAssertionCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			numberOfAssertions, err := v.chain.NumAssertions(ctx)
			if err != nil {
				log.Error(err)
				continue
			}
			latestConfirmedAssertion, err := v.chain.LatestConfirmed(ctx)
			if err != nil {
				log.Error(err)
				continue
			}

			for i := uint64(latestConfirmedAssertion.SeqNum()); i < numberOfAssertions; i++ {
				v.assertionsLock.RLock()
				_, ok := v.assertions[protocol.AssertionSequenceNumber(i)]
				v.assertionsLock.RUnlock()
				if ok {
					continue
				}
				assertion, err := v.chain.AssertionBySequenceNum(ctx, protocol.AssertionSequenceNumber(i))
				if err != nil {
					log.Error(err)
					continue
				}
				v.assertionsLock.Lock()
				v.assertions[assertion.SeqNum()] = assertion
				v.assertionsLock.Unlock()
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
		case <-ctx.Done():
			return
		}
	}
}
