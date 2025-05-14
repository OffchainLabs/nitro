package chess

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGame(t *testing.T) {
	white := common.HexToAddress("0xfafa")
	black := common.HexToAddress("0xbebe")
	game := NewGame(white, black)

	err := game.MakeMove(black, "")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	t.Log("got expected err:", err)

	err = game.MakeMove(white, "d2d5")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	t.Log("got expected err:", err)

	err = game.MakeMove(white, "d2d4")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	t.Log(game.Draw())
}
