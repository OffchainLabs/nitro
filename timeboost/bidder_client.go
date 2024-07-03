package timeboost

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/timeboost/bindings"
	"github.com/pkg/errors"
)

type sequencerConnection interface {
	SendExpressLaneTx(ctx context.Context, tx *types.Transaction) error
}

type auctioneerConnection interface {
	SubmitBid(ctx context.Context, bid *Bid) error
}

type BidderClient struct {
	chainId               uint64
	name                  string
	signatureDomain       uint16
	txOpts                *bind.TransactOpts
	client                arbutil.L1Interface
	privKey               *ecdsa.PrivateKey
	auctionContract       *bindings.ExpressLaneAuction
	sequencer             sequencerConnection
	auctioneer            auctioneerConnection
	initialRoundTimestamp time.Time
	roundDuration         time.Duration
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
	client arbutil.L1Interface,
	auctionContractAddress common.Address,
	sequencer sequencerConnection,
	auctioneer auctioneerConnection,
) (*BidderClient, error) {
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	auctionContract, err := bindings.NewExpressLaneAuction(auctionContractAddress, client)
	if err != nil {
		return nil, err
	}
	sigDomain, err := auctionContract.BidSignatureDomainValue(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	initialRoundTimestamp, err := auctionContract.InitialRoundTimestamp(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	roundDurationSeconds, err := auctionContract.RoundDurationSeconds(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	return &BidderClient{
		chainId:               chainId.Uint64(),
		name:                  name,
		signatureDomain:       sigDomain,
		client:                client,
		txOpts:                wallet.TxOpts,
		privKey:               wallet.PrivKey,
		auctionContract:       auctionContract,
		sequencer:             sequencer,
		auctioneer:            auctioneer,
		initialRoundTimestamp: time.Unix(initialRoundTimestamp.Int64(), 0),
		roundDuration:         time.Duration(roundDurationSeconds) * time.Second,
	}, nil
}

func (bd *BidderClient) Start(ctx context.Context) {
	// Monitor for newly assigned express lane controllers, and if the client's address
	// is the controller in order to send express lane txs.
	go bd.monitorAuctionResolutions(ctx)
	// Monitor for auction closures by the autonomous auctioneer.
	go bd.monitorAuctionCancelations(ctx)
	// Monitor for express lane control delegations to take over if needed.
	go bd.monitorExpressLaneDelegations(ctx)
}

func (bd *BidderClient) monitorAuctionResolutions(ctx context.Context) {
	winningBidders := []common.Address{bd.txOpts.From}
	latestBlock, err := bd.client.HeaderByNumber(ctx, nil)
	if err != nil {
		panic(err)
	}
	fromBlock := latestBlock.Number.Uint64()
	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			latestBlock, err := bd.client.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Error("Could not get latest header", "err", err)
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Context: ctx,
				Start:   fromBlock,
				End:     &toBlock,
			}
			it, err := bd.auctionContract.FilterAuctionResolved(filterOpts, winningBidders, nil)
			if err != nil {
				log.Error("Could not filter auction resolutions", "error", err)
				continue
			}
			for it.Next() {
				upcomingRound := CurrentRound(bd.initialRoundTimestamp, bd.roundDuration) + 1
				ev := it.Event
				if ev.WinnerRound.Uint64() == upcomingRound {
					// TODO: Log the time to next round.
					log.Info(
						"WON the express lane auction for next round - can send fast lane txs to sequencer",
						"winner", ev.WinningBidder,
						"upcomingRound", upcomingRound,
						"firstPlaceBidAmount", fmt.Sprintf("%#x", ev.WinningBidAmount),
						"secondPlaceBidAmount", fmt.Sprintf("%#x", ev.WinningBidAmount),
					)
				}
			}
			fromBlock = toBlock
		}
	}
}

func (bd *BidderClient) monitorAuctionCancelations(ctx context.Context) {
	// TODO: Implement.
}

func (bd *BidderClient) monitorExpressLaneDelegations(ctx context.Context) {
	delegatedTo := []common.Address{bd.txOpts.From}
	latestBlock, err := bd.client.HeaderByNumber(ctx, nil)
	if err != nil {
		panic(err)
	}
	fromBlock := latestBlock.Number.Uint64()
	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			latestBlock, err := bd.client.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Error("Could not get latest header", "err", err)
				continue
			}
			toBlock := latestBlock.Number.Uint64()
			if fromBlock == toBlock {
				continue
			}
			filterOpts := &bind.FilterOpts{
				Context: ctx,
				Start:   fromBlock,
				End:     &toBlock,
			}
			it, err := bd.auctionContract.FilterExpressLaneControlDelegated(filterOpts, nil, delegatedTo)
			if err != nil {
				log.Error("Could not filter auction resolutions", "error", err)
				continue
			}
			for it.Next() {
				upcomingRound := CurrentRound(bd.initialRoundTimestamp, bd.roundDuration) + 1
				ev := it.Event
				// TODO: Log the time to next round.
				log.Info(
					"Received express lane delegation for next round -  can send fast lane txs to sequencer",
					"delegatedFrom", ev.From,
					"upcomingRound", upcomingRound,
				)
			}
			fromBlock = toBlock
		}
	}
}

func (bd *BidderClient) sendExpressLaneTx(ctx context.Context, tx *types.Transaction) error {
	return bd.sequencer.SendExpressLaneTx(ctx, tx)
}

func (bd *BidderClient) Deposit(ctx context.Context, amount *big.Int) error {
	tx, err := bd.auctionContract.SubmitDeposit(bd.txOpts, amount)
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

func (bd *BidderClient) Bid(ctx context.Context, amount *big.Int) (*Bid, error) {
	newBid := &Bid{
		chainId: bd.chainId,
		address: bd.txOpts.From,
		round:   CurrentRound(bd.initialRoundTimestamp, bd.roundDuration) + 1,
		amount:  amount,
	}
	packedBidBytes, err := encodeBidValues(
		bd.signatureDomain, new(big.Int).SetUint64(newBid.chainId), new(big.Int).SetUint64(newBid.round), amount,
	)
	if err != nil {
		return nil, err
	}
	sig, prefixed := sign(packedBidBytes, bd.privKey)
	newBid.signature = sig
	_ = prefixed
	if err = bd.auctioneer.SubmitBid(ctx, newBid); err != nil {
		return nil, err
	}
	return newBid, nil
}

func sign(message []byte, key *ecdsa.PrivateKey) ([]byte, []byte) {
	hash := crypto.Keccak256(message)
	prefixed := crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), hash)
	sig, err := secp256k1.Sign(prefixed, math.PaddedBigBytes(key.D, 32))
	if err != nil {
		panic(err)
	}
	return sig, prefixed
}
