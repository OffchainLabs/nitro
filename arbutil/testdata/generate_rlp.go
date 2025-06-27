package main

import (
	"log"
	"os"

	"github.com/ethereum/go-ethereum/rlp"
)

// MessageIndex is an alias for backward compatibility testing.
type MessageIndex uint64

// OldSubmittedEspressoTx represents the structure of the transaction before
// the SubmittedAt field was added.
type OldSubmittedEspressoTx struct {
	Hash    string
	Pos     []MessageIndex
	Payload []byte
}

func main() {
	oldSubmittedTx := OldSubmittedEspressoTx{
		Hash:    "0x1234",
		Pos:     []MessageIndex{MessageIndex(10)},
		Payload: []byte{0, 1, 2, 3},
	}

	b, err := rlp.EncodeToBytes(&oldSubmittedTx)
	if err != nil {
		log.Fatalf("Failed to encode: %v", err)
	}

	err = os.WriteFile("arbutil/testdata/old_submitted_espresso_tx.rlp", b, 0644)
	if err != nil {
		log.Fatalf("Failed to write artifact file: %v", err)
	}
}
