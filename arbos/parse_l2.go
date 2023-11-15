package arbos

import (
	"bytes"
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
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type InfallibleBatchFetcher func(batchNum uint64, batchHash common.Hash) []byte

func ParseL2Transactions(msg *arbostypes.L1IncomingMessage, chainId *big.Int, batchFetcher InfallibleBatchFetcher) (types.Transactions, error) {
	if len(msg.L2msg) > arbostypes.MaxL2MessageSize {
		// ignore the message if l2msg is too large
		return nil, errors.New("message too large")
	}
	switch msg.Header.Kind {
	case arbostypes.L1MessageType_L2Message:
		return parseL2Message(bytes.NewReader(msg.L2msg), msg.Header.Poster, msg.Header.Timestamp, msg.Header.RequestId, chainId, 0)
	case arbostypes.L1MessageType_Initialize:
		return nil, errors.New("ParseL2Transactions encounted initialize message (should've been handled explicitly at genesis)")
	case arbostypes.L1MessageType_EndOfBlock:
		return nil, nil
	case arbostypes.L1MessageType_L2FundedByL1:
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
	case arbostypes.L1MessageType_SubmitRetryable:
		tx, err := parseSubmitRetryableMessage(bytes.NewReader(msg.L2msg), msg.Header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case arbostypes.L1MessageType_BatchForGasEstimation:
		return nil, errors.New("L1 message type BatchForGasEstimation is unimplemented")
	case arbostypes.L1MessageType_EthDeposit:
		tx, err := parseEthDepositMessage(bytes.NewReader(msg.L2msg), msg.Header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case arbostypes.L1MessageType_RollupEvent:
		log.Debug("ignoring rollup event message")
		return types.Transactions{}, nil
	case arbostypes.L1MessageType_BatchPostingReport:
		tx, err := parseBatchPostingReportMessage(bytes.NewReader(msg.L2msg), chainId, msg.BatchGasCost, batchFetcher)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case arbostypes.L1MessageType_Invalid:
		// intentionally invalid message
		return nil, errors.New("invalid message")
	default:
		// invalid message, just ignore it
		return nil, fmt.Errorf("invalid message type %v", msg.Header.Kind)
	}
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

func parseL2Message(rd io.Reader, poster common.Address, timestamp uint64, requestId *common.Hash, chainId *big.Int, depth int) (types.Transactions, error) {
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
			nextMsg, err := util.BytestringFromReader(rd, arbostypes.MaxL2MessageSize)
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
			nestedSegments, err := parseL2Message(bytes.NewReader(nextMsg), poster, timestamp, nextRequestId, chainId, depth+1)
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
		if newTx.Type() >= types.ArbitrumDepositTxType || newTx.Type() == types.BlobTxType {
			// Should be unreachable for Arbitrum types due to UnmarshalBinary not accepting Arbitrum internal txs
			// and we want to disallow BlobTxType since Arbitrum doesn't support EIP-4844 txs yet.
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

func parseEthDepositMessage(rd io.Reader, header *arbostypes.L1IncomingMessageHeader, chainId *big.Int) (*types.Transaction, error) {
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

func parseSubmitRetryableMessage(rd io.Reader, header *arbostypes.L1IncomingMessageHeader, chainId *big.Int) (*types.Transaction, error) {
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
	if dataLength > arbostypes.MaxL2MessageSize {
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

func parseBatchPostingReportMessage(rd io.Reader, chainId *big.Int, msgBatchGasCost *uint64, batchFetcher InfallibleBatchFetcher) (*types.Transaction, error) {
	batchTimestamp, batchPosterAddr, batchHash, batchNum, l1BaseFee, extraGas, err := arbostypes.ParseBatchPostingReportMessageFields(rd)
	if err != nil {
		return nil, err
	}
	var batchDataGas uint64
	if msgBatchGasCost != nil {
		batchDataGas = *msgBatchGasCost
	} else {
		batchData := batchFetcher(batchNum, batchHash)
		batchDataGas = arbostypes.ComputeBatchGasCost(batchData)
	}
	batchDataGas = arbmath.SaturatingUAdd(batchDataGas, extraGas)

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
