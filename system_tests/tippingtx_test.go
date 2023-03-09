package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
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
	dynamicTx := types.NewTx(dynamic)
	tippingTx, err := types.NewArbitrumTippingTx(dynamicTx)
	testhelpers.RequireImpl(t, err)
	dynamicBytes, err := dynamicTx.MarshalBinary()
	testhelpers.RequireImpl(t, err)
	tippingBytes, err := tippingTx.MarshalBinary()
	testhelpers.RequireImpl(t, err)
	if len(tippingBytes) < 2 {
		testhelpers.FailImpl(t, "got too short binary for tipping tx")
	}
	if tippingBytes[0] != types.ArbitrumSubtypedTxType {
		testhelpers.FailImpl(t, "got wrong first byte (tx type), want:", types.ArbitrumSubtypedTxType, "got:", tippingBytes[0])
	}
	if tippingBytes[1] != types.ArbitrumTippingTxSubtype {
		testhelpers.FailImpl(t, "got wrong second byte (tx subtype), want:", types.ArbitrumTippingTxSubtype, "got:", tippingBytes[0])
	}
	if !bytes.Equal(tippingBytes[2:], dynamicBytes[1:]) {
		testhelpers.FailImpl(t, "unexpected tipping tx binary")
	}
	unmarshalledTx := new(types.Transaction)
	err = unmarshalledTx.UnmarshalBinary(tippingBytes)
	testhelpers.RequireImpl(t, err)
	if unmarshalledTx.Type() != types.ArbitrumSubtypedTxType {
		testhelpers.FailImpl(t, "unmarshalled unexpected tx type, want:", types.ArbitrumSubtypedTxType, "got:", unmarshalledTx.Type())
	}
	inner, ok := unmarshalledTx.GetInner().(*types.ArbitrumSubtypedTx)
	if !ok {
		testhelpers.FailImpl(t, "failed to get inner tx as ArbitrumSubtypedTx")
	}
	if types.GetArbitrumTxSubtype(unmarshalledTx) != types.ArbitrumTippingTxSubtype {
		testhelpers.FailImpl(t, "unmarshalled unexpected tx subtype, want:", types.ArbitrumTippingTxSubtype, "got:", unmarshalledTx.Type())
	}
	unmarshalledTipping, ok := inner.TxData.(*types.ArbitrumTippingTx)
	if !ok {
		testhelpers.FailImpl(t, "failed to cast inner TxData to ArbitrumTippingTx")
	}
	unmarshalledTippingBytes, err := types.NewTx(&unmarshalledTipping.DynamicFeeTx).MarshalBinary()
	testhelpers.RequireImpl(t, err)
	if !bytes.Equal(unmarshalledTippingBytes, dynamicBytes) {
		testhelpers.FailImpl(t, "unmarshalled tipping tx doesn't contain original DynamicFeeTx")
	}
}

func TestTippingTxJsonMarshalling(t *testing.T) {
	info := NewL1TestInfo(t)
	info.GenerateAccount("tester")
	address := common.HexToAddress("0xdeadbeef")
	accesses := types.AccessList{types.AccessTuple{
		Address: address,
		StorageKeys: []common.Hash{
			{0},
		},
	}}
	dynamic := types.DynamicFeeTx{
		ChainID:    big.NewInt(1337),
		To:         &address,
		Gas:        210000,
		GasFeeCap:  big.NewInt(13),
		GasTipCap:  big.NewInt(7),
		Value:      big.NewInt(8),
		AccessList: accesses,
		Nonce:      44,
		Data:       []byte{0xde, 0xad, 0xbe, 0xef},
	}
	tipping := &types.ArbitrumSubtypedTx{TxData: &types.ArbitrumTippingTx{DynamicFeeTx: dynamic}}
	tippingTx := info.SignTxAs("tester", tipping)
	tippingJson, err := tippingTx.MarshalJSON()
	testhelpers.RequireImpl(t, err)
	expectedJson := []byte(`{"type":"0x63","nonce":"0x2c","gasPrice":"0x0","maxPriorityFeePerGas":"0x7","maxFeePerGas":"0xd","gas":"0x33450","value":"0x8","input":"0xdeadbeef","v":"0x0","r":"0x61397aeca5e1059cda7c7ae10bf7fbf828fab81015c9eed4174d175406c1e6b6","s":"0xb56b52f6d874105b26cc19c6eaca899b5e26f1649845b774a32c5f8266a8d09","to":"0x00000000000000000000000000000000deadbeef","chainId":"0x539","accessList":[{"address":"0x00000000000000000000000000000000deadbeef","storageKeys":["0x0000000000000000000000000000000000000000000000000000000000000000"]}],"subtype":"0x1","hash":"0x904e53f088edb34cd09a10fbfc54f809ee840873fb262fd35a6d2434da24bae7"}`)
	if !bytes.Equal(tippingJson, expectedJson) {
		testhelpers.FailImpl(t, "Unexpected json result, want:", string(expectedJson), "got:", string(tippingJson))
	}
	var unmarshalledTx types.Transaction
	err = json.Unmarshal(tippingJson, &unmarshalledTx)
	Require(t, err)
	assertEqualTx(t, tippingTx, &unmarshalledTx)
}

