package espressocrypto

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"testing"
)

type merkleProofTestData struct {
	Proof             json.RawMessage `json:"proof"`
	Header            json.RawMessage `json:"header"`
	BlockMerkleRoot   string          `json:"block_merkle_root"`
	HotShotCommitment []uint8         `json:"hotshot_commitment"`
}

func TestMerkleProofVerification(t *testing.T) {
	file, err := os.Open("./merkle_proof_test_data.json")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file")
	}

	var data merkleProofTestData

	if err := json.Unmarshal(bytes, &data); err != nil {
		log.Fatalf("Failed to unmarshal the test data")
	}

	r := verifyMerkleProof(data.Proof, data.Header, []byte(data.BlockMerkleRoot), data.HotShotCommitment)
	if !r {
		log.Fatalf("Failed to verify the merkle proof")
	}

	// Tamper with the correct data and see if it will return false
	data.HotShotCommitment[0] = 1

	r = verifyMerkleProof(data.Proof, data.Header, []byte(data.BlockMerkleRoot), data.HotShotCommitment)
	if r {
		log.Fatalf("Failed to verify the merkle proof")
	}

}

type namespaceProofTestData struct {
	NsProof   json.RawMessage `json:"ns_proof"`
	VidCommit string          `json:"vid_commit"`
	VidCommon json.RawMessage `json:"vid_common"`
	Namespace uint64          `json:"namespace"`
	NsTable   []uint8         `json:"ns_table"`
	TxCommit  string          `json:"tx_commit"`
}

func TestNamespaceProofVerification(t *testing.T) {
	file, err := os.Open("./namespace_proof_test_data.json")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file")
	}

	var data namespaceProofTestData

	if err := json.Unmarshal(bytes, &data); err != nil {
		log.Fatalf("Failed to unmarshal the test data: %v", err)
	}

	r := verifyNamespace(data.Namespace, data.NsProof, []byte(data.VidCommit), data.NsTable, []byte(data.TxCommit), data.VidCommon)
	if !r {
		log.Fatalf("Failed to verify the namespace proof")
	}

	// Tamper with the correct data and see if it will return false
	data.Namespace = 1

	r = verifyNamespace(data.Namespace, data.NsProof, []byte(data.VidCommit), data.NsTable, []byte(data.TxCommit), data.VidCommon)
	if r {
		log.Fatalf("Failed to verify the namespace proof")
	}
}
