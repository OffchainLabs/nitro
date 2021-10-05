package arbos

import (
	"bytes"
	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"io"
)

const (
	L1MessageType_L2Message             = 3
	L1MessageType_SetChainParams        = 4
	L1MessageType_EndOfBlock            = 6
	L1MessageType_L2FundedByL1          = 7
	L1MessageType_SubmitRetryable       = 9
	L1MessageType_BatchForGasEstimation = 10
)

type IncomingMessage interface {
	handle(state *ArbosState)
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

func ParseIncomingL1Message(rd io.Reader) (*L1IncomingMessage, error) {
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

func (msg *L1IncomingMessage) handle(state *ArbosState) {
	switch msg.header.kind {
	case L1MessageType_L2Message:
		parseAndHandleL2Message(bytes.NewReader(msg.l2msg), msg.header)
	case L1MessageType_SetChainParams:
		panic("unimplemented")
	case L1MessageType_EndOfBlock:
		panic("unimplemented")
	case L1MessageType_L2FundedByL1:
		panic("unimplemented")
	case L1MessageType_SubmitRetryable:
		panic("unimplemented")
	case L1MessageType_BatchForGasEstimation:
		panic("unimplemented")
	default:
		// invalid message, just ignore it
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

func parseAndHandleL2Message(rd io.Reader, header *L1IncomingMessageHeader) {
	var l2KindBuf [1]byte
	if _, err := rd.Read(l2KindBuf[:]); err != nil {
		return
	}

	switch(l2KindBuf[0]) {
	case L2MessageKind_UnsignedUserTx:
		handleUnsignedTx(rd, header, true)
	case L2MessageKind_ContractTx:
		handleUnsignedTx(rd, header, false)
	case L2MessageKind_NonmutatingCall:
		panic("unimplemented")
	case L2MessageKind_Batch:
		for {
			nextMsg, err := BytestringFromReader(rd)
			if err != nil {
				return
			}
			parseAndHandleL2Message(bytes.NewReader(nextMsg), header)
		}
	case L2MessageKind_SignedTx:
		panic("unimplemented")
	case L2MessageKind_Heartbeat:
		// do nothing
	case L2MessageKind_SignedCompressedTx:
		panic("unimplemented")
	case L2MessageKind_BrotliCompressed:
		parseAndHandleL2Message(brotli.NewReader(rd), header)
	default:
		// ignore invalid message kind
	}
}

func handleUnsignedTx(rd io.Reader, header *L1IncomingMessageHeader, includesNonce bool) {
	gasLimit, err := HashFromReader(rd)
	if err != nil {
		return
	}

	gasPrice, err := HashFromReader(rd)
	if err != nil {
		return
	}

	var nonce common.Hash
	if includesNonce {
		nonce, err = HashFromReader(rd)
		if err != nil {
			return
		}
	}

	destination, err := AddressFrom256FromReader(rd)
	if err != nil {
		return
	}

	callvalue, err := HashFromReader(rd)
	if err != nil {
		return
	}

	calldata, err := io.ReadAll(rd)
	if err != nil {
		return
	}

	// keep the compiler from erroring for unused variables
	_ = gasLimit
	_ = gasPrice
	_ = nonce
	_ = destination
	_ = callvalue
	_ = calldata

	//TODO: send transaction to Geth for execution
}
