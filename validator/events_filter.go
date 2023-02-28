package validator

import (
	"bytes"
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	challengeStartedEventSig = hexutil.MustDecode("0x1811239f50280ab7ba21c37f9f04fc72d9796c8fe213e714d281b87606509dae")
	createdAssertionEventSig = hexutil.MustDecode("0x0811239f50280ab7ba21c37f9f04fc72d9796c8fe213e714d281b87606509dae")
)

type challengeStartedEvent struct {
	challenger             common.Address
	challengedAssertionNum protocol.AssertionSequenceNumber
	challengeNum           uint64
}

// Subscribes to events fired by the rollup contracts in order to listen to
// new assertion creations or challenge start events from the protocol.
func (v *Validator) handleRollupEvents(ctx context.Context) {
	logs := make(chan types.Log, 100)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{v.rollupAddr},
	}
	sub, err := v.backend.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		log.Error(err)
		return
	}
	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case vLog := <-logs:
			if len(vLog.Topics) == 0 {
				continue
			}
			topic := vLog.Topics[0]
			switch {
			case bytes.Equal(topic[:], createdAssertionEventSig):
				createdAssertion, err := v.rollup.ParseAssertionCreated(vLog)
				if err != nil {
					log.Error(err)
					continue
				}
				var assertion protocol.Assertion
				if err := v.chain.Call(func(tx protocol.ActiveTx) error {
					retrieved, err := v.chain.AssertionBySequenceNum(ctx, tx, protocol.AssertionSequenceNumber(createdAssertion.AssertionNum))
					if err != nil {
						return err
					}
					assertion = retrieved
					return nil
				}); err != nil {
					log.Error(err)
					continue
				}
				v.onLeafCreated(ctx, assertion)
			case bytes.Equal(topic[:], challengeStartedEventSig):
				chalStarted, err := v.rollup.ParseRollupChallengeStarted(vLog)
				if err != nil {
					log.Error(err)
					return
				}
				v.onChallengeStarted(ctx, &challengeStartedEvent{
					challenger:             chalStarted.Challenger,
					challengedAssertionNum: protocol.AssertionSequenceNumber(chalStarted.ChallengedAssertion),
					challengeNum:           chalStarted.ChallengeIndex,
				})
			default:
				log.Infof("Got event that did not match: %#x", topic)
			}
		}
	}
}
