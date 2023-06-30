// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package wavmio

func getGlobalStateBytes32(idx uint64, output []byte)
func setGlobalStateBytes32(idx uint64, val []byte)
func getGlobalStateU64(idx uint64) uint64
func setGlobalStateU64(idx uint64, val uint64)
func readInboxMessage(msgNum uint64, offset uint32, output []byte) uint32
func readDelayedInboxMessage(seqNum uint64, offset uint32, output []byte) uint32
func resolvePreImage(hash []byte, offset uint32, output []byte) uint32
