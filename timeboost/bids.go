package timeboost

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"sync"

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
	ErrInsufficientBid     = errors.New("insufficient bid")
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
		// If first place is empty or bid is higher than the current first place
		if result.firstPlace == nil || bid.amount.Cmp(result.firstPlace.amount) > 0 {
			result.secondPlace = result.firstPlace
			result.firstPlace = bid
		} else if result.secondPlace == nil || bid.amount.Cmp(result.secondPlace.amount) > 0 {
			// If second place is empty or bid is higher than current second place
			result.secondPlace = bid
		}
	}

	return result
}

func verifySignature(pubkey *ecdsa.PublicKey, message []byte, sig []byte) bool {
	prefixed := crypto.Keccak256(append([]byte("\x19Ethereum Signed Message:\n112"), message...))

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
