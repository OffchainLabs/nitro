package validator

import (
	"context"
	"time"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

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
				selfStakedAssertion, err := v.rollup.AssertionHasStaker(&bind.CallOpts{Context: ctx}, i, v.address)
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
