package chess

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestChessNode(t *testing.T) {
	node := NewChessNode()

	white := common.HexToAddress("0xfafa")
	black := common.HexToAddress("0xbebe")

	// CreateGame
	tx := []byte{0}
	tx = append(tx, white.Bytes()...)
	tx = append(tx, black.Bytes()...)
	err := node.Process(white, tx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// MakeMove
	tx = []byte{1}
	tx = binary.BigEndian.AppendUint64(tx, 0)
	tx = append(tx, []byte("d2d4")...)
	err = node.Process(white, tx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	// MakeMove
	tx = []byte{1}
	tx = binary.BigEndian.AppendUint64(tx, 0)
	tx = append(tx, []byte("d7d5")...)
	err = node.Process(black, tx)
	if err != nil {
		t.Fatalf("failed to process: %v", err)
	}

	t.Log(node.Status())
}
