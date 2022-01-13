//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package statetransfer

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbos/util"
	"github.com/offchainlabs/arbstate/solgen/go/classicgen"
	"math/big"
	"os"
	"path/filepath"
)

func openClassicArbAddressTable(client *ethclient.Client) (*classicgen.ArbAddressTableCaller, error) {
	return classicgen.NewArbAddressTableCaller(common.BigToAddress(big.NewInt(ArbAddressTableAsInt)), client)
}

func getAddressTableContents(caller *classicgen.ArbAddressTableCaller, callopts *bind.CallOpts, cachePath *string) ([]common.Address, error) {
	size, err := caller.Size(callopts)
	if err != nil {
		return nil, err
	}

	cache, err := openAddressTableCache(cachePath)
	if err != nil {
		return nil, err
	}
	defer cache.flush()

	if err := cache.checkReorgAndFix(caller, callopts); err != nil {
		// BUGBUG disabling the error return until Classic ArbOS is upgraded
		// return nil, err
		_ = err
	}

	for i := int64(cache.size()); i < size.Int64(); i++ {
		if (i % 100) == 0 {
			fmt.Println(i, " / ", size.Int64(), " addresses")
		}
		addr, err := caller.LookupIndex(callopts, big.NewInt(i))
		if err != nil {
			return nil, err
		}
		cache.append(addr)
	}
	return cache.contents(), nil
}

type addressTableCache struct {
	filepath    *string
	addrs       []common.Address
	initialSize uint64
	hashChains  map[uint64]common.Hash
}

func openAddressTableCache(cachePath *string) (*addressTableCache, error) {
	if cachePath == nil {
		return &addressTableCache{nil, []common.Address{}, 0, make(map[uint64]common.Hash)}, nil
	}
	fullPath := filepath.Join(*cachePath, "addressTableData")

	_, err := os.Stat(fullPath)
	if err != nil && os.IsNotExist(err) {
		// cache file doesn't exist; set things up so it will be created when cache is flushed
		return &addressTableCache{&fullPath, []common.Address{}, 0, make(map[uint64]common.Hash)}, nil
	}

	contents, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	size := len(contents) / 20
	addrs := make([]common.Address, size)
	for i := range addrs {
		addrs[i] = common.BytesToAddress(contents[20*i : 20*(i+1)])
	}
	return &addressTableCache{&fullPath, addrs, uint64(size), make(map[uint64]common.Hash)}, nil
}

func (cache *addressTableCache) checkReorgAndFix(caller *classicgen.ArbAddressTableCaller, callopts *bind.CallOpts) error {
	if _, exists := cache.hashChains[0]; !exists {
		cache.hashChains[0] = common.Hash{}
	}

	base := cache.hashChains[0]
	lo := uint64(0)
	hi := cache.size()
	for lo+1 < hi {
		mid := (lo + hi) / 2
		expected := cache.expectedHashChain(base, lo, hi)
		classic, err := caller.HashRange(callopts, util.UintToHash(lo).Big(), util.UintToHash(hi).Big(), base)
		if err != nil {
			return err
		}
		if expected == classic {
			lo = mid
			base = classic
		} else {
			hi = mid
		}
	}

	cache.addrs = cache.addrs[:hi]
	if cache.initialSize > uint64(len(cache.addrs)) {
		cache.initialSize = uint64(len(cache.addrs))
	}
	return nil
}

func (cache *addressTableCache) expectedHashChain(base common.Hash, lo uint64, hi uint64) common.Hash {
	accum := base.Bytes()
	for lo < hi {
		accum = crypto.Keccak256(accum, cache.addrs[lo].Bytes())
		lo++
	}
	return common.BytesToHash(accum)
}

func (cache *addressTableCache) append(addr common.Address) {
	cache.addrs = append(cache.addrs, addr)
}

func (cache *addressTableCache) size() uint64 {
	return uint64(len(cache.addrs))
}

func (cache *addressTableCache) contents() []common.Address {
	return cache.addrs
}

func (cache *addressTableCache) flush() { // try to flush updates to cache file, ignoring errors
	if cache.filepath == nil {
		return
	}
	if uint64(len(cache.addrs)) == cache.initialSize {
		return
	}

	f, err := os.OpenFile(*cache.filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	dataToWrite := []byte{}
	nextToWrite := cache.initialSize + 1
	for nextToWrite < cache.size() {
		dataToWrite = append(dataToWrite, cache.addrs[nextToWrite].Bytes()...)
		nextToWrite++
	}

	_, _ = f.Write(dataToWrite) // ignore errors
}
