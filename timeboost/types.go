package timeboost

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
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

func (v *ValidatedBid) Hash() string {
	// Concatenate the bidder address and the byte representation of the bid
	data := append(v.Bidder.Bytes(), padBigInt(v.ChainId)...)
	data = append(data, v.AuctionContractAddress.Bytes()...)
	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, v.Round)
	data = append(data, roundBytes...)
	data = append(data, v.Amount.Bytes()...)
	data = append(data, v.ExpressLaneController.Bytes()...)

	hash := sha256.Sum256(data)
	// Return the hash as a hexadecimal string
	return fmt.Sprintf("%x", hash)
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
	return encodeExpressLaneSubmission(
		domainValue,
		els.ChainId,
		els.Sequence,
		els.AuctionContractAddress,
		els.Round,
		els.Transaction,
	)
}

func encodeExpressLaneSubmission(
	domainValue []byte,
	chainId *big.Int,
	sequence uint64,
	auctionContractAddress common.Address,
	round uint64,
	tx *types.Transaction,
) ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(domainValue)
	buf.Write(padBigInt(chainId))
	seqBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBuf, sequence)
	buf.Write(seqBuf)
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

func verifySignature(pubkey *ecdsa.PublicKey, message []byte, sig []byte) bool {
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(message))), message...))
	return secp256k1.VerifySignature(crypto.FromECDSAPub(pubkey), prefixed, sig[:len(sig)-1])
}

// Helper function to pad a big integer to 32 bytes
func padBigInt(bi *big.Int) []byte {
	bb := bi.Bytes()
	padded := make([]byte, 32-len(bb), 32)
	padded = append(padded, bb...)
	return padded
}

func encodeBidValues(domainValue []byte, chainId *big.Int, auctionContractAddress common.Address, round uint64, amount *big.Int, expressLaneController common.Address) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode uint256 values - each occupies 32 bytes
	buf.Write(domainValue)
	buf.Write(padBigInt(chainId))
	buf.Write(auctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, round)
	buf.Write(roundBuf)
	buf.Write(padBigInt(amount))
	buf.Write(expressLaneController[:])

	return buf.Bytes(), nil
}
