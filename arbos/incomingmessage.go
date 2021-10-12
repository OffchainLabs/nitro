package arbos

import (
	"bytes"
	"errors"
	"io"
	"math/big"

	"github.com/andybalholm/brotli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	solsha3 "github.com/miguelmota/go-solidity-sha3"
)

const (
	L1MessageType_L2Message             = 3
	L1MessageType_SetChainParams        = 4
	L1MessageType_EndOfBlock            = 6
	L1MessageType_L2FundedByL1          = 7
	L1MessageType_SubmitRetryable       = 9
	L1MessageType_BatchForGasEstimation = 10   // probably won't use this in practice
	L1MessageType_EthDeposit            = 11
)

const MaxL2MessageSize = 256*1024

func SplitInboxMessage(inputBytes []byte, chainId *big.Int) ([]*MessageSegment, error) {
	return ParseIncomingL1Message(bytes.NewReader(inputBytes), chainId)
}

type L1IncomingMessageHeader struct {
	kind        uint8
	sender      common.Address
	blockNumber common.Hash
	timestamp   common.Hash
	requestId   common.Hash
	gasPriceL1  common.Hash
}

type L1IncomingMessage struct {
	header *L1IncomingMessageHeader
	l2msg  []byte
}

type L1Info struct {
	l1Sender common.Address
	l1BlockNumber *big.Int
	l1Timestamp *big.Int
}

func (info *L1Info) Equals(o *L1Info) bool  {
	return info.l1Sender == o.l1Sender &&
		info.l1BlockNumber.Cmp(o.l1BlockNumber) == 0 &&
		info.l1Timestamp.Cmp(o.l1Timestamp) == 0
}

type MessageSegment struct {
	L1Info L1Info
	// l1GasPrice may be null
	l1GasPrice  *big.Int
	txes types.Transactions
}

func (msg *L1IncomingMessage) Serialize() ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := wr.WriteByte(msg.header.kind); err != nil {
		return nil, err
	}

	if err := AddressTo256ToWriter(msg.header.sender, wr); err != nil {
		return nil, err
	}

	if err := HashToWriter(msg.header.blockNumber, wr); err != nil {
		return nil, err
	}

	if err := HashToWriter(msg.header.timestamp, wr); err != nil {
		return nil, err
	}

	if err := HashToWriter(msg.header.requestId, wr); err != nil {
		return nil, err
	}

	if err := HashToWriter(msg.header.gasPriceL1, wr); err != nil {
		return nil, err
	}

	if _, err := wr.Write(msg.l2msg); err != nil {
		return nil, err
	}

	return wr.Bytes(), nil
}

func (msg *L1IncomingMessage) Equals(other *L1IncomingMessage) bool {
	return msg.header.Equals(other.header) && (bytes.Compare(msg.l2msg, other.l2msg) == 0)
}

func (header *L1IncomingMessageHeader) Equals(other *L1IncomingMessageHeader) bool {
	return (header.kind == other.kind) &&
		(header.sender.Hash().Big().Cmp(other.sender.Hash().Big()) == 0) &&
		(header.blockNumber.Big().Cmp(other.blockNumber.Big()) == 0) &&
		(header.timestamp.Big().Cmp(other.timestamp.Big()) == 0) &&
		(header.requestId.Big().Cmp(other.requestId.Big()) == 0) &&
		(header.gasPriceL1.Big().Cmp(other.gasPriceL1.Big()) == 0)
}

func ParseIncomingL1Message(rd io.Reader, chainId *big.Int) ([]*MessageSegment, error) {
	var kindBuf [1]byte
	_, err := rd.Read(kindBuf[:])
	if err != nil {
		return nil, err
	}

	sender, err := AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}

	blockNumber, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	timestamp, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	requestId, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	gasPriceL1, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	msg := &L1IncomingMessage{
		&L1IncomingMessageHeader{
			kindBuf[0],
			sender,
			blockNumber,
			timestamp,
			requestId,
			gasPriceL1,
		},
		data,
	}
	txes, err := msg.typeSpecificParse(chainId)
	if err != nil {
		return nil, err
	}
	return []*MessageSegment{
		{
			L1Info: L1Info{
				l1Sender:      sender,
				l1BlockNumber: blockNumber.Big(),
				l1Timestamp:   timestamp.Big(),
			},
			l1GasPrice:    gasPriceL1.Big(),
			txes:          txes,
		},
	}, nil
}

