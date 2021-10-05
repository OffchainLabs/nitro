package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	header := L1IncomingMessageHeader{
		L1MessageType_EndOfBlock,
		common.BigToAddress(big.NewInt(4684)),
		common.BigToHash(big.NewInt(864513)),
		common.BigToHash(big.NewInt(8794561564)),
		common.BigToHash(big.NewInt(3)),
		common.BigToHash(big.NewInt(10000000000000)),
	}
	msg := L1IncomingMessage{
		&header,
		[]byte{ 3, 2, 1 },
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}
	parsedMsg, err := ParseIncomingL1Message(bytes.NewReader(serialized))
	if err != nil {
		t.Error(err)
	}
	if ! msg.header.Equals(parsedMsg.header) {
		t.Fatal()
	}
	if ! msg.Equals(parsedMsg) {
		t.Fatal()
	}
}