// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package l1pricing

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
)

func TestBatchPosterTable(t *testing.T) {
	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := InitializeBatchPostersTable(sto)
	Require(t, err)

	bpTable := OpenBatchPostersTable(sto)

	addr1 := common.Address{1, 2, 3}
	pay1 := common.Address{4, 5, 6, 7}
	addr2 := common.Address{2, 4, 6}
	pay2 := common.Address{8, 10, 12, 14}

	// test creation and counting of bps
	allPosters, err := bpTable.AllPosters(math.MaxUint64)
	Require(t, err)
	if len(allPosters) != 0 {
		t.Fatal()
	}
	exists, err := bpTable.ContainsPoster(addr1)
	Require(t, err)
	if exists {
		t.Fatal()
	}

	bp1, err := bpTable.AddPoster(addr1, pay1)
	Require(t, err)
	getPay1, err := bp1.PayTo()
	Require(t, err)
	if getPay1 != pay1 {
		t.Fatal()
	}
	getDue1, err := bp1.FundsDue()
	Require(t, err)
	if getDue1.Sign() != 0 {
		t.Fatal()
	}
	exists, err = bpTable.ContainsPoster(addr1)
	Require(t, err)
	if !exists {
		t.Fatal()
	}

	bp2, err := bpTable.AddPoster(addr2, pay2)
	Require(t, err)
	_ = bp2
	getPay2, err := bp2.PayTo()
	Require(t, err)
	if getPay2 != pay2 {
		t.Fatal()
	}
	getDue2, err := bp2.FundsDue()
	Require(t, err)
	if getDue2.Sign() != 0 {
		t.Fatal()
	}
	exists, err = bpTable.ContainsPoster(addr2)
	Require(t, err)
	if !exists {
		t.Fatal()
	}

	allPosters, err = bpTable.AllPosters(math.MaxUint64)
	Require(t, err)
	if len(allPosters) != 2 {
		t.Fatal()
	}

	// test get/set of BP fields
	bp1, err = bpTable.OpenPoster(addr1, false)
	Require(t, err)
	err = bp1.SetPayTo(addr2)
	Require(t, err)
	getPay1, err = bp1.PayTo()
	Require(t, err)
	if getPay1 != addr2 {
		t.Fatal()
	}
	err = bp1.SetFundsDue(big.NewInt(13))
	Require(t, err)
	getDue1, err = bp1.FundsDue()
	Require(t, err)
	if getDue1.Uint64() != 13 {
		t.Fatal()
	}

	// test adding up the fundsDue
	err = bp2.SetFundsDue(big.NewInt(42))
	Require(t, err)
	getDue2, err = bp2.FundsDue()
	Require(t, err)
	if getDue2.Uint64() != 42 {
		t.Fatal()
	}

	totalDue, err := bpTable.TotalFundsDue()
	Require(t, err)
	if totalDue.Uint64() != 13+42 {
		t.Fatal()
	}
}
