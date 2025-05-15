package chess

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/google/go-cmp/cmp"
)

func TestChessAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create tcp port: %v", err)
	}
	t.Logf("listening at %v", ln.Addr())

	game := NewGame(common.HexToAddress("0xfafa"), common.HexToAddress("0xbebe"))
	game.MakeMove(common.HexToAddress("0xfafa"), "d2d4")
	engine := NewChessEngine()
	engine.games = append(engine.games, game)

	rpcServer := rpc.NewServer()
	chessApi := CreateChessAPI(engine)
	rpcServer.RegisterName(chessApi.Namespace, chessApi.Service)
	httpServer := &http.Server{
		Handler: rpcServer,
	}
	defer httpServer.Shutdown(ctx)
	go httpServer.Serve(ln)

	client, err := rpc.Dial("http://" + ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to connect to the rpc: %v", err)
	}
	defer client.Close()

	var result string
	err = client.CallContext(ctx, &result, "chess_status")
	if err != nil {
		t.Fatalf("Failed to call chess_status: %v", err)
	}
	expected := `Game:  0
White: 0x000000000000000000000000000000000000fafa
Black: 0x000000000000000000000000000000000000BeBE
FEN:   rnbqkbnr/pppppppp/8/8/3P4/8/PPP1PPPP/RNBQKBNR b KQkq d3 0 1

 A B C D E F G H
8♜ ♞ ♝ ♛ ♚ ♝ ♞ ♜ 
7♟ ♟ ♟ ♟ ♟ ♟ ♟ ♟ 
6- - - - - - - - 
5- - - - - - - - 
4- - - ♙ - - - - 
3- - - - - - - - 
2♙ ♙ ♙ - ♙ ♙ ♙ ♙ 
1♖ ♘ ♗ ♕ ♔ ♗ ♘ ♖ 
`
	if result != expected {
		t.Fatalf("wrong result: %v", cmp.Diff(result, expected))
	}
}
