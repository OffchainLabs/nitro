// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/l1pricing"
	"github.com/offchainlabs/nitro/cmd/chaininfo"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

// DoesntRevert tests are useful to check if precompile calls revert due to differences in the
// return types of a contract between go and solidity.
// They are not a substitute for unit tests, as they don't test the actual functionality of the precompile.

func TestArbAddressTableDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbAddressTable, err := precompilesgen.NewArbAddressTable(types.ArbAddressTableAddress, builder.L2.Client)
	Require(t, err)

	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	exists, err := arbAddressTable.AddressExists(callOpts, addr)
	Require(t, err)
	if exists {
		Fatal(t, "expected address to not exist")
	}

	tx, err := arbAddressTable.Register(&auth, addr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	idx, err := arbAddressTable.Lookup(callOpts, addr)
	Require(t, err)

	retrievedAddr, err := arbAddressTable.LookupIndex(callOpts, idx)
	Require(t, err)
	if retrievedAddr.Cmp(addr) != 0 {
		Fatal(t, "expected retrieved address to be", addr, "got", retrievedAddr)
	}

	size, err := arbAddressTable.Size(callOpts)
	Require(t, err)
	if size.Cmp(big.NewInt(1)) != 0 {
		Fatal(t, "expected size to be 1, got", size)
	}

	tx, err = arbAddressTable.Compress(&auth, addr)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	res := []uint8{128}
	_, _, err = arbAddressTable.Decompress(callOpts, res, big.NewInt(0))
	Require(t, err)
}

func TestArbAggregatorDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)
	callOpts := &bind.CallOpts{Context: ctx}

	arbAggregator, err := precompilesgen.NewArbAggregator(types.ArbAggregatorAddress, builder.L2.Client)
	Require(t, err)

	tx, err := arbAggregator.SetFeeCollector(&auth, l1pricing.BatchPosterAddress, common.Address{})
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	_, err = arbAggregator.GetFeeCollector(callOpts, l1pricing.BatchPosterAddress)
	Require(t, err)
}

func TestArbosTestDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbosTest, err := precompilesgen.NewArbosTest(types.ArbosTestAddress, builder.L2.Client)
	Require(t, err)

	err = arbosTest.BurnArbGas(callOpts, big.NewInt(1))
	Require(t, err)
}

func TestArbSysDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, builder.L2.Client)
	Require(t, err)

	addr1 := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	addr2 := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])
	_, err = arbSys.MapL1SenderContractAddressToL2Alias(callOpts, addr1, addr2)
	Require(t, err)
}

func TestArbOwnerDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	chainConfig := chaininfo.ArbitrumDevTestChainConfig()
	chainConfig.ArbitrumChainParams.MaxCodeSize = 100
	serializedChainConfig, err := json.Marshal(chainConfig)
	Require(t, err)
	tx, err := arbOwner.SetChainConfig(&auth, string(serializedChainConfig))
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	tx, err = arbOwner.SetAmortizedCostCapBips(&auth, 77734)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	tx, err = arbOwner.ReleaseL1PricerSurplusFunds(&auth, big.NewInt(1))
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	tx, err = arbOwner.SetL2BaseFee(&auth, big.NewInt(1))
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}

func TestArbGasInfoDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}
	addr := common.BytesToAddress(crypto.Keccak256([]byte{})[:20])

	arbGasInfo, err := precompilesgen.NewArbGasInfo(types.ArbGasInfoAddress, builder.L2.Client)
	Require(t, err)

	_, err = arbGasInfo.GetGasBacklog(callOpts)
	Require(t, err)

	_, err = arbGasInfo.GetLastL1PricingUpdateTime(callOpts)
	Require(t, err)

	_, err = arbGasInfo.GetL1PricingFundsDueForRewards(callOpts)
	Require(t, err)

	_, err = arbGasInfo.GetL1PricingUnitsSinceUpdate(callOpts)
	Require(t, err)

	_, err = arbGasInfo.GetLastL1PricingSurplus(callOpts)
	Require(t, err)

	_, _, _, err = arbGasInfo.GetPricesInArbGas(callOpts)
	Require(t, err)

	_, _, _, err = arbGasInfo.GetPricesInArbGasWithAggregator(callOpts, addr)
	Require(t, err)

	_, err = arbGasInfo.GetAmortizedCostCapBips(callOpts)
	Require(t, err)

	_, err = arbGasInfo.GetL1FeesAvailable(callOpts)
	Require(t, err)

	_, _, _, _, _, _, err = arbGasInfo.GetPricesInWeiWithAggregator(callOpts, addr)
	Require(t, err)
}

func TestArbRetryableTxDoesntRevert(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	callOpts := &bind.CallOpts{Context: ctx}

	arbRetryableTx, err := precompilesgen.NewArbRetryableTx(common.HexToAddress("6e"), builder.L2.Client)
	Require(t, err)

	_, err = arbRetryableTx.GetCurrentRedeemer(callOpts)
	Require(t, err)
}