func TestTippingTxJsonRPC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, l2node, l2client, _, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	l2info.GenerateAccount("User1")
	l2info.GenerateAccount("User2")
	SendWaitTestTransactions(t, ctx, l2client, []*types.Transaction{l2info.PrepareTx("Owner", "User1", l2info.TransferGas, big.NewInt(1e18), nil)})
	baseFee := GetBaseFee(t, l2client, ctx)
	tipCap := arbmath.BigMulByUint(baseFee, 2)
	gasPrice := arbmath.BigAdd(baseFee, tipCap)
	tx := l2info.PrepareTippingTx("User1", "User2", gasPrice.Uint64(), tipCap, big.NewInt(1e12), nil)
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	txByHash, _, err := l2client.TransactionByHash(ctx, tx.Hash())
	Require(t, err)
	assertEqualTx(t, tx, txByHash)
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
	tipping := &types.ArbitrumSubtypedTx{TxData: &types.ArbitrumTippingTx{DynamicFeeTx: *dynamic}}
	_ = info.SignTxAs("tester", tipping)
}

func TestTippingTxTipPaid(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2info, l2node, l2client, _, _, _, l1stack := createTestNodeOnL1(t, ctx, true)
	defer requireClose(t, l1stack)
	defer l2node.StopAndWait()

	callOpts := l2info.GetDefaultCallOpts("Owner", ctx)

	// get the network fee account
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err, "failed to deploy contract")
	arbGasInfo, err := precompilesgen.NewArbGasInfo(common.HexToAddress("0x6c"), l2client)
	Require(t, err, "failed to deploy contract")
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	l1Estimate, err := arbGasInfo.GetL1BaseFeeEstimate(callOpts)
	Require(t, err)

	baseFee := GetBaseFee(t, l2client, ctx)
	l2info.GasPrice = baseFee
	l2info.GenerateAccount("User1")
	l2info.GenerateAccount("User2")
	SendWaitTestTransactions(t, ctx, l2client, []*types.Transaction{l2info.PrepareTx("Owner", "User1", l2info.TransferGas, big.NewInt(1e18), nil)})
	compressedTxSize := func(t *testing.T, tx *types.Transaction) uint64 {
		txBin, err := tx.MarshalBinary()
		Require(t, err)
		compressed, err := arbcompress.CompressFast(txBin)
		Require(t, err)
		return uint64(len(compressed))
	}

	testFees := func(tip uint64) (*big.Int, *big.Int) {
		tipCap := arbmath.BigMulByUint(baseFee, tip)
		gasPrice := arbmath.BigAdd(baseFee, tipCap)
		networkBefore := GetBalance(t, ctx, l2client, networkFeeAccount)
		user1Before := GetBalance(t, ctx, l2client, l2info.GetAddress("User1"))
		user2Before := GetBalance(t, ctx, l2client, l2info.GetAddress("User2"))

		tx := l2info.PrepareTippingTx("User1", "User2", gasPrice.Uint64(), tipCap, big.NewInt(1e12), nil)
		err := l2client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)

		user1After := GetBalance(t, ctx, l2client, l2info.GetAddress("User2"))
		user1Paid := arbmath.BigSub(user1After, user1Before)
		user2After := GetBalance(t, ctx, l2client, l2info.GetAddress("User2"))
		user2Got := arbmath.BigSub(user2After, user2Before)

		if arbmath.BigEquals(user1Paid, arbmath.BigAdd(new(big.Int).SetUint64(receipt.GasUsed), user2Got)) {
			Fail(t, "after transfer balances sanity check failed")
		}

		networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
		l1Charge := arbmath.BigMulByUint(l2info.GasPrice, receipt.GasUsedForL1)

		// the network should receive
		//     1. compute costs
		//     2. tip on the compute costs
		//     3. tip on the data costs
		networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
		gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1
		feePaidForL2 := arbmath.BigMulByUint(gasPrice, gasUsedForL2)
		tipPaidToNet := arbmath.BigMulByUint(tipCap, receipt.GasUsedForL1)
		gotTip := arbmath.BigEquals(networkRevenue, arbmath.BigAdd(feePaidForL2, tipPaidToNet))
		if !gotTip {
			Fail(t, "network didn't receive expected payment", networkRevenue, feePaidForL2, tipPaidToNet)
		}
		txSize := compressedTxSize(t, tx)
		l1GasBought := arbmath.BigDiv(l1Charge, l1Estimate).Uint64()
		l1GasActual := txSize * params.TxDataNonZeroGasEIP2028

		colors.PrintBlue("bytes ", l1GasBought/params.TxDataNonZeroGasEIP2028, txSize)

		if l1GasBought != l1GasActual {
			Fail(t, "the sequencer's future revenue does not match its costs", l1GasBought, l1GasActual)
		}
		return networkRevenue, tipPaidToNet
	}

	net0, tip0 := testFees(0)
	net2, tip2 := testFees(2)

	if tip0.Sign() != 0 {
		Fail(t, "nonzero tip")
	}
	if arbmath.BigEquals(arbmath.BigSub(net2, tip2), net0) {
		Fail(t, "a tip of 2 should yield a total of 3")
	}
}

func assertEqualTx(t *testing.T, a, b *types.Transaction) {
	if want, got := a.Hash(), b.Hash(); want != got {
		testhelpers.FailImpl(t, "Unexpected unmarshalled tx, hash missmatch, want:", want, "got:", got)
	}
	if want, got := a.ChainId(), b.ChainId(); want.Cmp(got) != 0 {
		testhelpers.FailImpl(t, "Unexpected unmarshalled tx, chain id missmatch, want:", want, "got:", got)
	}
	if want, got := a.AccessList(), b.AccessList(); want != nil || got != nil {
		if !reflect.DeepEqual(want, got) {
			testhelpers.FailImpl(t, "Unexpected unmarshalled tx, access list missmatch")
		}
	}
}
