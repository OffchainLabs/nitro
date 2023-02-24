// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	templates "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestRetryableRedeem(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := evm.Context.Time.Uint64() + 10000000
	from := common.HexToAddress("0x030405")
	to := common.HexToAddress("0x06070809")
	callvalue := big.NewInt(0)
	beneficiary := common.HexToAddress("0x0301040105090206")
	calldata := make([]byte, 42)
	for i := range calldata {
		calldata[i] = byte(i + 3)
	}
	_, err := precompileCtx.State.RetryableState().CreateRetryable(
		id,
		timeout,
		from,
		&to,
		callvalue,
		beneficiary,
		calldata,
	)
	Require(t, err)

	retryABI, err := templates.ArbRetryableTxMetaData.GetAbi()
	Require(t, err)
	redeemCalldata, err := retryABI.Pack("redeem", id)
	Require(t, err)

	retryAddress := types.ArbRetryableTxAddress
	_, gasLeft, err := Precompiles()[retryAddress].Call(
		redeemCalldata,
		retryAddress,
		retryAddress,
		common.Address{},
		big.NewInt(0),
		false,
		1000000,
		evm,
	)
	Require(t, err)

	if gasLeft != 0 {
		Fail(t, "didn't consume all the expected gas", gasLeft)
	}
}
