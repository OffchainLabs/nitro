package timeboost

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type AuctioneerAPI struct {
	*Auctioneer
}

type JsonBid struct {
	ChainId                uint64         `json:"chainId"`
	ExpressLaneController  common.Address `json:"expressLaneController"`
	Bidder                 common.Address `json:"bidder"`
	AuctionContractAddress common.Address `json:"auctionContractAddress"`
	Round                  uint64         `json:"round"`
	Amount                 *big.Int       `json:"amount"`
	Signature              string         `json:"signature"`
}

type JsonExpressLaneSubmission struct {
	ChainId                uint64                             `json:"chainId"`
	Round                  uint64                             `json:"round"`
	AuctionContractAddress common.Address                     `json:"auctionContractAddress"`
	Transaction            *types.Transaction                 `json:"transaction"`
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Signature              string                             `json:"signature"`
}

type ExpressLaneSubmission struct {
	ChainId                uint64
	Round                  uint64
	AuctionContractAddress common.Address
	Transaction            *types.Transaction
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Signature              []byte
}

func JsonSubmissionToGo(submission *JsonExpressLaneSubmission) *ExpressLaneSubmission {
	return &ExpressLaneSubmission{
		ChainId:                submission.ChainId,
		Round:                  submission.Round,
		AuctionContractAddress: submission.AuctionContractAddress,
		Transaction:            submission.Transaction,
		Options:                submission.Options,
		Signature:              common.Hex2Bytes(submission.Signature),
	}
}

func (els *ExpressLaneSubmission) ToJson() *JsonExpressLaneSubmission {
	return &JsonExpressLaneSubmission{
		ChainId:                els.ChainId,
		Round:                  els.Round,
		AuctionContractAddress: els.AuctionContractAddress,
		Transaction:            els.Transaction,
		Options:                els.Options,
		Signature:              common.Bytes2Hex(els.Signature),
	}
}

func (els *ExpressLaneSubmission) ToMessageBytes() ([]byte, error) {
	return encodeExpressLaneSubmission(
		domainValue,
		els.ChainId,
		els.AuctionContractAddress,
		els.Round,
		els.Transaction,
	)
}

func encodeExpressLaneSubmission(
	domainValue []byte, chainId uint64,
	auctionContractAddress common.Address,
	round uint64,
	tx *types.Transaction,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(domainValue)
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, chainId)
	buf.Write(roundBuf)
	buf.Write(auctionContractAddress[:])
	roundBuf = make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, round)
	buf.Write(roundBuf)
	rlpTx, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(rlpTx)
	return buf.Bytes(), nil
}

func (a *AuctioneerAPI) SubmitBid(ctx context.Context, bid *JsonBid) error {
	return a.receiveBid(ctx, &Bid{
		ChainId:                bid.ChainId,
		ExpressLaneController:  bid.ExpressLaneController,
		Bidder:                 bid.Bidder,
		AuctionContractAddress: bid.AuctionContractAddress,
		Round:                  bid.Round,
		Amount:                 bid.Amount,
		Signature:              common.Hex2Bytes(bid.Signature),
	})
}
