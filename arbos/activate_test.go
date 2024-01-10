// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbos

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestActivationDataFee(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	state, _ := arbosState.NewArbosMemoryBackedArbOSState()
	pricer := state.Programs().DataPricer()
	time := uint64(time.Now().Unix())

	assert := func(cond bool) {
		t.Helper()
		if !cond {
			Fail(t, "assertion failed")
		}
	}

	hour := uint64(60 * 60)
	commonSize := uint32(5 * 1024 * 1024)

	fee, _ := pricer.UpdateModel(0, time)
	assert(fee.Uint64() == 0)

	firstHourlyFee, _ := pricer.UpdateModel(commonSize, time)
	assert(firstHourlyFee.Uint64() > 0)

	capacity := uint32(programs.InitialHourlyBytes)
	usage := uint32(0)
	lastFee := common.Big0
	totalFees := common.Big0
	reset := func() {
		capacity = uint32(programs.InitialHourlyBytes)
		usage = uint32(0)
		lastFee = common.Big0
		totalFees = common.Big0
	}

	reset()
	for usage < capacity {
		bytes := uint32(5 * 1024 * 1024)
		fee, _ := pricer.UpdateModel(bytes, time+hour)
		assert(arbmath.BigGreaterThan(fee, lastFee))

		totalFees = arbmath.BigAdd(totalFees, fee)
		usage += bytes
		lastFee = fee
	}

	// ensure the chain made enough money
	minimumTotal := arbmath.UintToBig(uint64(capacity))
	minimumTotal = arbmath.BigMulByUint(minimumTotal, 140/10*1e9)
	colors.PrintBlue("total ", totalFees.String(), " ", minimumTotal.String())
	assert(arbmath.BigGreaterThan(totalFees, minimumTotal))

	// advance a bit past an hour to reset the pricer
	fee, _ = pricer.UpdateModel(commonSize, time+2*hour+60)
	assert(arbmath.BigEquals(fee, firstHourlyFee))

	// buy all the capacity at once
	fee, _ = pricer.UpdateModel(capacity, time+3*hour)
	colors.PrintBlue("every ", fee.String(), " ", minimumTotal.String())
	assert(arbmath.BigGreaterThan(fee, minimumTotal))

	reset()
	for usage < capacity {
		bytes := uint32(10 * 1024)
		fee, _ := pricer.UpdateModel(bytes, time+5*hour)
		assert(arbmath.BigGreaterThanOrEqual(fee, lastFee))

		totalFees = arbmath.BigAdd(totalFees, fee)
		usage += bytes
		lastFee = fee
	}

	// check small programs
	colors.PrintBlue("small ", totalFees.String(), " ", minimumTotal.String())
	assert(arbmath.BigGreaterThan(totalFees, minimumTotal))

	reset()
	for usage < capacity {
		bytes := testhelpers.RandomUint32(1, 1024*1024)
		fee, _ := pricer.UpdateModel(bytes, time+7*hour)

		totalFees = arbmath.BigAdd(totalFees, fee)
		usage += bytes
		lastFee = fee
	}

	// check random programs
	colors.PrintBlue("random ", totalFees.String(), " ", minimumTotal.String())
	assert(arbmath.BigGreaterThan(totalFees, minimumTotal))
}
