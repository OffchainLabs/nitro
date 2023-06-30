// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package util

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestRetryableEncoding(t *testing.T) {
	rand.Seed(time.Now().UnixMilli())
	fakeAddr := testhelpers.RandomAddress()
	key, err := crypto.GenerateKey()
	testhelpers.RequireImpl(t, err)
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	testhelpers.RequireImpl(t, err)

	alloc := make(core.GenesisAlloc)
	alloc[fakeAddr] = core.GenesisAccount{
		Code:    []byte{0},
		Balance: big.NewInt(0),
	}
	alloc[auth.From] = core.GenesisAccount{
		Balance: big.NewInt(1000000000000000000),
	}
	client := backends.NewSimulatedBackend(alloc, 1000000)

	dest := testhelpers.RandomAddress()
	innerTx := &types.ArbitrumSubmitRetryableTx{
		ChainId:          big.NewInt(654645),
		RequestId:        common.BigToHash(big.NewInt(rand.Int63n(1 << 32))),
		From:             testhelpers.RandomAddress(),
		L1BaseFee:        big.NewInt(876876),
		DepositValue:     big.NewInt(145331),
		GasFeeCap:        big.NewInt(76456),
		Gas:              37655,
		RetryTo:          &dest,
		RetryValue:       big.NewInt(23454),
		Beneficiary:      testhelpers.RandomAddress(),
		MaxSubmissionFee: big.NewInt(567356),
		FeeRefundAddr:    testhelpers.RandomAddress(),
		RetryData:        testhelpers.RandomizeSlice(make([]byte, rand.Int()%512)),
	}

	con, err := precompilesgen.NewArbRetryableTx(fakeAddr, client)
	testhelpers.RequireImpl(t, err)

	var retryTo common.Address
	if innerTx.RetryTo != nil {
		retryTo = *innerTx.RetryTo
	}
	tx, err := con.SubmitRetryable(
		auth,
		innerTx.RequestId,
		innerTx.L1BaseFee,
		innerTx.DepositValue,
		innerTx.RetryValue,
		innerTx.GasFeeCap,
		innerTx.Gas,
		innerTx.MaxSubmissionFee,
		innerTx.FeeRefundAddr,
		innerTx.Beneficiary,
		retryTo,
		innerTx.RetryData,
	)
	testhelpers.RequireImpl(t, err)

	if !bytes.Equal(tx.Data(), types.NewTx(innerTx).Data()) {
		testhelpers.FailImpl(t, "incorrect data encoding")
	}
}
