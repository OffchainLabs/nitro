package timeboost

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ExpressLaneClient struct {
	stopwaiter.StopWaiter
	privKey               *ecdsa.PrivateKey
	chainId               uint64
	initialRoundTimestamp time.Time
	roundDuration         time.Duration
	auctionContractAddr   common.Address
	client                *rpc.Client
}

func NewExpressLaneClient(
	privKey *ecdsa.PrivateKey,
	chainId uint64,
	initialRoundTimestamp time.Time,
	roundDuration time.Duration,
	auctionContractAddr common.Address,
	client *rpc.Client,
) *ExpressLaneClient {
	return &ExpressLaneClient{
		privKey:               privKey,
		chainId:               chainId,
		initialRoundTimestamp: initialRoundTimestamp,
		roundDuration:         roundDuration,
		auctionContractAddr:   auctionContractAddr,
		client:                client,
	}
}

func (elc *ExpressLaneClient) SendTransaction(ctx context.Context, transaction *types.Transaction) containers.PromiseInterface[struct{}] {
	return stopwaiter.LaunchPromiseThread(elc, func(ctx context.Context) (struct{}, error) {
		msg := &ExpressLaneSubmission{
			ChainId:                elc.chainId,
			Round:                  CurrentRound(elc.initialRoundTimestamp, elc.roundDuration),
			AuctionContractAddress: elc.auctionContractAddr,
			Transaction:            transaction,
		}
		signingMsg, err := msg.ToMessageBytes()
		if err != nil {
			return struct{}{}, err
		}
		signature, err := signSubmission(signingMsg, elc.privKey)
		if err != nil {
			return struct{}{}, err
		}
		msg.Signature = signature
		err = elc.client.CallContext(ctx, nil, "timeboost_sendExpressLaneTransaction", msg)
		return struct{}{}, err
	})
}

func signSubmission(message []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))), message...))
	sig, err := secp256k1.Sign(prefixed, math.PaddedBigBytes(key.D, 32))
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}