package timeboost

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type AuctioneerAPI struct {
	*Auctioneer
}

type JsonBid struct {
	ChainId                *hexutil.Big   `json:"chainId"`
	ExpressLaneController  common.Address `json:"expressLaneController"`
	Bidder                 common.Address `json:"bidder"`
	AuctionContractAddress common.Address `json:"auctionContractAddress"`
	Round                  hexutil.Uint64 `json:"round"`
	Amount                 *hexutil.Big   `json:"amount"`
	Signature              hexutil.Bytes  `json:"signature"`
}

type JsonExpressLaneSubmission struct {
	ChainId                *hexutil.Big                       `json:"chainId"`
	Round                  hexutil.Uint64                     `json:"round"`
	AuctionContractAddress common.Address                     `json:"auctionContractAddress"`
	Transaction            hexutil.Bytes                      `json:"transaction"`
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Signature              hexutil.Bytes                      `json:"signature"`
}

type ExpressLaneSubmission struct {
	ChainId                *big.Int
	Round                  uint64
	AuctionContractAddress common.Address
	Transaction            *types.Transaction
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Signature              []byte
}

func JsonSubmissionToGo(submission *JsonExpressLaneSubmission) (*ExpressLaneSubmission, error) {
	var tx *types.Transaction
	if err := tx.UnmarshalBinary(submission.Transaction); err != nil {
		return nil, err
	}
	return &ExpressLaneSubmission{
		ChainId:                submission.ChainId.ToInt(),
		Round:                  uint64(submission.Round),
		AuctionContractAddress: submission.AuctionContractAddress,
		Transaction:            tx,
		Options:                submission.Options,
		Signature:              submission.Signature,
	}, nil
}

func (els *ExpressLaneSubmission) ToJson() (*JsonExpressLaneSubmission, error) {
	encoded, err := els.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &JsonExpressLaneSubmission{
		ChainId:                (*hexutil.Big)(els.ChainId),
		Round:                  hexutil.Uint64(els.Round),
		AuctionContractAddress: els.AuctionContractAddress,
		Transaction:            encoded,
		Options:                els.Options,
		Signature:              els.Signature,
	}, nil
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
	domainValue []byte,
	chainId *big.Int,
	auctionContractAddress common.Address,
	round uint64,
	tx *types.Transaction,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(domainValue)
	buf.Write(padBigInt(chainId))
	buf.Write(auctionContractAddress[:])
	roundBuf := make([]byte, 8)
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
		ChainId:                bid.ChainId.ToInt(),
		ExpressLaneController:  bid.ExpressLaneController,
		Bidder:                 bid.Bidder,
		AuctionContractAddress: bid.AuctionContractAddress,
		Round:                  uint64(bid.Round),
		Amount:                 bid.Amount.ToInt(),
		Signature:              bid.Signature,
	})
}
