// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

package espressocrypto

func verifyNamespace(namespace uint64, proof []byte, block_comm []byte, ns_table []byte, tx_comm []byte)

func verifyMerkleProof(merkle_root [32]byte, proof []byte, block_comm [32]byte)
