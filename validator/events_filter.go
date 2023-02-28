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
	createdAssertionEventSig = hexutil.MustDecode("0xb795d7f067118d6e112dcd15e103f1a9de80c67210733e0d01e065a35bfb3242")
	challengeStartedEventSig = hexutil.MustDecode("0xc795d7f067118d6e112dcd15e103f1a9de80c67210733e0d01e065a35bfb3242")
)

type assertionCreatedEvent struct {
	assertionNum        protocol.AssertionSequenceNumber
	assertionHash       protocol.AssertionHash
	parentAssertionHash protocol.AssertionHash
	height              uint64
}

type challengeStartedEvent struct {
	challenger             common.Address
	challengedAssertionNum protocol.AssertionSequenceNumber
	challengeNum           uint64
}

// Subscribes to events fired by the rollup contracts in order to listen to
// new assertion creations or challenge start events from the protocol.
func (v *Validator) handleRollupEvents(ctx context.Context) {
	logs := make(chan types.Log)
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
					return
				}
				v.onLeafCreated(ctx, &assertionCreatedEvent{
					assertionNum:        protocol.AssertionSequenceNumber(createdAssertion.AssertionNum),
					assertionHash:       protocol.AssertionHash(createdAssertion.AssertionHash),
					parentAssertionHash: protocol.AssertionHash(createdAssertion.AssertionHash),
					numBlocks:           createdAssertion.Assertion.NumBlocks,
				})
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
