//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

//go:build js
// +build js

package wavmio

func getLastBlockHash(output []byte)
func readInboxMessage(offset uint32, output []byte) uint32
func readDelayedInboxMessage(seqNum uint64, offset uint32, output []byte) uint32
func advanceInboxMessage()
func resolvePreImage(hash []byte, offset uint32, output []byte) uint32
func setLastBlockHash([]byte)
func getPositionWithinMessage() uint64
func setPositionWithinMessage(pos uint64)
func getInboxPosition() uint64
