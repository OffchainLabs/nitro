package arbos

import (
	"github.com/ethereum/go-ethereum/common"
	"io"
)

const (
	MessageType_Heartbeat  = 0
	MessageType_EthDeposit = 1
	MessageType_TxNoNonce  = 2
)

type IncomingMessage interface {
	handle(storage *ArbosState)
}

func ParseAndHandleIncomingMessage(rd io.Reader, state *ArbosState) error {
	var typeBuf [1]byte
	_, err := rd.Read(typeBuf[:])
	if err != nil {
		return err
	}
	var message IncomingMessage
	switch typeBuf[0] {
	case MessageType_Heartbeat:
		message, err = ParseHeartbeatMessage(rd)
		if err != nil {
			return err
		}
	case MessageType_EthDeposit:
		message, err = ParseEthDepositMessage(rd)
		if err != nil {
			return err
		}
	case MessageType_TxNoNonce:
		message, err = ParseTxMessageNoNonce(rd)
		if err != nil {
			return err
		}
	}
	message.handle(state)
	return nil
}

type HeartbeatMessage struct {
	ethBlockNumber uint64
	timestamp      common.Hash
}

func ParseHeartbeatMessage(rd io.Reader) (IncomingMessage, error) {
	ethBlockNumber, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	timestamp, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	return &HeartbeatMessage{ethBlockNumber, timestamp}, nil
}

func (msg *HeartbeatMessage) handle(state *ArbosState) {
	state.AdvanceTimestampToAtLeast(msg.timestamp)
}

type EthDepositMessage struct {
	ethBlockNumber uint64
	timestamp      common.Hash
	account        common.Address
	balanceWei     common.Hash
}

func ParseEthDepositMessage(rd io.Reader) (IncomingMessage, error) {
	ethBlockNumber, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	timestamp, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	account, err := AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	balanceWei, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	return &EthDepositMessage{
		ethBlockNumber,
		timestamp,
		account,
		balanceWei,
	}, nil
}

func (msg *EthDepositMessage) handle(state *ArbosState) {
	state.AdvanceTimestampToAtLeast(msg.timestamp)
	//TODO: deposit funds into geth state
}

type TxMessageNoNonce struct {
	ethBlockNumber uint64
	timestamp      common.Hash
	from           common.Address
	to             common.Address // zero means it's a constructor
	callvalueWei   common.Hash
	calldata       []byte
}

func ParseTxMessageNoNonce(rd io.Reader) (IncomingMessage, error) {
	ethBlockNumber, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	timestamp, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	from, err := AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	to, err := AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	callvalueWei, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	calldata, err := BytestringFromReader(rd)
	if err != nil {
		return nil, err
	}
	return &TxMessageNoNonce{
		ethBlockNumber,
		timestamp,
		from,
		to,
		callvalueWei,
		calldata,
	}, nil
}

func (msg *TxMessageNoNonce) handle(state *ArbosState) {
	state.AdvanceTimestampToAtLeast(msg.timestamp)
	//TODO: dispatch to geth to execute the message
}
