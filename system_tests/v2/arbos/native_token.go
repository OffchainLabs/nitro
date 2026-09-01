// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbos

import (
	"math/big"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/precompiles"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

var nativeTokenTests = []systest.Scenario{
	systest.Test(testRunArbNativeTokenManager,
		systest.WithArbOS(params.ArbosVersion_50),
		systest.WithArbOSInit(params.ArbOSInit{NativeTokenSupplyManagementEnabled: true})),
	systest.Test(testRunNativeTokenManagementDisabledByDefault,
		systest.WithArbOS(params.ArbosVersion_50)),
}

func testRunArbNativeTokenManager(env *systest.Env) {
	authOwner := env.L2.TransactOpts("Owner")
	ownerAddr := env.L2.Info.GetAddress("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}

	// first tests native token owner management

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)
	arbOwnerPub, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, env.L2.Client)
	env.Require(err)

	nativeTokenOwnerName := "NativeTokenOwner"
	env.L2.Info.GenerateAccount(nativeTokenOwnerName)
	nativeTokenOwnerAddr := env.L2.Info.GetAddress(nativeTokenOwnerName)

	// checks that no native token owners are set
	isNativeTokenOwner, err := arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	env.Require(err)
	env.True(!isNativeTokenOwner, "expected native token owner to not be set")
	nativeTokenOwners, err := arbOwner.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	env.Len(nativeTokenOwners, 0, "expected no native token owners")
	// same checks to exercise the public interface
	isNativeTokenOwner, err = arbOwnerPub.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	env.Require(err)
	env.True(!isNativeTokenOwner, "expected native token owner to not be set")
	nativeTokenOwners, err = arbOwnerPub.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	env.Len(nativeTokenOwners, 0, "expected no native token owners")

	// adds native token owners
	tx, err := arbOwner.AddNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	tx, err = arbOwner.AddNativeTokenOwner(&authOwner, ownerAddr)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// checks that the native token owners are set
	isNativeTokenOwner, err = arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	env.Require(err)
	env.True(isNativeTokenOwner, "expected native token owner to be set")
	expectedNativeTokenOwners := []common.Address{nativeTokenOwnerAddr, ownerAddr}
	addrSorter := func(a, b common.Address) int {
		return a.Cmp(b)
	}
	slices.SortFunc(expectedNativeTokenOwners, addrSorter)
	nativeTokenOwners, err = arbOwner.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	slices.SortFunc(nativeTokenOwners, addrSorter)
	env.Equal(expectedNativeTokenOwners, nativeTokenOwners, "native token owners differ")
	// same checks to exercise the public interface
	isNativeTokenOwner, err = arbOwnerPub.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	env.Require(err)
	env.True(isNativeTokenOwner, "expected native token owner to be set")
	nativeTokenOwners, err = arbOwnerPub.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	slices.SortFunc(nativeTokenOwners, addrSorter)
	env.Equal(expectedNativeTokenOwners, nativeTokenOwners, "native token owners differ")

	// removes native token owner
	tx, err = arbOwner.RemoveNativeTokenOwner(&authOwner, ownerAddr)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	isNativeTokenOwner, err = arbOwner.IsNativeTokenOwner(callOpts, ownerAddr)
	env.Require(err)
	env.True(!isNativeTokenOwner, "expected native token owner to not be set")
	enabledTime, err := arbOwnerPub.GetNativeTokenManagementFrom(callOpts)
	env.Require(err)
	env.Equal(uint64(1), enabledTime, "unexpected enabledTime")
	nativeTokenOwners, err = arbOwner.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	env.Len(nativeTokenOwners, 1, "expected one native token owner")
	env.Equal(nativeTokenOwnerAddr, nativeTokenOwners[0], "unexpected native token owner")

	// tests minting and burning native tokens

	nativeTokenOwnerABI, err := precompilesgen.ArbNativeTokenManagerMetaData.GetAbi()
	env.Require(err)
	mintTopic := nativeTokenOwnerABI.Events["NativeTokenMinted"].ID
	burnTopic := nativeTokenOwnerABI.Events["NativeTokenBurned"].ID

	arbNativeTokenManager, err := precompilesgen.NewArbNativeTokenManager(types.ArbNativeTokenManagerAddress, env.L2.Client)
	env.Require(err)

	// tries to mint and burn without being a native token owner
	_, err = arbNativeTokenManager.MintNativeToken(&authOwner, big.NewInt(100))
	env.ErrorContains(err, "execution reverted", "expected minting to fail")
	_, err = arbNativeTokenManager.BurnNativeToken(&authOwner, big.NewInt(100))
	env.ErrorContains(err, "execution reverted", "expected burning to fail")

	// funds the native token owner
	tx = env.L2.Info.PrepareTx("Owner", nativeTokenOwnerName, env.L2.Info.TransferGas, big.NewInt(500000000000000000), nil)
	env.L2.SendTx(tx)
	env.L2.EnsureTxSucceeded(tx)

	authNativeTokenOwner := env.L2.TransactOpts(nativeTokenOwnerName)
	authNativeTokenOwner.GasLimit = 32000000

	getGasUsed := func(receipt *types.Receipt) *big.Int {
		gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
		gasUsed.Mul(gasUsed, receipt.EffectiveGasPrice)
		return gasUsed
	}

	// checks minting
	toMint := big.NewInt(100)
	balanceBeforeMinting := env.L2.BalanceAt(nativeTokenOwnerAddr)
	tx, err = arbNativeTokenManager.MintNativeToken(&authNativeTokenOwner, toMint)
	env.Require(err)
	receipt := env.L2.EnsureTxSucceeded(tx)
	mintLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == mintTopic {
			mintLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenMinted(*log)
			env.Require(err)
			env.Equal(nativeTokenOwnerAddr, parsedLog.To, "unexpected mint recipient")
			env.EqualBig(toMint, parsedLog.Amount, "unexpected mint amount")
		}
	}
	env.True(mintLogged, "expected mint event to be logged")
	balanceAfterMinting := env.L2.BalanceAt(nativeTokenOwnerAddr)
	gasUsed := getGasUsed(receipt)
	expectedBalance := new(big.Int).Sub(balanceBeforeMinting, gasUsed)
	expectedBalance = expectedBalance.Add(expectedBalance, toMint)
	env.EqualBig(expectedBalance, balanceAfterMinting, "unexpected balance after minting")

	// checks burning
	toBurn := big.NewInt(50)
	tx, err = arbNativeTokenManager.BurnNativeToken(&authNativeTokenOwner, toBurn)
	env.Require(err)
	receipt = env.L2.EnsureTxSucceeded(tx)
	burnLogged := false
	for _, log := range receipt.Logs {
		if log.Topics[0] == burnTopic {
			burnLogged = true
			parsedLog, err := arbNativeTokenManager.ParseNativeTokenBurned(*log)
			env.Require(err)
			env.Equal(nativeTokenOwnerAddr, parsedLog.From, "unexpected burn sender")
			env.EqualBig(toBurn, parsedLog.Amount, "unexpected burn amount")
		}
	}
	env.True(burnLogged, "expected burn event to be logged")
	balanceAfterBurning := env.L2.BalanceAt(nativeTokenOwnerAddr)
	gasUsed = getGasUsed(receipt)
	expectedBalance = new(big.Int).Sub(balanceAfterMinting, gasUsed)
	expectedBalance = expectedBalance.Sub(expectedBalance, toBurn)
	env.EqualBig(expectedBalance, balanceAfterBurning, "unexpected balance after burning")

	// checks sending L2 to L1 value is disabled while native token owners exist
	arbSys, err := precompilesgen.NewArbSys(types.ArbSysAddress, env.L2.Client)
	env.Require(err)
	authNativeTokenOwner.Value = big.NewInt(100)
	_, err = arbSys.SendTxToL1(&authNativeTokenOwner, common.Address{}, []byte{})
	env.ErrorContains(err, "execution reverted", "expected sending L2 to L1 value to fail")

	// After clearing the native token owners, sending L2 to L1 value should
	// work again.
	tx, err = arbOwner.RemoveNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)
	authNativeTokenOwner.Value = big.NewInt(100)
	_, err = arbSys.SendTxToL1(&authNativeTokenOwner, common.Address{}, []byte{})
	env.Require(err, "expected sending L2 to L1 value to succeed")
}

