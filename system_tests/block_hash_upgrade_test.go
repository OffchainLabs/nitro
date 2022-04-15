// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestUpgradeBlockHash(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainConfig := *params.ArbitrumTestnetChainConfig()
	ownerKey, err := crypto.GenerateKey()
	Require(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(ownerKey, chainConfig.ChainID)
	Require(t, err)
	chainConfig.ArbitrumChainParams.InitialChainOwner = auth.From

	l2info, _, l2client, _, _, _, stack := CreateTestNodeOnL1WithConfig(t, ctx, true, arbnode.ConfigDefaultL1Test(), &chainConfig)
	defer stack.Close()

	l2info.SetFullAccountInfo("RealOwner", &AccountInfo{
		Address:    auth.From,
		PrivateKey: ownerKey,
		Nonce:      0,
	})
	TransferBalance(t, "Faucet", "RealOwner", big.NewInt(5*params.Ether), l2info, l2client, ctx)

	_, _, simple, err := mocksgen.DeploySimple(auth, l2client)
	Require(t, err)

	_, err = simple.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	if err == nil {
		Fail(t, "CheckBlockHashes succeeded pre-upgrade")
	}

	arbOwner, err := precompilesgen.NewArbOwner(common.HexToAddress("0x70"), l2client)
	Require(t, err)
	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(common.HexToAddress("0x6b"), l2client)
	Require(t, err)

	isOwner, err := arbOwnerPublic.IsChainOwner(&bind.CallOpts{Context: ctx}, arbosState.TestnetUpgrade2Owner)
	Require(t, err)
	if isOwner {
		Fail(t, "TestnetUpgrade2Owner is an owner before the upgrade")
	}

	_, err = arbOwner.GetNetworkFeeAccount(&bind.CallOpts{Context: ctx, From: common.HexToAddress("0x1234")})
	if err == nil {
		Fail(t, "GetNetworkFeeAccount succeeded called from non-owner")
	}

	_, err = arbOwner.GetNetworkFeeAccount(&bind.CallOpts{Context: ctx, From: arbosState.TestnetUpgrade2Owner})
	Require(t, err, "GetNetworkFeeAccount failed called from TestnetUpgrade2Owner")

	tx, err := arbOwner.ScheduleArbOSUpgrade(auth, 2, 0)
	Require(t, err)
	_, err = WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	Require(t, err)

	TransferBalance(t, "Faucet", "Faucet", common.Big0, l2info, l2client, ctx)

	_, err = simple.CheckBlockHashes(&bind.CallOpts{Context: ctx})
	Require(t, err)

	isOwner, err = arbOwnerPublic.IsChainOwner(&bind.CallOpts{Context: ctx}, arbosState.TestnetUpgrade2Owner)
	Require(t, err)
	if !isOwner {
		Fail(t, "TestnetUpgrade2Owner isn't an owner after the upgrade")
	}

	tx, err = arbOwner.RemoveChainOwner(auth, arbosState.TestnetUpgrade2Owner)
	Require(t, err)
	_, err = WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
	Require(t, err)

	_, err = arbOwner.GetNetworkFeeAccount(&bind.CallOpts{Context: ctx, From: arbosState.TestnetUpgrade2Owner})
	if err == nil {
		Fail(t, "GetNetworkFeeAccount succeeded called from TestnetUpgrade2Owner after owner removal")
	}
}
