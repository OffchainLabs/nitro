//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package blsTable

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestLegacyBLS(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(false))
	tab := Open(sto)

	maxInt64 := big.NewInt(math.MaxInt64)

	address := testhelpers.RandomAddress()
	cases := [][]*big.Int{
		{big.NewInt(0), big.NewInt(16), big.NewInt(615), big.NewInt(1024)},
		{big.NewInt(32), big.NewInt(0), big.NewInt(808), big.NewInt(9364)},
		{maxInt64, arbmath.BigMulByFrac(maxInt64, math.MaxInt64, 2), big.NewInt(2), big.NewInt(0)},
	}

	for index, test := range cases {
		Require(t, tab.RegisterLegacyPublicKey(address, test[0], test[1], test[2], test[3]))
		x0, x1, y0, y1, err := tab.GetLegacyPublicKey(address)
		Require(t, err, fmt.Sprintf(
			"failed to set public key %d %s %s %s %s",
			index, x0.String(), x1.String(), y0.String(), y1.String()),
		)

		if x0.Cmp(test[0]) != 0 || x1.Cmp(test[1]) != 0 || y0.Cmp(test[2]) != 0 || y1.Cmp(test[3]) != 0 {
			Fail(t, "incorrect public key", index, test, x0, x1, y0, y1)
		}
	}
}

func Require(t *testing.T, err error, text ...string) {
	t.Helper()
	testhelpers.RequireImpl(t, err, text...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
