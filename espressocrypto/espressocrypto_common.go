// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package espressocrypto

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	espressoTypes "github.com/EspressoSystems/espresso-sequencer-go/types"
)

// TODO move to espresso-go-sequencer: https://github.com/EspressoSystems/nitro-espresso-integration/issues/88
func hashTxns(namespace uint64, txns []espressoTypes.Bytes) string {
	hasher := sha256.New()
	ns_buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(ns_buf, namespace)
	hasher.Write(ns_buf)
	for _, txn := range txns {
		hasher.Write(txn)
	}
	hashResult := hasher.Sum(nil)
	return hex.EncodeToString(hashResult)
}

func VerifyNamespace(namespace uint64, proof espressoTypes.NamespaceProof, block_comm espressoTypes.TaggedBase64, ns_table espressoTypes.NsTable, txs []espressoTypes.Bytes) {
	var txnComm = hashTxns(namespace, txs)
	verifyNamespace(namespace, proof, []byte(block_comm.String()), ns_table.Bytes, []byte(txnComm))
}

func VerifyMerkleProof(proof json.RawMessage, header json.RawMessage, blockComm espressoTypes.TaggedBase64, circuit_comm_bytes espressoTypes.Commitment) {
	verifyMerkleProof(proof, header, []byte(blockComm.String()), circuit_comm_bytes)
}
