// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	chainId := big.NewInt(6345634)
	requestId := common.BigToHash(big.NewInt(3))
	header := arbostypes.L1IncomingMessageHeader{
		Kind:        arbostypes.L1MessageType_EndOfBlock,
		Poster:      common.BigToAddress(big.NewInt(4684)),
		BlockNumber: 864513,
		Timestamp:   8794561564,
		RequestId:   &requestId,
		L1BaseFee:   big.NewInt(10000000000000),
	}
	msg := arbostypes.L1IncomingMessage{
		Header:       &header,
		L2msg:        []byte{3, 2, 1},
		BatchGasCost: nil,
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}
	newMsg, err := arbostypes.ParseIncomingL1Message(bytes.NewReader(serialized), nil)
	if err != nil {
		t.Error(err)
	}
	arbOSVersion := uint64(0)
	txes, err := ParseL2Transactions(newMsg, chainId, &arbOSVersion, nil)
	if err != nil {
		t.Error(err)
	}
	if len(txes) != 0 {
		Fail(t, "unexpected tx count")
	}
}
