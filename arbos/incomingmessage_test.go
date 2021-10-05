package arbos

import (
	"bytes"
	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"io"
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

func TestBrotli(t *testing.T) {
	orig := []byte("This is a long and repetitive string. Yadda yadda yadda yadda yadda. The quick brown fox jumped over the lazy dog.")
	outBuf := bytes.Buffer{}
	bwr := brotli.NewWriter(&outBuf)
	_, err := bwr.Write(orig)
	if err != nil {
		t.Error(err)
	}
	bwr.Flush()
	compressed := outBuf.Bytes()
	if len(compressed) >= len(orig) {
		t.Fatal("compression didn't make it smaller")
	}
	decompressor := brotli.NewReader(bytes.NewReader(compressed))
	result, err := io.ReadAll(decompressor)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(orig, result) != 0 {
		t.Fatal("decompressed data doesn't match original")
	}
}