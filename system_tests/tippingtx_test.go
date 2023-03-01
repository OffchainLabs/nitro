package arbtest

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTippingTxBinaryMarshalling(t *testing.T) {
	address := common.HexToAddress("0xdeadbeef")
	dynamic := &types.DynamicFeeTx{
		To:        &address,
		Gas:       210000,
		GasFeeCap: big.NewInt(13),
		Value:     big.NewInt(8),
		Nonce:     44,
	}
	tipping := &types.ArbitrumTippingTx{DynamicFeeTx: *dynamic}
	dynamicTx := types.NewTx(dynamic)
	tippingTx := types.NewTx(tipping)
	dynamicBytes, err := dynamicTx.MarshalBinary()
	testhelpers.RequireImpl(t, err)
	tippingBytes, err := tippingTx.MarshalBinary()
	testhelpers.RequireImpl(t, err)
	t.Log("tipping:", tippingBytes)
	t.Log("dynamic:", dynamicBytes)
	if len(tippingBytes) == 0 {
		testhelpers.FailImpl(t, "got empty binary for tipping tx")
	}
	if tippingBytes[0] != types.ArbitrumTippingTxType {
		testhelpers.FailImpl(t, "got wrong first byte, want:", types.ArbitrumTippingTxType, "have:", tippingBytes[0])
	}
	if !bytes.Equal(tippingBytes[1:], dynamicBytes[1:]) {
		testhelpers.FailImpl(t, "unexpected tipping tx binary")
	}
	unmarshalledTx := new(types.Transaction)
	err = unmarshalledTx.UnmarshalBinary(tippingBytes)
	testhelpers.RequireImpl(t, err)
	if unmarshalledTx.Type() != types.ArbitrumTippingTxType {
		testhelpers.FailImpl(t, "unmarshalled unexpected tx type, want:", types.ArbitrumTippingTxType, "have:", unmarshalledTx.Type())
	}
}

func TestTippingTxSigning(t *testing.T) {
	info := NewL1TestInfo(t)
	info.GenerateAccount("tester")
	address := common.HexToAddress("0xdeadbeef")
	dynamic := &types.DynamicFeeTx{
		To:        &address,
		Gas:       210000,
		GasFeeCap: big.NewInt(13),
		Value:     big.NewInt(8),
		Nonce:     44,
	}
	tipping := &types.ArbitrumTippingTx{DynamicFeeTx: *dynamic}
	_ = info.SignTxAs("tester", tipping)
}
