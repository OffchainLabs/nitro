//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbos

import (
	"bytes"
	"io"
	"math/big"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/ethereum/go-ethereum/common"
)

func TestSerializeAndParseL1Message(t *testing.T) {
	chainId := big.NewInt(6345634)
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
		[]byte{3, 2, 1},
	}
	serialized, err := msg.Serialize()
	if err != nil {
		t.Error(err)
	}
	newMsg, err := ParseIncomingL1Message(bytes.NewReader(serialized))
	if err != nil {
		t.Error(err)
	}
	segment, err := IncomingMessageToSegment(newMsg, chainId)
	if err != nil {
		t.Error(err)
	}
	if len(segment.Txes) != 0 {
		t.Fatal("unexpected tx count")
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
	err = bwr.Flush()
	if err != nil {
		t.Error(err)
	}
	compressed := outBuf.Bytes()
	if len(compressed) >= len(orig) {
		t.Fatal("compression didn't make it smaller")
	}
	decompressor := brotli.NewReader(bytes.NewReader(compressed))
	result, err := io.ReadAll(decompressor)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(orig, result) {
		t.Fatal("decompressed data doesn't match original")
	}
}
