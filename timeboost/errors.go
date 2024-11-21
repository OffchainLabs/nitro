package timeboost

import "github.com/pkg/errors"

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
	ErrDuplicateSequenceNumber  = errors.New("SUBMISSION_NONCE_ALREADY_SEEN")
	ErrSequenceNumberTooLow     = errors.New("SUBMISSION_NONCE_TOO_LOW")
	ErrTooManyBids              = errors.New("PER_ROUND_BID_LIMIT_REACHED")
)
