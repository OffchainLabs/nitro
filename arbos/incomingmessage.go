// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const (
	L1MessageType_L2Message             = 3
	L1MessageType_EndOfBlock            = 6
	L1MessageType_L2FundedByL1          = 7
	L1MessageType_RollupEvent           = 8
	L1MessageType_SubmitRetryable       = 9
	L1MessageType_BatchForGasEstimation = 10 // probably won't use this in practice
	L1MessageType_Initialize            = 11
	L1MessageType_EthDeposit            = 12
	L1MessageType_BatchPostingReport    = 13
	L1MessageType_Invalid               = 0xFF
)

const MaxL2MessageSize = 256 * 1024

type L1IncomingMessageHeader struct {
	Kind        uint8          `json:"kind"`
	Poster      common.Address `json:"sender"`
	BlockNumber uint64         `json:"blockNumber"`
	Timestamp   uint64         `json:"timestamp"`
	RequestId   *common.Hash   `json:"requestId" rlp:"nilList"`
	L1BaseFee   *big.Int       `json:"baseFeeL1"`
}

func (h L1IncomingMessageHeader) SeqNum() (uint64, error) {
	if h.RequestId == nil {
		return 0, errors.New("no requestId")
	}
	seqNumBig := h.RequestId.Big()
	if !seqNumBig.IsUint64() {
		return 0, errors.New("bad requestId")
	}
	return seqNumBig.Uint64(), nil
}

type L1IncomingMessage struct {
	Header *L1IncomingMessageHeader `json:"header"`
	L2msg  []byte                   `json:"l2Msg"`

	// Only used for `L1MessageType_BatchPostingReport`
	BatchGasCost *uint64 `json:"batchGasCost,omitempty" rlp:"optional"`
}

var EmptyTestIncomingMessage = L1IncomingMessage{
	Header: &L1IncomingMessageHeader{},
}

var TestIncomingMessageWithRequestId = L1IncomingMessage{
	Header: &L1IncomingMessageHeader{
		Kind:      L1MessageType_Invalid,
		RequestId: &common.Hash{},
		L1BaseFee: big.NewInt(0),
	},
}

type L1Info struct {
	poster        common.Address
	l1BlockNumber uint64
	l1Timestamp   uint64
}

func (info *L1Info) Equals(o *L1Info) bool {
	return info.poster == o.poster && info.l1BlockNumber == o.l1BlockNumber && info.l1Timestamp == o.l1Timestamp
}

func (msg *L1IncomingMessage) Serialize() ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := wr.WriteByte(msg.Header.Kind); err != nil {
		return nil, err
	}

	if err := util.AddressTo256ToWriter(msg.Header.Poster, wr); err != nil {
		return nil, err
	}

	if err := util.Uint64ToWriter(msg.Header.BlockNumber, wr); err != nil {
		return nil, err
	}

	if err := util.Uint64ToWriter(msg.Header.Timestamp, wr); err != nil {
		return nil, err
	}

	if msg.Header.RequestId == nil {
		return nil, errors.New("cannot serialize L1IncomingMessage without RequestId")
	}
	requestId := *msg.Header.RequestId
	if err := util.HashToWriter(requestId, wr); err != nil {
		return nil, err
	}

	var l1BaseFeeHash common.Hash
	if msg.Header.L1BaseFee == nil {
		return nil, errors.New("cannot serialize L1IncomingMessage without L1BaseFee")
	}
	l1BaseFeeHash = common.BigToHash(msg.Header.L1BaseFee)
	if err := util.HashToWriter(l1BaseFeeHash, wr); err != nil {
		return nil, err
	}

	if _, err := wr.Write(msg.L2msg); err != nil {
		return nil, err
	}

	return wr.Bytes(), nil
}

func (msg *L1IncomingMessage) Equals(other *L1IncomingMessage) bool {
	return msg.Header.Equals(other.Header) && bytes.Equal(msg.L2msg, other.L2msg)
}

