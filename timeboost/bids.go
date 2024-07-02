package timeboost

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
)

var (
	ErrMalformedData       = errors.New("malformed bid data")
	ErrNotDepositor        = errors.New("not a depositor")
	ErrWrongChainId        = errors.New("wrong chain id")
	ErrWrongSignature      = errors.New("wrong signature")
	ErrBadRoundNumber      = errors.New("bad round number")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type Bid struct {
	chainId   uint64
	address   common.Address
	round     uint64
	amount    *big.Int
	signature []byte
}

type validatedBid struct {
	Bid
}

func (am *AuctionMaster) newValidatedBid(bid *Bid) (*validatedBid, error) {
	// Check basic integrity.
	if bid == nil {
		return nil, errors.Wrap(ErrMalformedData, "nil bid")
	}
	if bid.address == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty bidder address")
	}
	// Verify chain id.
	if new(big.Int).SetUint64(bid.chainId).Cmp(am.chainId) != 0 {
		return nil, errors.Wrapf(ErrWrongChainId, "wanted %#x, got %#x", am.chainId, bid.chainId)
	}
	// Check if for upcoming round.
	upcomingRound := CurrentRound(am.initialRoundTimestamp, am.roundDuration) + 1
	if bid.round != upcomingRound {
		return nil, errors.Wrapf(ErrBadRoundNumber, "wanted %d, got %d", upcomingRound, bid.round)
	}
	// Check bid amount.
	if bid.amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, errors.Wrap(ErrMalformedData, "expected a non-negative, non-zero bid amount")
	}
	// Validate the signature.
	packedBidBytes, err := encodeBidValues(
		am.signatureDomain, new(big.Int).SetUint64(bid.chainId), new(big.Int).SetUint64(bid.round), bid.amount,
	)
	if err != nil {
		return nil, ErrMalformedData
	}
	// Ethereum signatures contain the recovery id at the last byte
	if len(bid.signature) != 65 {
		return nil, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	hash := crypto.Keccak256(packedBidBytes)
	prefixed := crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), hash)
	pubkey, err := crypto.SigToPub(prefixed, bid.signature)
	if err != nil {
		return nil, err
	}
	if !verifySignature(pubkey, packedBidBytes, bid.signature) {
		return nil, ErrWrongSignature
	}
	// Validate if the user if a depositor in the contract and has enough balance for the bid.
	// TODO: Retry some number of times if flakey connection.
	// TODO: Validate reserve price against amount of bid.
	depositBal, err := am.auctionContract.DepositBalance(&bind.CallOpts{}, bid.address)
	if err != nil {
		return nil, err
	}
	if depositBal.Cmp(new(big.Int)) == 0 {
		return nil, ErrNotDepositor
	}
	if depositBal.Cmp(bid.amount) < 0 {
		return nil, errors.Wrapf(ErrInsufficientBalance, "onchain balance %#x, bid amount %#x", depositBal, bid.amount)
	}
	return &validatedBid{*bid}, nil
}

type bidCache struct {
	sync.RWMutex
	latestBidBySender map[common.Address]*validatedBid
}

func newBidCache() *bidCache {
	return &bidCache{
		latestBidBySender: make(map[common.Address]*validatedBid),
	}
}

func (bc *bidCache) add(bid *validatedBid) {
	bc.Lock()
	defer bc.Unlock()
	bc.latestBidBySender[bid.address] = bid
}

// TwoTopBids returns the top two bids for the given chain ID and round
type auctionResult struct {
	firstPlace  *validatedBid
	secondPlace *validatedBid
}

func (bc *bidCache) topTwoBids() *auctionResult {
	bc.RLock()
	defer bc.RUnlock()
	result := &auctionResult{}
	for _, bid := range bc.latestBidBySender {
		if result.firstPlace == nil || bid.amount.Cmp(result.firstPlace.amount) > 0 {
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		} else if result.secondPlace == nil || bid.amount.Cmp(result.secondPlace.amount) > 0 {
			result.secondPlace = bid
		}
	}
	return result
}

func verifySignature(pubkey *ecdsa.PublicKey, message []byte, sig []byte) bool {
	hash := crypto.Keccak256(message)
	prefixed := crypto.Keccak256([]byte("\x19Ethereum Signed Message:\n32"), hash)

	return secp256k1.VerifySignature(crypto.FromECDSAPub(pubkey), prefixed, sig[:len(sig)-1])
}

// Helper function to pad a big integer to 32 bytes
func padBigInt(bi *big.Int) []byte {
	bb := bi.Bytes()
	padded := make([]byte, 32-len(bb), 32)
	padded = append(padded, bb...)
	return padded
}

func encodeBidValues(domainPrefix uint16, chainId, round, amount *big.Int) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode uint16 - occupies 2 bytes
	err := binary.Write(buf, binary.BigEndian, domainPrefix)
	if err != nil {
		return nil, err
	}

	// Encode uint256 values - each occupies 32 bytes
	buf.Write(padBigInt(chainId))
	buf.Write(padBigInt(round))
	buf.Write(padBigInt(amount))

	return buf.Bytes(), nil
}
