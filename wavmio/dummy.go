// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//
//go:build !sp1

package wavmio

import (
	"unsafe"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
)

func ResolveTypedPreimage(ty arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	return readBuffer(func(offset uint32, buf unsafe.Pointer) uint32 {
		hashUnsafe := unsafe.Pointer(&hash[0])
		return resolveTypedPreimage(uint32(ty), hashUnsafe, offset, buf)
	}), nil
}