func (h *L1IncomingMessageHeader) Equals(other *L1IncomingMessageHeader) bool {
	// These are all non-pointer types so it's safe to use the == operator
	return h.Kind == other.Kind &&
		h.Poster == other.Poster &&
		h.BlockNumber == other.BlockNumber &&
		h.Timestamp == other.Timestamp &&
		h.RequestId == other.RequestId &&
		h.L1BaseFee == other.L1BaseFee
}

func (msg *L1IncomingMessage) FillInBatchGasCost(batchFetcher FallibleBatchFetcher) error {
	if batchFetcher == nil || msg.Header.Kind != L1MessageType_BatchPostingReport || msg.BatchGasCost != nil {
		return nil
	}
	_, _, batchHash, batchNum, _, err := parseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
	if err != nil {
		return fmt.Errorf("failed to parse batch posting report: %w", err)
	}
	batchData, err := batchFetcher(batchNum)
	if err != nil {
		return fmt.Errorf("failed to fetch batch mentioned by batch posting report: %w", err)
	}
	gotHash := crypto.Keccak256Hash(batchData)
	if gotHash != batchHash {
		return fmt.Errorf("batch fetcher returned incorrect data hash %v (wanted %v for batch %v)", gotHash, batchHash, batchNum)
	}
	gas := computeBatchGasCost(batchData)
	msg.BatchGasCost = &gas
	return nil
}

func ParseIncomingL1Message(rd io.Reader, batchFetcher FallibleBatchFetcher) (*L1IncomingMessage, error) {
	var kindBuf [1]byte
	_, err := rd.Read(kindBuf[:])
	if err != nil {
		return nil, err
	}
	kind := kindBuf[0]

	sender, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}

	blockNumber, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}

	timestamp, err := util.Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}

	requestId, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	baseFeeL1, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	msg := &L1IncomingMessage{
		&L1IncomingMessageHeader{
			kind,
			sender,
			blockNumber,
			timestamp,
			&requestId,
			baseFeeL1.Big(),
		},
		data,
		nil,
	}
	err = msg.FillInBatchGasCost(batchFetcher)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type InfallibleBatchFetcher func(batchNum uint64, batchHash common.Hash) []byte

