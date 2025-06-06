// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm
// +build wasm

package melwavmio

import "unsafe"

//go:wasmimport wavmio resolveTypedPreimage
func resolveTypedPreimage(ty uint32, hash unsafe.Pointer, offset uint32, output unsafe.Pointer) uint32
