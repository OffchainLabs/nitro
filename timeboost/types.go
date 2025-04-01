package timeboost

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
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

func (b *Bid) ToEIP712Hash(domainSeparator [32]byte) (common.Hash, error) {
	types := apitypes.Types{
		"Bid": []apitypes.Type{
			{Name: "round", Type: "uint64"},
			{Name: "expressLaneController", Type: "address"},
			{Name: "amount", Type: "uint256"},
		},
	}

	message := apitypes.TypedDataMessage{
		"round":                 big.NewInt(0).SetUint64(b.Round),
		"expressLaneController": [20]byte(b.ExpressLaneController),
		"amount":                b.Amount,
	}

	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "Bid",
		Message:     message,
		Domain:      apitypes.TypedDataDomain{Salt: "Unused; domain separator fetched from method on contract. This must be nonempty for validation."},
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return common.Hash{}, err
	}

	bidHash := crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSeparator[:],
		messageHash,
	)

	return bidHash, nil
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
	ChainId                *big.Int
	AuctionContractAddress common.Address
	Signature              []byte

	// For tie breaking
	Bidder                common.Address
	ExpressLaneController common.Address
	Round                 uint64
	Amount                *big.Int
}

// BigIntHash returns the hash of the bidder and bidBytes in the form of a big.Int.
// The hash is equivalent to the following Solidity implementation:
//
//	uint256(keccak256(abi.encodePacked(bidder, bidBytes)))
//
// This is only used for breaking ties amongst equivalent bids and not used for
// Bid signing, which uses EIP 712 as the hashing scheme.
func (v *ValidatedBid) BigIntHash(domainSeparator [32]byte) *big.Int {
	bid := &Bid{
		ExpressLaneController: v.ExpressLaneController,
		Round:                 v.Round,
		Amount:                v.Amount,
	}
	// Since ToEIP712Hash is deterministic, this error can be ignored here, as the bidvalidator
	// would have previously validated it when calculating bidHash
	bidHash, _ := bid.ToEIP712Hash(domainSeparator)
	bidder := v.Bidder.Bytes()

	return new(big.Int).SetBytes(crypto.Keccak256Hash(bidder, bidHash.Bytes()).Bytes())
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
	SequenceNumber         hexutil.Uint64                     `json:"sequenceNumber"`
	Signature              hexutil.Bytes                      `json:"signature"`
}

type ExpressLaneSubmission struct {
	ChainId                *big.Int
	Round                  uint64
	AuctionContractAddress common.Address
	Transaction            *types.Transaction
	Options                *arbitrum_types.ConditionalOptions `json:"options"`
	SequenceNumber         uint64
	Signature              []byte

	sender common.Address
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
		SequenceNumber:         uint64(submission.SequenceNumber),
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
		SequenceNumber:         hexutil.Uint64(els.SequenceNumber),
		Signature:              els.Signature,
	}, nil
}

func (els *ExpressLaneSubmission) ToMessageBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.Write(domainValue)
	buf.Write(padBigInt(els.ChainId))
	buf.Write(els.AuctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, els.Round)
	buf.Write(roundBuf)
	seqBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(seqBuf, els.SequenceNumber)
	buf.Write(seqBuf)
	rlpTx, err := els.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(rlpTx)
	return buf.Bytes(), nil
}

func (els *ExpressLaneSubmission) Sender() (common.Address, error) {
	if (els.sender != common.Address{}) {
		return els.sender, nil
	}
	// Reconstruct the message being signed over and recover the sender address.
	signingMessage, err := els.ToMessageBytes()
	if err != nil {
		return common.Address{}, ErrMalformedData
	}
	if len(els.Signature) != 65 {
		return common.Address{}, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	prefixed := crypto.Keccak256(append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(signingMessage))), signingMessage...))
	sigItem := make([]byte, len(els.Signature))
	copy(sigItem, els.Signature)
	// Signature verification expects the last byte of the signature to have 27 subtracted,
	// as it represents the recovery ID. If the last byte is greater than or equal to 27, it indicates a recovery ID that hasn't been adjusted yet,
	// it's needed for internal signature verification logic.
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(prefixed, sigItem)
	if err != nil {
		return common.Address{}, ErrMalformedData
	}
	els.sender = crypto.PubkeyToAddress(*pubkey)
	return els.sender, nil
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
