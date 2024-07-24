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
	chainId                uint64
	expressLaneController  common.Address
	bidder                 common.Address
	auctionContractAddress common.Address
	round                  uint64
	amount                 *big.Int
	signature              []byte
}

type validatedBid struct {
	expressLaneController common.Address
	amount                *big.Int
	signature             []byte
}

func (am *Auctioneer) fetchReservePrice() *big.Int {
	am.reservePriceLock.RLock()
	defer am.reservePriceLock.RUnlock()
	return new(big.Int).Set(am.reservePrice)
}

func (am *Auctioneer) newValidatedBid(bid *Bid) (*validatedBid, error) {
	// Check basic integrity.
	if bid == nil {
		return nil, errors.Wrap(ErrMalformedData, "nil bid")
	}
	if bid.bidder == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty bidder address")
	}
	if bid.expressLaneController == (common.Address{}) {
		return nil, errors.Wrap(ErrMalformedData, "empty express lane controller address")
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
	reservePrice := am.fetchReservePrice()
	if bid.amount.Cmp(reservePrice) == -1 {
		return nil, errors.Wrap(ErrMalformedData, "expected bid to be at least of reserve price magnitude")
	}
	// Validate the signature.
	packedBidBytes, err := encodeBidValues(
		new(big.Int).SetUint64(bid.chainId),
		am.auctionContractAddr,
		bid.round,
		bid.amount,
		bid.expressLaneController,
	)
	if err != nil {
		return nil, ErrMalformedData
	}
	if len(bid.signature) != 65 {
		return nil, errors.Wrap(ErrMalformedData, "signature length is not 65")
	}
	// Recover the public key.
	prefixed := crypto.Keccak256(append([]byte("\x19Ethereum Signed Message:\n112"), packedBidBytes...))
	sigItem := make([]byte, len(bid.signature))
	copy(sigItem, bid.signature)
	if sigItem[len(sigItem)-1] >= 27 {
		sigItem[len(sigItem)-1] -= 27
	}
	pubkey, err := crypto.SigToPub(prefixed, sigItem)
	if err != nil {
		return nil, ErrMalformedData
	}
	if !verifySignature(pubkey, packedBidBytes, sigItem) {
		return nil, ErrWrongSignature
	}
	// Validate if the user if a depositor in the contract and has enough balance for the bid.
	// TODO: Retry some number of times if flakey connection.
	// TODO: Validate reserve price against amount of bid.
	// TODO: No need to do anything expensive if the bid coming is in invalid.
	// Cache this if the received time of the bid is too soon. Include the arrival timestamp.
	depositBal, err := am.auctionContract.BalanceOf(&bind.CallOpts{}, bid.bidder)
	if err != nil {
		return nil, err
	}
	if depositBal.Cmp(new(big.Int)) == 0 {
		return nil, ErrNotDepositor
	}
	if depositBal.Cmp(bid.amount) < 0 {
		return nil, errors.Wrapf(ErrInsufficientBalance, "onchain balance %#x, bid amount %#x", depositBal, bid.amount)
	}
	return &validatedBid{
		expressLaneController: bid.expressLaneController,
		amount:                bid.amount,
		signature:             bid.signature,
	}, nil
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

func (bc *bidCache) topTwoBids() *auctionResult {
	bc.RLock()
	defer bc.RUnlock()
	result := &auctionResult{}
	// TODO: Tiebreaker handle.
	for _, bid := range bc.bidsByExpressLaneControllerAddr {
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

func encodeBidValues(chainId *big.Int, auctionContractAddress common.Address, round uint64, amount *big.Int, expressLaneController common.Address) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode uint256 values - each occupies 32 bytes
	buf.Write(padBigInt(chainId))
	buf.Write(auctionContractAddress[:])
	roundBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(roundBuf, round)
	buf.Write(roundBuf)
	buf.Write(padBigInt(amount))
	buf.Write(expressLaneController[:])

	return buf.Bytes(), nil
}
