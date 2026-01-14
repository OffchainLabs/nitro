// Copyright 2026-2027, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm
// +build wasm

package melwavmio

import "unsafe"

//go:wasmimport wavmio resolveTypedPreimage
func resolveTypedPreimage(ty uint32, hash unsafe.Pointer, offset uint32, output unsafe.Pointer) uint32

//go:wasmimport wavmio getGlobalStateBytes32
func getGlobalStateBytes32(idx uint32, output unsafe.Pointer)

//go:wasmimport wavmio getGlobalStateUint64
func getGlobalStateUint64(idx uint32) uint64

//go:wasmimport wavmio setGlobalStateU64
func setGlobalStateU64(idx uint32, val uint64)

//go:wasmimport wavmio setGlobalStateBytes32
func setGlobalStateBytes32(idx uint32, val unsafe.Pointer)

//go:wasmimport wavmio getEndParentChainBlockHash
func getEndParentChainBlockHash(unsafe.Pointer)
