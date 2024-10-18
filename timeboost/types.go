package timeboost

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Bid struct {
	Id                     uint64         `db:"Id"`
	ChainId                *big.Int       `db:"ChainId"`
	ExpressLaneController  common.Address `db:"ExpressLaneController"`
	AuctionContractAddress common.Address `db:"AuctionContractAddress"`
	Round                  uint64         `db:"Round"`
	Amount                 *big.Int       `db:"Amount"`
	Signature              []byte         `db:"Signature"`
}

func (b *Bid) ToJson() *JsonBid {
	return &JsonBid{
		ChainId:                (*hexutil.Big)(b.ChainId),
		ExpressLaneController:  b.ExpressLaneController,
		AuctionContractAddress: b.AuctionContractAddress,
		Round:                  hexutil.Uint64(b.Round),
		Amount:                 (*hexutil.Big)(b.Amount),
		Signature:              b.Signature,
	}
}

func (b *Bid) ToMessageBytes() []byte {
	buf := new(bytes.Buffer)
	// Encode uint256 values - each occupies 32 bytes
	buf.Write(domainValue)
	buf.Write(padBigInt(b.ChainId))
	buf.Write(b.AuctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, b.Round)
	buf.Write(roundBuf)
	buf.Write(padBigInt(b.Amount))
	buf.Write(b.ExpressLaneController[:])

	return buf.Bytes()
}

type JsonBid struct {
	ChainId                *hexutil.Big   `json:"chainId"`
	ExpressLaneController  common.Address `json:"expressLaneController"`
	AuctionContractAddress common.Address `json:"auctionContractAddress"`
	Round                  hexutil.Uint64 `json:"round"`
	Amount                 *hexutil.Big   `json:"amount"`
	Signature              hexutil.Bytes  `json:"signature"`
}

type ValidatedBid struct {
	ExpressLaneController common.Address
	Amount                *big.Int
	Signature             []byte
	// For tie breaking
	ChainId                *big.Int
	AuctionContractAddress common.Address
	Round                  uint64
	Bidder                 common.Address
}

// BigIntHash returns the hash of the bidder and bidBytes in the form of a big.Int.
// The hash is equivalent to the following Solidity implementation:
//
//	uint256(keccak256(abi.encodePacked(bidder, bidBytes)))
func (v *ValidatedBid) BigIntHash() *big.Int {
	bidBytes := v.BidBytes()
	bidder := v.Bidder.Bytes()

	return new(big.Int).SetBytes(crypto.Keccak256Hash(bidder, bidBytes).Bytes())
}

// BidBytes returns the byte representation equivalent to the Solidity implementation of
//
//	abi.encodePacked(BID_DOMAIN, block.chainid, address(this), _round, _amount, _expressLaneController)
func (v *ValidatedBid) BidBytes() []byte {
	var buffer bytes.Buffer

	buffer.Write(domainValue)
	buffer.Write(v.ChainId.Bytes())
	buffer.Write(v.AuctionContractAddress.Bytes())

	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, v.Round)
	buffer.Write(roundBytes)

	buffer.Write(v.Amount.Bytes())
	buffer.Write(v.ExpressLaneController.Bytes())

	return buffer.Bytes()
}

func (v *ValidatedBid) ToJson() *JsonValidatedBid {
	return &JsonValidatedBid{
		ExpressLaneController:  v.ExpressLaneController,
		Amount:                 (*hexutil.Big)(v.Amount),
		Signature:              v.Signature,
		ChainId:                (*hexutil.Big)(v.ChainId),
		AuctionContractAddress: v.AuctionContractAddress,
		Round:                  hexutil.Uint64(v.Round),
		Bidder:                 v.Bidder,
	}
}

type JsonValidatedBid struct {
	ExpressLaneController  common.Address `json:"expressLaneController"`
	Amount                 *hexutil.Big   `json:"amount"`
	Signature              hexutil.Bytes  `json:"signature"`
	ChainId                *hexutil.Big   `json:"chainId"`
	AuctionContractAddress common.Address `json:"auctionContractAddress"`
	Round                  hexutil.Uint64 `json:"round"`
	Bidder                 common.Address `json:"bidder"`
}

func JsonValidatedBidToGo(bid *JsonValidatedBid) *ValidatedBid {
	return &ValidatedBid{
		ExpressLaneController:  bid.ExpressLaneController,
		Amount:                 bid.Amount.ToInt(),
		Signature:              bid.Signature,
		ChainId:                bid.ChainId.ToInt(),
		AuctionContractAddress: bid.AuctionContractAddress,
		Round:                  uint64(bid.Round),
		Bidder:                 bid.Bidder,
	}
}

type JsonExpressLaneSubmission struct {
	ChainId                *hexutil.Big                       `json:"chainId"`
	Round                  hexutil.Uint64                     `json:"round"`
	AuctionContractAddress common.Address                     `json:"auctionContractAddress"`
	Transaction            hexutil.Bytes                      `json:"transaction"`
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Sequence               hexutil.Uint64
	Signature              hexutil.Bytes `json:"signature"`
}

type ExpressLaneSubmission struct {
	ChainId                *big.Int
	Round                  uint64
	AuctionContractAddress common.Address
	Transaction            *types.Transaction
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	Sequence               uint64
	Signature              []byte
}

func JsonSubmissionToGo(submission *JsonExpressLaneSubmission) (*ExpressLaneSubmission, error) {
	tx := &types.Transaction{}
	if err := tx.UnmarshalBinary(submission.Transaction); err != nil {
		return nil, err
	}
	return &ExpressLaneSubmission{
		ChainId:                submission.ChainId.ToInt(),
		Round:                  uint64(submission.Round),
		AuctionContractAddress: submission.AuctionContractAddress,
		Transaction:            tx,
		Options:                submission.Options,
		Sequence:               uint64(submission.Sequence),
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
		Sequence:               hexutil.Uint64(els.Sequence),
		Signature:              els.Signature,
	}, nil
}

func (els *ExpressLaneSubmission) ToMessageBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(domainValue)
	buf.Write(padBigInt(els.ChainId))
	seqBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBuf, els.Sequence)
	buf.Write(seqBuf)
	buf.Write(els.AuctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, els.Round)
	buf.Write(roundBuf)
	rlpTx, err := els.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(rlpTx)
	return buf.Bytes(), nil
}

// Helper function to pad a big integer to 32 bytes
func padBigInt(bi *big.Int) []byte {
	bb := bi.Bytes()
	padded := make([]byte, 32-len(bb), 32)
	padded = append(padded, bb...)
	return padded
}

type SqliteDatabaseBid struct {
	Id                     uint64 `db:"Id"`
	ChainId                string `db:"ChainId"`
	Bidder                 string `db:"Bidder"`
	ExpressLaneController  string `db:"ExpressLaneController"`
	AuctionContractAddress string `db:"AuctionContractAddress"`
	Round                  uint64 `db:"Round"`
	Amount                 string `db:"Amount"`
	Signature              string `db:"Signature"`
}
