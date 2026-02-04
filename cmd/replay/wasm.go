// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build wasm

package main

import "runtime/debug"

func setupGarbageCollector() {
	// Decrease the GC activity to run once the freshly allocated memory
	// reaches 10x the previous live memory (since the last collection).
	debug.SetGCPercent(1000)
}
