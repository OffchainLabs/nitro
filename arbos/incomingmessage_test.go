package arbos

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestHeartbeatMessage(t *testing.T) {
	state := OpenArbosState(NewMemoryBackingEvmStorage(), common.Hash{})
	timestampBefore := state.LastTimestampSeen()

	msgBuf := &bytes.Buffer{}
	emitHeartbeatMessage(msgBuf, t)
	msg := bytes.NewReader(msgBuf.Bytes())
	if err := ParseAndHandleIncomingMessage(msg, state); err != nil {
		t.Error(err)
	}
	timestampAfter := state.LastTimestampSeen()

	if timestampBefore.Big().Cmp(timestampAfter.Big()) >= 0 {
		t.Fatalf("before: %v, after %v", timestampBefore.Big().String(), timestampAfter.Big().String())
	}

	if timestampAfter.Big().Cmp(IntToHash(100000).Big()) != 0 {
		t.Fail()
	}
}

func emitHeartbeatMessage(wr *bytes.Buffer, t *testing.T) {
	if err := wr.WriteByte(MessageType_Heartbeat); err != nil {
		t.Error(err)
	}
	if err := Uint64ToWriter(100, wr); err != nil {
		t.Error(err)
	}
	if err := HashToWriter(IntToHash(100000), wr); err != nil {
		t.Error(err)
	}
}

func TestEthDepositMessage(t *testing.T) {
	state := OpenArbosState(NewMemoryBackingEvmStorage(), common.Hash{})
	timestampBefore := state.LastTimestampSeen()

	msgBuf := &bytes.Buffer{}
	emitEthDepositMessage(msgBuf, t)
	msg := bytes.NewReader(msgBuf.Bytes())
	if err := ParseAndHandleIncomingMessage(msg, state); err != nil {
		t.Error(err)
	}
	timestampAfter := state.LastTimestampSeen()

	if timestampBefore.Big().Cmp(timestampAfter.Big()) >= 0 {
		t.Fatalf("before: %v, after %v", timestampBefore.Big().String(), timestampAfter.Big().String())
	}

	if timestampAfter.Big().Cmp(IntToHash(100000).Big()) != 0 {
		t.Fail()
	}
}

func emitEthDepositMessage(wr *bytes.Buffer, t *testing.T) {
	if err := wr.WriteByte(MessageType_EthDeposit); err != nil {
		t.Error(err)
	}
	if err := Uint64ToWriter(100, wr); err != nil {
		t.Error(err)
	}
	if err := HashToWriter(IntToHash(100000), wr); err != nil {
		t.Error(err)
	}
	addr := common.BigToAddress(big.NewInt(13980))
	if err := AddressToWriter(addr, wr); err != nil {
		t.Error(err)
	}
	balanceWei := IntToHash(8149280)
	if err := HashToWriter(balanceWei, wr); err != nil {
		t.Error(err)
	}
}

func TestTxNoNonceMessage(t *testing.T) {
	state := OpenArbosState(NewMemoryBackingEvmStorage(), common.Hash{})
	timestampBefore := state.LastTimestampSeen()

	msgBuf := &bytes.Buffer{}
	emitTxNoNonceMessage(msgBuf, t)
	msg := bytes.NewReader(msgBuf.Bytes())
	if err := ParseAndHandleIncomingMessage(msg, state); err != nil {
		t.Error(err)
	}
	timestampAfter := state.LastTimestampSeen()

	if timestampBefore.Big().Cmp(timestampAfter.Big()) >= 0 {
		t.Fatalf("before: %v, after %v", timestampBefore.Big().String(), timestampAfter.Big().String())
	}

	if timestampAfter.Big().Cmp(IntToHash(100000).Big()) != 0 {
		t.Fail()
	}
}

func emitTxNoNonceMessage(wr *bytes.Buffer, t *testing.T) {
	if err := wr.WriteByte(MessageType_TxNoNonce); err != nil {
		t.Error(err)
	}
	if err := Uint64ToWriter(100, wr); err != nil {
		t.Error(err)
	}
	if err := HashToWriter(IntToHash(100000), wr); err != nil {
		t.Error(err)
	}
	from := common.BigToAddress(big.NewInt(13980))
	if err := AddressToWriter(from, wr); err != nil {
		t.Error(err)
	}
	to := common.BigToAddress(big.NewInt(798789))
	if err := AddressToWriter(to, wr); err != nil {
		t.Error(err)
	}
	callvalueWei := IntToHash(8149280)
	if err := HashToWriter(callvalueWei, wr); err != nil {
		t.Error(err)
	}
	calldata := []byte("The quick brown fox jumped over the lazy dog.")
	if err := BytestringToWriter(calldata, wr); err != nil {
		t.Error(err)
	}
}