package timeboost

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/pkg/errors"
)

var (
	ErrMalformedData            = errors.New("MALFORMED_DATA")
	ErrNotDepositor             = errors.New("NOT_DEPOSITOR")
	ErrWrongChainId             = errors.New("WRONG_CHAIN_ID")
	ErrWrongSignature           = errors.New("WRONG_SIGNATURE")
	ErrBadRoundNumber           = errors.New("BAD_ROUND_NUMBER")
	ErrInsufficientBalance      = errors.New("INSUFFICIENT_BALANCE")
	ErrReservePriceNotMet       = errors.New("RESERVE_PRICE_NOT_MET")
	ErrNoOnchainController      = errors.New("NO_ONCHAIN_CONTROLLER")
	ErrWrongAuctionContract     = errors.New("WRONG_AUCTION_CONTRACT")
	ErrNotExpressLaneController = errors.New("NOT_EXPRESS_LANE_CONTROLLER")
)

type Bid struct {
	ChainId                uint64
	ExpressLaneController  common.Address
	Bidder                 common.Address
	AuctionContractAddress common.Address
	Round                  uint64
	Amount                 *big.Int
	Signature              []byte
}

type validatedBid struct {
	expressLaneController common.Address
	amount                *big.Int
	signature             []byte
	// For tie breaking
	chainId                uint64
	auctionContractAddress common.Address
	round                  uint64
	bidder                 common.Address
}
type bidCache struct {
	sync.RWMutex
	bidsByExpressLaneControllerAddr map[common.Address]*validatedBid
}

func newBidCache() *bidCache {
	return &bidCache{
		bidsByExpressLaneControllerAddr: make(map[common.Address]*validatedBid),
	}
}

func (bc *bidCache) add(bid *validatedBid) {
	bc.Lock()
	defer bc.Unlock()
	bc.bidsByExpressLaneControllerAddr[bid.expressLaneController] = bid
}

// TwoTopBids returns the top two bids for the given chain ID and round
type auctionResult struct {
	firstPlace  *validatedBid
	secondPlace *validatedBid
}

func (bc *bidCache) size() int {
	bc.RLock()
	defer bc.RUnlock()
	return len(bc.bidsByExpressLaneControllerAddr)

}

// topTwoBids returns the top two bids in the cache.
func (bc *bidCache) topTwoBids() *auctionResult {
	bc.RLock()
	defer bc.RUnlock()

	result := &auctionResult{}

	for _, bid := range bc.bidsByExpressLaneControllerAddr {
		if result.firstPlace == nil {
			result.firstPlace = bid
		} else if bid.amount.Cmp(result.firstPlace.amount) > 0 {
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		} else if bid.amount.Cmp(result.firstPlace.amount) == 0 {
			if hashBid(bid) > hashBid(result.firstPlace) {
				result.secondPlace = result.firstPlace
				result.firstPlace = bid
			} else if result.secondPlace == nil || hashBid(bid) > hashBid(result.secondPlace) {
				result.secondPlace = bid
			}
		} else if result.secondPlace == nil || bid.amount.Cmp(result.secondPlace.amount) > 0 {
			result.secondPlace = bid
		} else if bid.amount.Cmp(result.secondPlace.amount) == 0 {
			if hashBid(bid) > hashBid(result.secondPlace) {
				result.secondPlace = bid
			}
		}
	}

	return result
}

// hashBid hashes the bidder address concatenated with the respective byte-string representation of the bid using the Keccak256 hashing scheme.
func hashBid(bid *validatedBid) string {
	chainIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(chainIdBytes, bid.chainId)
	roundBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBytes, bid.round)

	// Concatenate the bidder address and the byte representation of the bid
	data := append(bid.bidder.Bytes(), chainIdBytes...)
	data = append(data, bid.auctionContractAddress.Bytes()...)
	data = append(data, roundBytes...)
	data = append(data, bid.amount.Bytes()...)
	data = append(data, bid.expressLaneController.Bytes()...)

	hash := sha256.Sum256(data)

	// Return the hash as a hexadecimal string
	return fmt.Sprintf("%x", hash)
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

func encodeBidValues(domainValue []byte, chainId uint64, auctionContractAddress common.Address, round uint64, amount *big.Int, expressLaneController common.Address) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode uint256 values - each occupies 32 bytes
	buf.Write(domainValue)
	chainIdBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(chainIdBuf, chainId)
	buf.Write(chainIdBuf)
	buf.Write(auctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, round)
	buf.Write(roundBuf)
	buf.Write(padBigInt(amount))
	buf.Write(expressLaneController[:])

	return buf.Bytes(), nil
}