func (msg *L1IncomingMessage) ParseL2Transactions(chainId *big.Int, arbOSVersion uint64, batchFetcher InfallibleBatchFetcher) (types.Transactions, error) {
	if len(msg.L2msg) > MaxL2MessageSize {
		// ignore the message if l2msg is too large
		return nil, errors.New("message too large")
	}
	switch msg.Header.Kind {
	case L1MessageType_L2Message:
		return parseL2Message(bytes.NewReader(msg.L2msg), msg.Header.Poster, msg.Header.Timestamp, msg.Header.RequestId, chainId, 0, arbOSVersion)
	case L1MessageType_Initialize:
		return nil, errors.New("ParseL2Transactions encounted initialize message (should've been handled explicitly at genesis)")
	case L1MessageType_EndOfBlock:
		return nil, nil
	case L1MessageType_L2FundedByL1:
		if len(msg.L2msg) < 1 {
			return nil, errors.New("L2FundedByL1 message has no data")
		}
		if msg.Header.RequestId == nil {
			return nil, errors.New("cannot issue L2 funded by L1 tx without L1 request id")
		}
		kind := msg.L2msg[0]
		depositRequestId := crypto.Keccak256Hash(msg.Header.RequestId[:], math.U256Bytes(common.Big0))
		unsignedRequestId := crypto.Keccak256Hash(msg.Header.RequestId[:], math.U256Bytes(common.Big1))
		tx, err := parseUnsignedTx(bytes.NewReader(msg.L2msg[1:]), msg.Header.Poster, &unsignedRequestId, chainId, kind)
		if err != nil {
			return nil, err
		}
		deposit := types.NewTx(&types.ArbitrumDepositTx{
			ChainId:     chainId,
			L1RequestId: depositRequestId,
			// Matches the From of parseUnsignedTx
			To:    msg.Header.Poster,
			Value: tx.Value(),
		})
		return types.Transactions{deposit, tx}, nil
	case L1MessageType_SubmitRetryable:
		tx, err := parseSubmitRetryableMessage(bytes.NewReader(msg.L2msg), msg.Header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L1MessageType_BatchForGasEstimation:
		return nil, errors.New("L1 message type BatchForGasEstimation is unimplemented")
	case L1MessageType_EthDeposit:
		tx, err := parseEthDepositMessage(bytes.NewReader(msg.L2msg), msg.Header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L1MessageType_RollupEvent:
		log.Debug("ignoring rollup event message")
		return types.Transactions{}, nil
	case L1MessageType_BatchPostingReport:
		tx, err := parseBatchPostingReportMessage(bytes.NewReader(msg.L2msg), chainId, msg.BatchGasCost, batchFetcher)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L1MessageType_Invalid:
		// intentionally invalid message
		return nil, errors.New("invalid message")
	default:
		// invalid message, just ignore it
		return nil, fmt.Errorf("invalid message type %v", msg.Header.Kind)
	}
}

// ParseInitMessage returns the chain id on success
func (msg *L1IncomingMessage) ParseInitMessage() (*big.Int, error) {
	if msg.Header.Kind != L1MessageType_Initialize {
		return nil, fmt.Errorf("invalid init message kind %v", msg.Header.Kind)
	}
	if len(msg.L2msg) != 32 {
		return nil, fmt.Errorf("invalid init message data %v", hex.EncodeToString(msg.L2msg))
	}
	chainId := new(big.Int).SetBytes(msg.L2msg[:32])
	return chainId, nil
}

const (
	L2MessageKind_UnsignedUserTx  = 0
	L2MessageKind_ContractTx      = 1
	L2MessageKind_NonmutatingCall = 2
	L2MessageKind_Batch           = 3
	L2MessageKind_SignedTx        = 4
	// 5 is reserved
	L2MessageKind_Heartbeat          = 6 // deprecated
	L2MessageKind_SignedCompressedTx = 7
	// 8 is reserved for BLS signed batch
)

// Warning: this does not validate the day of the week or if DST is being observed
func parseTimeOrPanic(format string, value string) time.Time {
	t, err := time.Parse(format, value)
	if err != nil {
		panic(err)
	}
	return t
}

var HeartbeatsDisabledAt = uint64(parseTimeOrPanic(time.RFC1123, "Mon, 08 Aug 2022 16:00:00 GMT").Unix())

func parseL2Message(rd io.Reader, poster common.Address, timestamp uint64, requestId *common.Hash, chainId *big.Int, depth int, arbOSVersion uint64) (types.Transactions, error) {
	var l2KindBuf [1]byte
	if _, err := rd.Read(l2KindBuf[:]); err != nil {
		return nil, err
	}

	switch l2KindBuf[0] {
	case L2MessageKind_UnsignedUserTx:
		tx, err := parseUnsignedTx(rd, poster, requestId, chainId, L2MessageKind_UnsignedUserTx)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_ContractTx:
		tx, err := parseUnsignedTx(rd, poster, requestId, chainId, L2MessageKind_ContractTx)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_NonmutatingCall:
		return nil, errors.New("L2 message kind NonmutatingCall is unimplemented")
	case L2MessageKind_Batch:
		if depth >= 16 {
			return nil, errors.New("L2 message batches have a max depth of 16")
		}
		segments := make(types.Transactions, 0)
		index := big.NewInt(0)
		for {
			nextMsg, err := util.BytestringFromReader(rd, MaxL2MessageSize)
			if err != nil {
				// an error here means there are no further messages in the batch
				// nolint:nilerr
				return segments, nil
			}

			var nextRequestId *common.Hash
			if requestId != nil {
				subRequestId := crypto.Keccak256Hash(requestId[:], math.U256Bytes(index))
				nextRequestId = &subRequestId
			}
			nestedSegments, err := parseL2Message(bytes.NewReader(nextMsg), poster, timestamp, nextRequestId, chainId, depth+1, arbOSVersion)
			if err != nil {
				return nil, err
			}
			segments = append(segments, nestedSegments...)
			index.Add(index, big.NewInt(1))
		}
	case L2MessageKind_SignedTx:
		newTx := new(types.Transaction)
		// Safe to read in its entirety, as all input readers are limited
		readBytes, err := io.ReadAll(rd)
		if err != nil {
			return nil, err
		}
		if err := newTx.UnmarshalBinary(readBytes); err != nil {
			return nil, err
		}
		if newTx.Type() == types.ArbitrumTippingTxType && arbOSVersion < 11 {
			return nil, types.ErrTxTypeNotSupported
		}
		if newTx.Type() >= types.ArbitrumDepositTxType {
			// Should be unreachable due to UnmarshalBinary not accepting Arbitrum internal txs
			return nil, types.ErrTxTypeNotSupported
		}
		return types.Transactions{newTx}, nil
	case L2MessageKind_Heartbeat:
		if timestamp >= HeartbeatsDisabledAt {
			return nil, errors.New("heartbeat messages have been disabled")
		}
		// do nothing
		return nil, nil
	case L2MessageKind_SignedCompressedTx:
		return nil, errors.New("L2 message kind SignedCompressedTx is unimplemented")
	default:
		// ignore invalid message kind
		return nil, fmt.Errorf("unkown L2 message kind %v", l2KindBuf[0])
	}
}

func parseUnsignedTx(rd io.Reader, poster common.Address, requestId *common.Hash, chainId *big.Int, txKind byte) (*types.Transaction, error) {
	gasLimitHash, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	gasLimitBig := gasLimitHash.Big()
	if !gasLimitBig.IsUint64() {
		return nil, errors.New("unsigned user tx gas limit >= 2^64")
	}
	gasLimit := gasLimitBig.Uint64()

	maxFeePerGas, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	var nonce uint64
	if txKind == L2MessageKind_UnsignedUserTx {
		nonceAsHash, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		nonceAsBig := nonceAsHash.Big()
		if !nonceAsBig.IsUint64() {
			return nil, errors.New("unsigned user tx nonce >= 2^64")
		}
		nonce = nonceAsBig.Uint64()
	}

	to, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	var destination *common.Address
	if to != (common.Address{}) {
		destination = &to
	}

	value, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	calldata, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	var inner types.TxData

	switch txKind {
	case L2MessageKind_UnsignedUserTx:
		inner = &types.ArbitrumUnsignedTx{
			ChainId:   chainId,
			From:      poster,
			Nonce:     nonce,
			GasFeeCap: maxFeePerGas.Big(),
			Gas:       gasLimit,
			To:        destination,
			Value:     value.Big(),
			Data:      calldata,
		}
	case L2MessageKind_ContractTx:
		if requestId == nil {
			return nil, errors.New("cannot issue contract tx without L1 request id")
		}
		inner = &types.ArbitrumContractTx{
			ChainId:   chainId,
			RequestId: *requestId,
			From:      poster,
			GasFeeCap: maxFeePerGas.Big(),
			Gas:       gasLimit,
			To:        destination,
			Value:     value.Big(),
			Data:      calldata,
		}
	default:
		return nil, errors.New("invalid L2 tx type in parseUnsignedTx")
	}

	return types.NewTx(inner), nil
}

func parseEthDepositMessage(rd io.Reader, header *L1IncomingMessageHeader, chainId *big.Int) (*types.Transaction, error) {
	to, err := util.AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	balance, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	if header.RequestId == nil {
		return nil, errors.New("cannot issue deposit tx without L1 request id")
	}
	tx := &types.ArbitrumDepositTx{
		ChainId:     chainId,
		L1RequestId: *header.RequestId,
		From:        header.Poster,
		To:          to,
		Value:       balance.Big(),
	}
	return types.NewTx(tx), nil
}

func parseSubmitRetryableMessage(rd io.Reader, header *L1IncomingMessageHeader, chainId *big.Int) (*types.Transaction, error) {
	retryTo, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	pRetryTo := &retryTo
	if retryTo == (common.Address{}) {
		pRetryTo = nil
	}
	callvalue, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	depositValue, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	maxSubmissionFee, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	feeRefundAddress, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	callvalueRefundAddress, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	gasLimit, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	gasLimitBig := gasLimit.Big()
	if !gasLimitBig.IsUint64() {
		return nil, errors.New("gas limit too large")
	}
	maxFeePerGas, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	dataLength256, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	dataLengthBig := dataLength256.Big()
	if !dataLengthBig.IsUint64() {
		return nil, errors.New("data length field too large")
	}
	dataLength := dataLengthBig.Uint64()
	if dataLength > MaxL2MessageSize {
		return nil, errors.New("retryable data too large")
	}
	retryData := make([]byte, dataLength)
	if dataLength > 0 {
		if _, err := rd.Read(retryData); err != nil {
			return nil, err
		}
	}
	if header.RequestId == nil {
		return nil, errors.New("cannot issue submit retryable tx without L1 request id")
	}
	tx := &types.ArbitrumSubmitRetryableTx{
		ChainId:          chainId,
		RequestId:        *header.RequestId,
		From:             header.Poster,
		L1BaseFee:        header.L1BaseFee,
		DepositValue:     depositValue.Big(),
		GasFeeCap:        maxFeePerGas.Big(),
		Gas:              gasLimitBig.Uint64(),
		RetryTo:          pRetryTo,
		RetryValue:       callvalue.Big(),
		Beneficiary:      callvalueRefundAddress,
		MaxSubmissionFee: maxSubmissionFee.Big(),
		FeeRefundAddr:    feeRefundAddress,
		RetryData:        retryData,
	}
	return types.NewTx(tx), err
}

func parseBatchPostingReportMessageFields(rd io.Reader) (*big.Int, common.Address, common.Hash, uint64, *big.Int, error) {
	batchTimestamp, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, err
	}
	batchPosterAddr, err := util.AddressFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, err
	}
	dataHash, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, err
	}
	batchNum, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, err
	}
	l1BaseFee, err := util.HashFromReader(rd)
	if err != nil {
		return nil, common.Address{}, common.Hash{}, 0, nil, err
	}
	batchNumBig := batchNum.Big()
	if !batchNumBig.IsUint64() {
		return nil, common.Address{}, common.Hash{}, 0, nil, fmt.Errorf("batch number %v is not a uint64", batchNumBig)
	}
	return batchTimestamp.Big(), batchPosterAddr, dataHash, batchNumBig.Uint64(), l1BaseFee.Big(), nil
}

