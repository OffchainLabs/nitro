//go:build !wasm
// +build !wasm

package espressocrypto

func verifyNamespace(namespace uint64, proof []byte, block_comm []byte, ns_table []byte, tx_comm []byte, common_data []byte) {
}

func verifyMerkleProof(proof []byte, header []byte, block_comm []byte, circuit_comm_bytes []byte) {

}
