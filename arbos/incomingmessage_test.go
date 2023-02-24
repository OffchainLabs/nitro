// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	chainId := big.NewInt(6345634)
	requestId := common.BigToHash(big.NewInt(3))
	header := L1IncomingMessageHeader{
		L1MessageType_EndOfBlock,
		common.BigToAddress(big.NewInt(4684)),
		864513,
		8794561564,
		&requestId,
		big.NewInt(10000000000000),
	}
	msg := L1IncomingMessage{
		&header,
		[]byte{3, 2, 1},
		nil,
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}
	newMsg, err := ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Error(err)
	}
	txes, err := newMsg.ParseL2Transactions(chainId, 0, nil)
	if err != nil {
		t.Error(err)
	}
	if len(txes) != 0 {
		Fail(t, "unexpected tx count")
	}
}