func (msg *L1IncomingMessage) typeSpecificParse(chainId *big.Int) (types.Transactions, error) {
	if len(msg.l2msg) > MaxL2MessageSize {
		// ignore the message if l2msg is too large
		return nil, errors.New("message too large")
	}
	switch msg.header.kind {
	case L1MessageType_L2Message:
		return parseL2Message(bytes.NewReader(msg.l2msg), msg.header.sender, msg.header.requestId, true)
	case L1MessageType_SetChainParams:
		panic("unimplemented")
	case L1MessageType_EndOfBlock:
		return nil, nil
	case L1MessageType_L2FundedByL1:
		panic("unimplemented")
	case L1MessageType_SubmitRetryable:
		panic("unimplemented")
	case L1MessageType_BatchForGasEstimation:
		panic("unimplemented")
	case L1MessageType_EthDeposit:
		tx, err := parseEthDepositMessage(bytes.NewReader(msg.l2msg), msg.header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	default:
		// invalid message, just ignore it
		return nil, errors.New("invalid message types")
	}
}

const (
	L2MessageKind_UnsignedUserTx = 0
	L2MessageKind_ContractTx = 1
	L2MessageKind_NonmutatingCall = 2
	L2MessageKind_Batch = 3
	L2MessageKind_SignedTx = 4
	// 5 is reserved
	L2MessageKind_Heartbeat = 6
	L2MessageKind_SignedCompressedTx = 7
	// 8 is reserved for BLS signed batch
	L2MessageKind_BrotliCompressed = 8

)

func parseL2Message(rd io.Reader, l1Sender common.Address, requestId common.Hash, isTopLevel bool) (types.Transactions, error) {
	var l2KindBuf [1]byte
	if _, err := rd.Read(l2KindBuf[:]); err != nil {
		return nil, err
	}

	switch l2KindBuf[0] {
	case L2MessageKind_UnsignedUserTx:
		tx, err := parseUnsignedTx(rd, l1Sender, requestId, true)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_ContractTx:
		tx, err := parseUnsignedTx(rd, l1Sender, requestId, false)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_NonmutatingCall:
		panic("unimplemented")
	case L2MessageKind_Batch:
		segments := make(types.Transactions, 0)
		index := big.NewInt(0)
		for {
			nextMsg, err := BytestringFromReader(rd)
			if err != nil {
				return segments, nil
			}
			nestedRequestIdSlice := solsha3.SoliditySHA3(solsha3.Bytes32(requestId), solsha3.Uint256(index))
			var nextRequestId common.Hash
			copy(nextRequestId[:], nestedRequestIdSlice)
			nestedSegments, err := parseL2Message(bytes.NewReader(nextMsg), l1Sender, nextRequestId, false)
			if err != nil {
				return nil, err
			}
			segments = append(segments, nestedSegments...)
			index.Add(index, big.NewInt(1))
		}
	case L2MessageKind_SignedTx:
		newTx := new(types.Transaction)
		if err := newTx.DecodeRLP(rlp.NewStream(rd, math.MaxUint64)); err != nil {
			return nil, err
		}
		return types.Transactions{newTx}, nil
	case L2MessageKind_Heartbeat:
		// do nothing
		return nil, nil
	case L2MessageKind_SignedCompressedTx:
		panic("unimplemented")
	case L2MessageKind_BrotliCompressed:
		if !isTopLevel { // ignore compressed messages if not top level
			return nil, errors.New("can only compress top level batch")
		}
		decompressed, err := io.ReadAll(io.LimitReader(brotli.NewReader(rd), MaxL2MessageSize))
		if err != nil {
			return nil, err
		}
		return parseL2Message(bytes.NewReader(decompressed), l1Sender, requestId, false)
	default:
		// ignore invalid message kind
		return nil, nil
	}
}

func parseUnsignedTx(rd io.Reader, l1Sender common.Address, requestId common.Hash, includesNonce bool) (*types.Transaction, error) {
	gasLimit, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	gasPrice, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	var nonce uint64
	if includesNonce {
		nonceAsHash, err := HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		nonce = nonceAsHash.Big().Uint64()
	}

	destAddr, err := AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	var destination *common.Address
	if destAddr.Hash().Big().Cmp(big.NewInt(0)) != 0 {
		destination = &destAddr
	}

	callvalue, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	calldata, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	var inner types.TxData

	if includesNonce {
		inner = &types.ArbitrumUnsignedTx{
			ChainId:   nil,
			From:      l1Sender,
			Nonce:     nonce,
			GasPrice:  gasPrice.Big(),
			Gas:       gasLimit.Big().Uint64(),
			To:        destination,
			Value:     callvalue.Big(),
			Data:      calldata,
		}
	} else {
		inner = &types.ArbitrumContractTx{
			ChainId:   nil,
			RequestId: requestId,
			From:      l1Sender,
			GasPrice:  gasPrice.Big(),
			Gas:       gasLimit.Big().Uint64(),
			To:        destination,
			Value:     callvalue.Big(),
			Data:      calldata,
		}
	}

	return types.NewTx(inner), nil
}

func parseEthDepositMessage(rd io.Reader, header *L1IncomingMessageHeader, chainId *big.Int) (*types.Transaction, error) {
	balance, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	tx := &types.DepositTx{
		ChainId:               chainId,
		L1RequestId: header.requestId,
		To:                    header.sender,
		Value:                 balance.Big(),
	}
	return types.NewTx(tx), nil
}
