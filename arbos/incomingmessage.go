//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"errors"
	"io"
	"math/big"

	"github.com/andybalholm/brotli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	solsha3 "github.com/miguelmota/go-solidity-sha3"

	"github.com/offchainlabs/arbstate/arbos/util"
)

const (
	L1MessageType_L2Message             = 3
	L1MessageType_SetChainParams        = 4
	L1MessageType_EndOfBlock            = 6
	L1MessageType_L2FundedByL1          = 7
	L1MessageType_SubmitRetryable       = 9
	L1MessageType_BatchForGasEstimation = 10 // probably won't use this in practice
	L1MessageType_EthDeposit            = 11
	L1MessageType_Invalid               = 0xFF
)

const MaxL2MessageSize = 256 * 1024

type L1IncomingMessageHeader struct {
	Kind        uint8          `json:"kind"`
	Poster      common.Address `json:"sender"`
	BlockNumber common.Hash    `json:"blockNumber"`
	Timestamp   common.Hash    `json:"timestamp"`
	RequestId   common.Hash    `json:"requestId"`
	GasPriceL1  common.Hash    `json:"gasPriceL1"`
}

func (h L1IncomingMessageHeader) SeqNum() (uint64, error) {
	seqNumBig := h.RequestId.Big()
	if !seqNumBig.IsUint64() {
		return 0, errors.New("bad requestId")
	}
	return seqNumBig.Uint64(), nil
}

type L1IncomingMessage struct {
	Header *L1IncomingMessageHeader `json:"header"`
	L2msg  []byte                   `json:"l2Msg"`
}

type L1Info struct {
	poster        common.Address
	l1BlockNumber *big.Int
	l1Timestamp   *big.Int
}

func (info *L1Info) Equals(o *L1Info) bool {
	return info.poster == o.poster &&
		info.l1BlockNumber.Cmp(o.l1BlockNumber) == 0 &&
		info.l1Timestamp.Cmp(o.l1Timestamp) == 0
}

