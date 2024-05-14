// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package espressocrypto

import (
	"unsafe"

	"github.com/offchainlabs/nitro/arbutil"
)

//go:wasmimport espressocrypto verifyNamespace
func verify_namespace(
	namespace uint64,
	proof_ptr unsafe.Pointer,
	proof_len uint64,
	block_comm_ptr unsafe.Pointer,
	block_comm_len uint64,
	ns_table_ptr unsafe.Pointer,
	ns_table_len uint64,
	tx_comm_ptr unsafe.Pointer,
	tx_comm_len uint64,
)

//go:wasmimport espressocrypto verifyMerkleProof
func verify_merkle_proof(
	proof_ptr unsafe.Pointer,
	proof_len uint64,
	header_ptr unsafe.Pointer,
	header_len uint64,
	block_comm_ptr unsafe.Pointer,
	block_comm_len uint64,
	circuit_comm_ptr unsafe.Pointer,
	circuit_comm_len uint64,
)

func verifyNamespace(namespace uint64, proof []byte, block_comm []byte, ns_table []byte, tx_comm []byte) {
	verify_namespace(
		namespace,
		arbutil.SliceToUnsafePointer(proof), uint64(len(proof)),
		arbutil.SliceToUnsafePointer(block_comm), uint64(len(block_comm)),
		arbutil.SliceToUnsafePointer(ns_table), uint64(len(ns_table)),
		arbutil.SliceToUnsafePointer(tx_comm), uint64(len(tx_comm)),
	)
}

func verifyMerkleProof(proof []byte, header []byte, block_comm []byte, circuit_comm []byte) {
	verify_merkle_proof(
		arbutil.SliceToUnsafePointer(proof), uint64(len(proof)),
		arbutil.SliceToUnsafePointer(header), uint64(len(header)),
		arbutil.SliceToUnsafePointer(block_comm), uint64(len(block_comm)),
		arbutil.SliceToUnsafePointer(circuit_comm), uint64(len(circuit_comm)),
	)
}
