//go:build !js
// +build !js

package espressocrypto

func verifyNamespace(namespace uint64, proof []byte, block_comm []byte, ns_table []byte, tx_comm []byte) {
}

func verifyMerkleProof(proof []byte, header []byte, circuit_comm_bytes [32]byte) {

}
