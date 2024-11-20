package espressocrypto

/*
#cgo LDFLAGS: -L${SRCDIR}/../target/lib/ -lespresso_crypto_helper
#include <stdbool.h>
#include <stdint.h>

bool verify_merkle_proof_helper(
    const uint8_t* proof_ptr, size_t proof_len,
    const uint8_t* header_ptr, size_t header_len,
    const uint8_t* block_comm_ptr, size_t block_comm_len,
    const uint8_t* circuit_block_ptr, size_t circuit_block_len
);
bool verify_namespace_helper(
    uint64_t namespace,
    const uint8_t* proof_ptr, size_t proof_len,
    const uint8_t* commit_ptr, size_t commit_len,
    const uint8_t* ns_table_ptr, size_t ns_table_len,
    const uint8_t* tx_comm_ptr, size_t tx_comm_len,
    const uint8_t* common_data_ptr, size_t common_data_len
);
*/
import "C"
import "unsafe"

func verifyNamespace(namespace uint64, proof []byte, blockComm []byte, nsTable []byte, txComm []byte, commonData []byte) bool {
	c_namespace := C.uint64_t(namespace)

	proofPtr := (*C.uint8_t)(unsafe.Pointer(&proof[0]))
	proofLen := C.size_t(len(proof))

	blockCommPtr := (*C.uint8_t)(unsafe.Pointer(&blockComm[0]))
	blockCommLen := C.size_t(len(blockComm))

	nsTablePtr := (*C.uint8_t)(unsafe.Pointer(&nsTable[0]))
	nsTableLen := C.size_t(len(nsTable))

	txCommPtr := (*C.uint8_t)(unsafe.Pointer(&txComm[0]))
	txCommLen := C.size_t(len(txComm))

	commonDataPtr := (*C.uint8_t)(unsafe.Pointer(&commonData[0]))
	commonDataLen := C.size_t(len(commonData))

	valid_namespace_proof := bool(C.verify_namespace_helper(
		c_namespace, proofPtr, proofLen, blockCommPtr, blockCommLen, nsTablePtr, nsTableLen, txCommPtr, txCommLen, commonDataPtr, commonDataLen))

	return valid_namespace_proof
}

func verifyMerkleProof(proof []byte, header []byte, blockComm []byte, circuitBlock []byte) bool {

	proofPtr := (*C.uint8_t)(unsafe.Pointer(&proof[0]))
	proofLen := C.size_t(len(proof))

	headerPtr := (*C.uint8_t)(unsafe.Pointer(&header[0]))
	headerLen := C.size_t(len(header))

	blockCommPtr := (*C.uint8_t)(unsafe.Pointer(&blockComm[0]))
	blockCommLen := C.size_t(len(blockComm))

	circuitBlockPtr := (*C.uint8_t)(unsafe.Pointer(&circuitBlock[0]))
	circuitBlockLen := C.size_t(len(circuitBlock))

	valid_merkle_proof := bool(C.verify_merkle_proof_helper(proofPtr, proofLen, headerPtr, headerLen, blockCommPtr, blockCommLen, circuitBlockPtr, circuitBlockLen))

	return valid_merkle_proof

}
