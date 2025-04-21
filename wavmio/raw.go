// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm
// +build wasm

package wavmio

import "unsafe"

//go:wasmimport wavmio getGlobalStateBytes32
func getGlobalStateBytes32(idx uint32, output unsafe.Pointer)

//go:wasmimport wavmio setGlobalStateBytes32
func setGlobalStateBytes32(idx uint32, val unsafe.Pointer)

//go:wasmimport wavmio getGlobalStateU64
func getGlobalStateU64(idx uint32) uint64

//go:wasmimport wavmio setGlobalStateU64
func setGlobalStateU64(idx uint32, val uint64)

//go:wasmimport wavmio readInboxMessage
func readInboxMessage(msgNum uint64, offset uint32, output unsafe.Pointer) uint32

//go:wasmimport wavmio readDelayedInboxMessage
func readDelayedInboxMessage(seqNum uint64, offset uint32, output unsafe.Pointer) uint32

//go:wasmimport wavmio resolveTypedPreimage
func resolveTypedPreimage(ty uint32, hash unsafe.Pointer, offset uint32, output unsafe.Pointer) uint32
