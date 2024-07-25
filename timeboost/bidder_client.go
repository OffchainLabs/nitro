package timeboost

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/offchainlabs/nitro/solgen/go/express_lane_auctiongen"
	"github.com/pkg/errors"
)

type Client interface {
	bind.ContractBackend
	bind.DeployBackend
	ChainID(ctx context.Context) (*big.Int, error)
}

type auctioneerConnection interface {
	receiveBid(ctx context.Context, bid *Bid) error
}

type BidderClient struct {
	chainId                uint64
	name                   string
	auctionContractAddress common.Address
	txOpts                 *bind.TransactOpts
	client                 Client
	privKey                *ecdsa.PrivateKey
	auctionContract        *express_lane_auctiongen.ExpressLaneAuction
	auctioneer             auctioneerConnection
	initialRoundTimestamp  time.Time
	roundDuration          time.Duration
	domainValue            []byte
}

// TODO: Provide a safer option.
type Wallet struct {
	TxOpts  *bind.TransactOpts
	PrivKey *ecdsa.PrivateKey
}

func NewBidderClient(
	ctx context.Context,
	name string,
	wallet *Wallet,
	client Client,
	auctionContractAddress common.Address,
	auctioneer auctioneerConnection,
) (*BidderClient, error) {
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	auctionContract, err := express_lane_auctiongen.NewExpressLaneAuction(auctionContractAddress, client)
	if err != nil {
		return nil, err
	}
	roundTimingInfo, err := auctionContract.RoundTimingInfo(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	initialTimestamp := time.Unix(int64(roundTimingInfo.OffsetTimestamp), 0)
	roundDuration := time.Duration(roundTimingInfo.RoundDurationSeconds) * time.Second

	return &BidderClient{
		chainId:                chainId.Uint64(),
		name:                   name,
		auctionContractAddress: auctionContractAddress,
		client:                 client,
		txOpts:                 wallet.TxOpts,
		privKey:                wallet.PrivKey,
		auctionContract:        auctionContract,
		auctioneer:             auctioneer,
		initialRoundTimestamp:  initialTimestamp,
		roundDuration:          roundDuration,
		domainValue:            domainValue,
	}, nil
}

func (bd *BidderClient) Deposit(ctx context.Context, amount *big.Int) error {
	tx, err := bd.auctionContract.Deposit(bd.txOpts, amount)
	if err != nil {
		return err
	}
	receipt, err := bind.WaitMined(ctx, bd.client, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("deposit failed")
	}
	return nil
}

func (bd *BidderClient) Bid(
	ctx context.Context, amount *big.Int, expressLaneController common.Address,
) (*Bid, error) {
	newBid := &Bid{
		ChainId:                bd.chainId,
		ExpressLaneController:  expressLaneController,
		AuctionContractAddress: bd.auctionContractAddress,
		Bidder:                 bd.txOpts.From,
		Round:                  CurrentRound(bd.initialRoundTimestamp, bd.roundDuration) + 1,
		Amount:                 amount,
		Signature:              nil,
	}
	packedBidBytes, err := encodeBidValues(
		bd.domainValue,
		new(big.Int).SetUint64(newBid.ChainId),
		bd.auctionContractAddress,
		newBid.Round,
		amount,
		expressLaneController,
	)
	if err != nil {
		return nil, err
	}
	sig, err := sign(packedBidBytes, bd.privKey)
	if err != nil {
		return nil, err
	}
	newBid.Signature = sig
	if err = bd.auctioneer.receiveBid(ctx, newBid); err != nil {
		return nil, err
	}
	return newBid, nil
}

func sign(message []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	prefixed := crypto.Keccak256(append([]byte("\x19Ethereum Signed Message:\n112"), message...))
	sig, err := secp256k1.Sign(prefixed, math.PaddedBigBytes(key.D, 32))
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}
