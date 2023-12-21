package arbtest

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
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
	if len(tippingBytes) < 3 {
		testhelpers.FailImpl(t, "got too short binary for tipping tx")
	}
	if tippingBytes[0] != types.ArbitrumSubtypedTxType {
		testhelpers.FailImpl(t, "got wrong first byte (tx type), want:", types.ArbitrumSubtypedTxType, "got:", tippingBytes[0])
	}
	if tippingBytes[1] != types.ArbitrumTippingTxSubtype {
		testhelpers.FailImpl(t, "got wrong second byte (tx subtype), want:", types.ArbitrumTippingTxSubtype, "got:", tippingBytes[0])
	}
	if !bytes.Equal(tippingBytes[3:], dynamicBytes[1:]) {
		testhelpers.FailImpl(t, "unexpected tipping tx binary")
	}
	if int(tippingBytes[2])-0xc0 != len(tippingBytes[3:]) {
		testhelpers.FailImpl(t, "got unexpected list header, have:", int(tippingBytes[2])-0xc0, "want", len(tippingBytes[3:]))
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
	info := NewArbTestInfo(t, params.ArbitrumDevTestChainConfig().ChainID)
	info.GenerateAccount("tester")
	address := common.HexToAddress("0xdeadbeef")
	accesses := types.AccessList{types.AccessTuple{
		Address: address,
		StorageKeys: []common.Hash{
			{0},
		},
	}}
	dynamic := types.DynamicFeeTx{
		ChainID:    params.ArbitrumDevTestChainConfig().ChainID,
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
	expectedJson := []byte(`{"type":"0x63","chainId":"0x64aba","nonce":"0x2c","to":"0x00000000000000000000000000000000deadbeef","gas":"0x33450","gasPrice":"0x0","maxPriorityFeePerGas":"0x7","maxFeePerGas":"0xd","value":"0x8","input":"0xdeadbeef","accessList":[{"address":"0x00000000000000000000000000000000deadbeef","storageKeys":["0x0000000000000000000000000000000000000000000000000000000000000000"]}],"v":"0x0","r":"0x310d9c5e7be1c093fcba664e903b53db3e9f79feb829c54723438093aa6b8081","s":"0x54520686b9955d60c1e0f0af3922fdd978b952464e5dd652b002e8439a207cda","subtype":"0x1","hash":"0xe0d0cb6ce7c1fd9487d19391878e28fdef953efb7fb892f7d1834d2a0482cef5"}`)
	if !bytes.Equal(tippingJson, expectedJson) {
		testhelpers.FailImpl(t, "Unexpected json result, want:\n\t", string(expectedJson), "\ngot:\n\t", string(tippingJson))
	}
	var unmarshalledTx types.Transaction
	err = json.Unmarshal(tippingJson, &unmarshalledTx)
	Require(t, err)
	assertEqualTx(t, tippingTx, &unmarshalledTx)
}

func TestTippingTxJsonRPC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	// make sure ArbOSVersion supports ArbitrumSubtypedTx
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(arbostypes.ArbosVersion_ArbitrumTippingTx, builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion)
	cleanup := builder.Build(t)
	defer cleanup()
	l2client, l2info := builder.L2.Client, builder.L2Info

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
	info := NewArbTestInfo(t, params.ArbitrumDevTestChainConfig().ChainID)
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
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	// make sure ArbOSVersion supports ArbitrumTippingTx
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = arbmath.MaxInt(arbostypes.ArbosVersion_ArbitrumTippingTx, builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion)
	cleanup := builder.Build(t)
	defer cleanup()
	l2client, l2info := builder.L2.Client, builder.L2Info

	callOpts := l2info.GetDefaultCallOpts("Owner", ctx)

	// get the network fee account
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err, "failed to deploy contract")
	networkFeeAccount, err := arbOwnerPublic.GetNetworkFeeAccount(callOpts)
	Require(t, err, "could not get the network fee account")

	baseFee := GetBaseFee(t, l2client, ctx)
	l2info.GasPrice = baseFee
	l2info.GenerateAccount("User1")
	l2info.GenerateAccount("User2")
	SendWaitTestTransactions(t, ctx, l2client, []*types.Transaction{l2info.PrepareTx("Owner", "User1", l2info.TransferGas, big.NewInt(1e18), nil)})

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
			Fatal(t, "after transfer balances sanity check failed")
		}

		// the network should receive
		//     1. compute costs
		//     2. tip on the compute costs
		//     3. tip on the data costs
		networkAfter := GetBalance(t, ctx, l2client, networkFeeAccount)
		networkRevenue := arbmath.BigSub(networkAfter, networkBefore)
		gasUsedForL2 := receipt.GasUsed - receipt.GasUsedForL1
		feePaidForL2 := arbmath.BigMulByUint(gasPrice, gasUsedForL2)
		tipPaidToNet := arbmath.BigMulByUint(tipCap, receipt.GasUsedForL1)
		gotTip := arbmath.BigEquals(networkRevenue, arbmath.BigAdd(feePaidForL2, tipPaidToNet))
		if !gotTip {
			Fatal(t, "network didn't receive expected payment", networkRevenue, feePaidForL2, tipPaidToNet)
		}
		return networkRevenue, tipPaidToNet
	}

	net0, tip0 := testFees(0)
	net2, tip2 := testFees(2)

	if tip0.Sign() != 0 {
		Fatal(t, "nonzero tip")
	}
	if arbmath.BigEquals(arbmath.BigSub(net2, tip2), net0) {
		Fatal(t, "a tip of 2 should yield a total of 3")
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

func testTippingTxOldArbosVersion(t *testing.T, version uint64) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.chainConfig.ArbitrumChainParams.InitialArbOSVersion = version
	cleanup := builder.Build(t)
	defer cleanup()
	l2client, l2info := builder.L2.Client, builder.L2Info

	baseFee := GetBaseFee(t, l2client, ctx)
	tipCap := arbmath.BigMulByUint(baseFee, 2)
	gasPrice := arbmath.BigAdd(baseFee, tipCap)
	l2info.GenerateAccount("User1")
	tx := l2info.PrepareTippingTx("Owner", "User1", gasPrice.Uint64(), tipCap, big.NewInt(1e12), nil)
	err := l2client.SendTransaction(ctx, tx)
	if err == nil || !strings.Contains(err.Error(), types.ErrTxTypeNotSupported.Error()) {
		testhelpers.FailImpl(t, "tx didn't fail as it should for arbos version:", version, "err:", err)
	}
}

func TestTippingTxOldArbosVersion(t *testing.T) {
	testTippingTxOldArbosVersion(t, 1)
	testTippingTxOldArbosVersion(t, 10)
	testTippingTxOldArbosVersion(t, 11)
}
