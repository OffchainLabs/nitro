// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package precompiles

import (
	"math/big"
	"testing"

	"github.com/offchainlabs/nitro/arbos/storage"

	"github.com/ethereum/go-ethereum/common"
	templates "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
)

func TestRetryableRedeem(t *testing.T) {
	evm := newMockEVMForTesting()
	precompileCtx := testContext(common.Address{}, evm)

	id := common.BigToHash(big.NewInt(978645611142))
	timeout := evm.Context.Time + 10000000
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

	retryAddress := common.HexToAddress("6e")
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

	if gasLeft != storage.StorageWriteCost-storage.StorageWriteZeroCost {
		// We expect to have some gas left over, because in this test we write a zero, but in other
		//     use cases the precompile would cause a non-zero write. So the precompile allocates enough gas
		//     to handle both cases, and some will be left over in this test's use case.
		Fail(t, "didn't consume all the expected gas")
	}
}