func computeBatchGasCost(data []byte) uint64 {
	var gas uint64
	for _, b := range data {
		if b == 0 {
			gas += params.TxDataZeroGas
		} else {
			gas += params.TxDataNonZeroGasEIP2028
		}
	}

	// the poster also pays to keccak the batch and place it and a batch-posting report into the inbox
	keccakWords := arbmath.WordsForBytes(uint64(len(data)))
	gas += params.Keccak256Gas + (keccakWords * params.Keccak256WordGas)
	gas += 2 * params.SstoreSetGasEIP2200
	return gas
}

func parseBatchPostingReportMessage(rd io.Reader, chainId *big.Int, msgBatchGasCost *uint64, batchFetcher InfallibleBatchFetcher) (*types.Transaction, error) {
	batchTimestamp, batchPosterAddr, batchHash, batchNum, l1BaseFee, err := parseBatchPostingReportMessageFields(rd)
	if err != nil {
		return nil, err
	}
	var batchDataGas uint64
	if msgBatchGasCost != nil {
		batchDataGas = *msgBatchGasCost
	} else {
		batchData := batchFetcher(batchNum, batchHash)
		batchDataGas = computeBatchGasCost(batchData)
	}

	data, err := util.PackInternalTxDataBatchPostingReport(
		batchTimestamp, batchPosterAddr, batchNum, batchDataGas, l1BaseFee,
	)
	if err != nil {
		return nil, err
	}
	return types.NewTx(&types.ArbitrumInternalTx{
		ChainId: chainId,
		Data:    data,
		// don't need to fill in the other fields, since they exist only to ensure uniqueness, and batchNum is already unique
	}), nil
}
