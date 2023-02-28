package validator

import (
	"bytes"
	"context"

	"github.com/OffchainLabs/challenge-protocol-v2/protocol"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	challengeStartedEventSig = hexutil.MustDecode("0x1811239f50280ab7ba21c37f9f04fc72d9796c8fe213e714d281b87606509dae") // TODO: Not a real sig.
	createdAssertionEventSig = hexutil.MustDecode("0x0811239f50280ab7ba21c37f9f04fc72d9796c8fe213e714d281b87606509dae")
	vertexAddedEventSig      = hexutil.MustDecode("0x4383ba11a7cd16be5880c5f674b93be38b3b1fcafd7a7b06151998fa2a675349")
	mergeEventSig            = hexutil.MustDecode("0x72b50597145599e4288d411331c925b40b33b0fa3cccadc1f57d2a1ab973553a")
	bisectEventSig           = hexutil.MustDecode("0x69d5465c81edf7aaaf2e5c6c8829500df87d84c87f8d5b1221b59eaeaca70d27")
)

type challengeStartedEvent struct {
	challenger             common.Address
	challengedAssertionNum protocol.AssertionSequenceNumber
	challengeNum           uint64
}

// Subscribes to events fired by the rollup contracts in order to listen to
// new assertion creations or challenge start events from the protocol.
// TODO: Brittle - should be based on querying the chain instead.
func (v *Validator) handleRollupEvents(ctx context.Context) {
	assertionChainLogs := make(chan types.Log, 100)
	assertionChainQuery := ethereum.FilterQuery{
		Addresses: []common.Address{v.rollupAddr},
	}
	assertionSub, err := v.backend.SubscribeFilterLogs(ctx, assertionChainQuery, assertionChainLogs)
	if err != nil {
		log.Error(err)
		return
	}
	defer assertionSub.Unsubscribe()

	chalLogs := make(chan types.Log, 100)
	chalQuery := ethereum.FilterQuery{
		Addresses: []common.Address{v.chalManagerAddr},
	}
	chalSub, err := v.backend.SubscribeFilterLogs(ctx, chalQuery, chalLogs)
	if err != nil {
		log.Error(err)
		return
	}
	defer chalSub.Unsubscribe()

	for {
		select {
		case err := <-assertionSub.Err():
			log.Fatal(err)
		case err := <-chalSub.Err():
			log.Fatal(err)
		case vLog := <-chalLogs:
			if len(vLog.Topics) == 0 {
				continue
			}
			topic := vLog.Topics[0]
			switch {
			case bytes.Equal(topic[:], challengeStartedEventSig):
				chalStarted, err := v.chalManager.ParseChallengeStarted(vLog)
				if err != nil {
					log.Error(err)
					return
				}
				if err := v.onChallengeStarted(ctx, &challengeStartedEvent{
					challenger:             chalStarted.Challenger,
					challengedAssertionNum: protocol.AssertionSequenceNumber(chalStarted.ChallengedAssertion),
					challengeNum:           chalStarted.ChallengeIndex,
				}); err != nil {
					log.Error(err)
				}
			default:
			}
		case vLog := <-assertionChainLogs:
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
			// TODO: This is not working.
			case bytes.Equal(topic[:], challengeStartedEventSig):
				chalStarted, err := v.rollup.ParseRollupChallengeStarted(vLog)
				if err != nil {
					log.Error(err)
					return
				}
				if err := v.onChallengeStarted(ctx, &challengeStartedEvent{
					challenger:             chalStarted.Challenger,
					challengedAssertionNum: protocol.AssertionSequenceNumber(chalStarted.ChallengedAssertion),
					challengeNum:           chalStarted.ChallengeIndex,
				}); err != nil {
					log.Error(err)
				}
			default:
			}
		}
	}
}
