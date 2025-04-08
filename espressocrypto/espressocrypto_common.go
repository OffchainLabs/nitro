// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package espressocrypto

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	espressoTypes "github.com/EspressoSystems/espresso-network-go/types"
)

// TODO move to espresso-go-sequencer: https://github.com/EspressoSystems/nitro-espresso-integration/issues/88
func hashTxns(namespace uint32, txns []espressoTypes.Bytes) string {
	hasher := sha256.New()
	ns_buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(ns_buf, namespace)
	hasher.Write(ns_buf)
	for _, txn := range txns {
		hasher.Write(txn)
	}
	hashResult := hasher.Sum(nil)
	return hex.EncodeToString(hashResult)
}

func VerifyNamespace(
	namespace uint64,
	proof espressoTypes.NamespaceProof,
	block_comm espressoTypes.TaggedBase64,
	ns_table espressoTypes.NsTable,
	txs []espressoTypes.Bytes,
	common_data json.RawMessage,
) bool {
	// TODO: this code will likely no longer be used in the STF soon.
	// G115: integer overflow conversion uint64 -> uint32 (gosec)
	// #nosec G115
	var txnComm = hashTxns(uint32(namespace), txs)
	res := verifyNamespace(
		namespace,
		proof,
		[]byte(block_comm.String()),
		ns_table.Bytes,
		[]byte(txnComm),
		common_data,
	)
	return res
}

func VerifyMerkleProof(
	proof json.RawMessage,
	header json.RawMessage,
	blockComm espressoTypes.TaggedBase64,
	circuit_comm_bytes espressoTypes.Commitment,
) bool {
	return verifyMerkleProof(proof, header, []byte(blockComm.String()), circuit_comm_bytes[:])
}