func (msg *L1IncomingMessage) Serialize() ([]byte, error) {
	wr := &bytes.Buffer{}
	if err := wr.WriteByte(msg.Header.Kind); err != nil {
		return nil, err
	}

	if err := util.AddressTo256ToWriter(msg.Header.Poster, wr); err != nil {
		return nil, err
	}

	if err := util.HashToWriter(msg.Header.BlockNumber, wr); err != nil {
		return nil, err
	}

	if err := util.HashToWriter(msg.Header.Timestamp, wr); err != nil {
		return nil, err
	}

	if err := util.HashToWriter(msg.Header.RequestId, wr); err != nil {
		return nil, err
	}

	if err := util.HashToWriter(msg.Header.GasPriceL1, wr); err != nil {
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

func (header *L1IncomingMessageHeader) Equals(other *L1IncomingMessageHeader) bool {
	return (header.Kind == other.Kind) &&
		(header.Poster.Hash().Big().Cmp(other.Poster.Hash().Big()) == 0) &&
		(header.BlockNumber.Big().Cmp(other.BlockNumber.Big()) == 0) &&
		(header.Timestamp.Big().Cmp(other.Timestamp.Big()) == 0) &&
		(header.RequestId.Big().Cmp(other.RequestId.Big()) == 0) &&
		(header.GasPriceL1.Big().Cmp(other.GasPriceL1.Big()) == 0)
}

func ParseIncomingL1Message(rd io.Reader) (*L1IncomingMessage, error) {
	var kindBuf [1]byte
	_, err := rd.Read(kindBuf[:])
	if err != nil {
		return nil, err
	}

	sender, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}

	blockNumber, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	timestamp, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	requestId, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	gasPriceL1, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	return &L1IncomingMessage{
		&L1IncomingMessageHeader{
			kindBuf[0],
			sender,
			blockNumber,
			timestamp,
			requestId,
			gasPriceL1,
		},
		data,
	}, nil
}

func (msg *L1IncomingMessage) ParseL2Transactions(chainId *big.Int) (types.Transactions, error) {
	if len(msg.L2msg) > MaxL2MessageSize {
		// ignore the message if l2msg is too large
		return nil, errors.New("message too large")
	}
	switch msg.Header.Kind {
	case L1MessageType_L2Message:
		return parseL2Message(bytes.NewReader(msg.L2msg), msg.Header.Poster, msg.Header.RequestId, 0)
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
		tx, err := parseEthDepositMessage(bytes.NewReader(msg.L2msg), msg.Header, chainId)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L1MessageType_Invalid:
		// intentionally invalid message
		return nil, errors.New("invalid message")
	default:
		// invalid message, just ignore it
		return nil, errors.New("invalid message types")
	}
}

const (
	L2MessageKind_UnsignedUserTx  = 0
	L2MessageKind_ContractTx      = 1
	L2MessageKind_NonmutatingCall = 2
	L2MessageKind_Batch           = 3
	L2MessageKind_SignedTx        = 4
	// 5 is reserved
	L2MessageKind_Heartbeat          = 6
	L2MessageKind_SignedCompressedTx = 7
	// 8 is reserved for BLS signed batch
	L2MessageKind_BrotliCompressed = 9
)

func parseL2Message(rd io.Reader, poster common.Address, requestId common.Hash, depth int) (types.Transactions, error) {
	var l2KindBuf [1]byte
	if _, err := rd.Read(l2KindBuf[:]); err != nil {
		return nil, err
	}

	switch l2KindBuf[0] {
	case L2MessageKind_UnsignedUserTx:
		tx, err := parseUnsignedTx(rd, poster, requestId, true)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_ContractTx:
		tx, err := parseUnsignedTx(rd, poster, requestId, false)
		if err != nil {
			return nil, err
		}
		return types.Transactions{tx}, nil
	case L2MessageKind_NonmutatingCall:
		panic("unimplemented")
	case L2MessageKind_Batch:
		if depth >= 16 {
			return nil, errors.New("L2 message batches have a max depth of 16")
		}
		segments := make(types.Transactions, 0)
		index := big.NewInt(0)
		for {
			nextMsg, err := util.BytestringFromReader(rd, MaxL2MessageSize)
			if err != nil {
				//lint:ignore nilerr an error here means there are no further messages in the batch
				return segments, nil
			}
			nestedRequestIdSlice := solsha3.SoliditySHA3(solsha3.Bytes32(requestId), solsha3.Uint256(index))
			var nextRequestId common.Hash
			copy(nextRequestId[:], nestedRequestIdSlice)
			nestedSegments, err := parseL2Message(bytes.NewReader(nextMsg), poster, nextRequestId, depth+1)
			if err != nil {
				return nil, err
			}
			segments = append(segments, nestedSegments...)
			index.Add(index, big.NewInt(1))
		}
	case L2MessageKind_SignedTx:
		newTx := new(types.Transaction)
		// Safe to read in its entirety, as all input readers are limited
		bytes, err := io.ReadAll(rd)
		if err != nil {
			return nil, err
		}
		if err := newTx.UnmarshalBinary(bytes); err != nil {
			return nil, err
		}
		return types.Transactions{newTx}, nil
	case L2MessageKind_Heartbeat:
		// do nothing
		return nil, nil
	case L2MessageKind_SignedCompressedTx:
		panic("unimplemented")
	case L2MessageKind_BrotliCompressed:
		if depth > 0 { // ignore compressed messages if not top level
			return nil, errors.New("can only compress top level batch")
		}
		reader := io.LimitReader(brotli.NewReader(rd), MaxL2MessageSize)
		return parseL2Message(reader, poster, requestId, depth+1)
	default:
		// ignore invalid message kind
		return nil, nil
	}
}

func parseUnsignedTx(rd io.Reader, poster common.Address, requestId common.Hash, includesNonce bool) (*types.Transaction, error) {
	gasLimit, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	gasPrice, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}

	var nonce uint64
	if includesNonce {
		nonceAsHash, err := util.HashFromReader(rd)
		if err != nil {
			return nil, err
		}
		nonce = nonceAsHash.Big().Uint64()
	}

	destAddr, err := util.AddressFrom256FromReader(rd)
	if err != nil {
		return nil, err
	}
	var destination *common.Address
	if destAddr.Hash().Big().Cmp(big.NewInt(0)) != 0 {
		destination = &destAddr
	}

	callvalue, err := util.HashFromReader(rd)
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
			ChainId:  nil,
			From:     util.RemapL1Address(poster),
			Nonce:    nonce,
			GasPrice: gasPrice.Big(),
			Gas:      gasLimit.Big().Uint64(),
			To:       destination,
			Value:    callvalue.Big(),
			Data:     calldata,
		}
	} else {
		inner = &types.ArbitrumContractTx{
			ChainId:   nil,
			RequestId: requestId,
			From:      util.RemapL1Address(poster),
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
	balance, err := util.HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	tx := &types.ArbitrumDepositTx{
		ChainId:     chainId,
		L1RequestId: header.RequestId,
		To:          header.Poster,
		Value:       balance.Big(),
	}
	return types.NewTx(tx), nil
}