func testRunNativeTokenManagementDisabledByDefault(env *systest.Env) {
	authOwner := env.L2.TransactOpts("Owner")
	callOpts := &bind.CallOpts{Context: env.Ctx}

	// tests that native token owner management is disabled by default
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, env.L2.Client)
	env.Require(err)

	nativeTokenOwnerName := "NativeTokenOwner"
	env.L2.Info.GenerateAccount(nativeTokenOwnerName)
	nativeTokenOwnerAddr := env.L2.Info.GetAddress(nativeTokenOwnerName)

	// checks that no native token owners are set
	isNativeTokenOwner, err := arbOwner.IsNativeTokenOwner(callOpts, nativeTokenOwnerAddr)
	env.Require(err)
	env.True(!isNativeTokenOwner, "expected native token owner to not be set")
	nativeTokenOwners, err := arbOwner.GetAllNativeTokenOwners(callOpts)
	env.Require(err)
	env.Len(nativeTokenOwners, 0, "expected no native token owners")

	// attempts to add native token owners before the feature is enabled
	_, err = arbOwner.AddNativeTokenOwner(&authOwner, nativeTokenOwnerAddr)
	env.ErrorContains(err, "execution reverted", "expected adding native token owner to fail")

	now := time.Now()
	// #nosec G115
	sixDaysFromNow := uint64(now.Add(24 * 6 * time.Hour).Unix())
	// #nosec G115
	sevenAndAHalfDaysFromNow := uint64(now.Add(24*7*time.Hour + 12*time.Hour).Unix())
	// #nosec G115
	eightDaysFromNow := uint64(now.Add(24 * 8 * time.Hour).Unix())

	// attempts to enable the feature too early (6 days from now, instead of 7)
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sixDaysFromNow)
	env.ErrorContains(err, "execution reverted", "expected enabling native token management to fail")

	// succeeds to enable the feature enough in the future (8 days from now)
	tx, err := arbOwner.SetNativeTokenManagementFrom(&authOwner, eightDaysFromNow)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// succeeds to shorten the time to enable the feature as long as it is still
	// far enough in the future (7.5 days from now)
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sevenAndAHalfDaysFromNow)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// fails to shorten the time to enable the feature if it is too close to
	// the current time (6 days from now)
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, sixDaysFromNow)
	env.ErrorContains(err, "execution reverted", "expected enabling native token management to fail")

	// Test the shortening boundary: once the stored value is within the
	// [now, now+FeatureEnableDelay] window, the new timestamp must be >= stored.
	//
	// Set enable-at to blockTime + delay + 2, then sleep 3s so the stored value
	// falls into the shortening window (stored <= new_blockTime + delay).
	hdr := env.L2.HeaderByNumber(nil)
	// #nosec G115
	barelyOverDelay := hdr.Time + uint64(precompiles.FeatureEnableDelay) + 2
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, barelyOverDelay)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// Wait for block time to advance past the 2s margin
	time.Sleep(3 * time.Second)

	// Setting a new value later than stored should succeed
	shortenedForward := barelyOverDelay + 1
	tx, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, shortenedForward)
	env.Require(err)
	env.L2.EnsureTxSucceeded(tx)

	// Going backwards (earlier than stored) should fail
	_, err = arbOwner.SetNativeTokenManagementFrom(&authOwner, barelyOverDelay)
	env.ErrorContains(err, "execution reverted", "expected enabling native token management to fail")
}
