// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package blockhash

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestBlockhash(t *testing.T) {
	arbosVersion := uint64(8)

	sto := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	InitializeBlockhashes(sto)

	bh := OpenBlockhashes(sto)
	bnum, err := bh.L1BlockNumber()
	Require(t, err, "failed to read blocknum in new Blockhashes")
	if bnum != 0 {
		Fail(t, "incorrect blocknum in new Blockhashes")
	}
	_, err = bh.BlockHash(0)
	if err == nil {
		Fail(t, "should have generated error on Blockhash(0) in new Blockhashes")
	}
	_, err = bh.BlockHash(4242)
	if err == nil {
		Fail(t, "should have generated error on Blockhash(4242) in new Blockhashes")
	}

	hash0 := common.BytesToHash(crypto.Keccak256([]byte{0}))
	err = bh.RecordNewL1Block(0, hash0, arbosVersion)
	Require(t, err)
	bnum, err = bh.L1BlockNumber()
	Require(t, err)
	if bnum != 1 {
		Fail(t, "incorrect NextBlockNumber after initial Blockhash(0)")
	}
	h, err := bh.BlockHash(0)
	Require(t, err)
	if h != hash0 {
		Fail(t, "incorrect hash return for initial Blockhash(0)")
	}

	hash4242 := common.BytesToHash(crypto.Keccak256([]byte{42, 42}))
	err = bh.RecordNewL1Block(4242, hash4242, arbosVersion)
	Require(t, err)
	bnum, err = bh.L1BlockNumber()
	Require(t, err)
	if bnum != 4243 {
		Fail(t, "incorrect NextBlockNumber after big jump")
	}
	_, err = bh.BlockHash(4243)
	if err == nil {
		Fail(t, "BlockHash for future block should generate error")
	}
	h, err = bh.BlockHash(4242)
	Require(t, err)
	if h != hash4242 {
		Fail(t, "incorrect BlockHash(4242)")
	}
	h2, err := bh.BlockHash(4242 - 1)
	Require(t, err)
	if h2 == h {
		Fail(t, "same blockhash at different blocknums")
	}
	h3, err := bh.BlockHash(4242 - 2)
	Require(t, err)
	if h3 == h2 || h3 == h {
		Fail(t, "same blockhash at different blocknums")
	}
	h255, err := bh.BlockHash(4242 - 255)
	Require(t, err)
	if h255 == h || h255 == h3 {
		Fail(t, "same blockhash at different blocknums")
	}
	_, err = bh.BlockHash(4242 - 256)
	if err == nil {
		Fail(t, "old blockhash should give error")
	}

}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
