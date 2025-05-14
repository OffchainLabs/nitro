package chess

import (
	"fmt"

	"github.com/corentings/chess"
	"github.com/ethereum/go-ethereum/common"
)

type Game struct {
	game  *chess.Game
	white common.Address
	black common.Address
}

func NewGame(white, black common.Address) *Game {
	return &Game{
		game:  chess.NewGame(),
		white: white,
		black: black,
	}
}

func (g *Game) MakeMove(player common.Address, move string) error {
	if g.game.Outcome() != chess.NoOutcome {
		return fmt.Errorf("game is already over")
	}
	turn := g.game.Position().Turn()
	if turn == chess.White {
		if player != g.white {
			return fmt.Errorf("wrong player, expected white (%v)", g.white)
		}
	} else if player != g.black {
		return fmt.Errorf("wrong player, expected black (%v)", g.black)
	}
	return g.game.MoveStr(string(move))
}

func (g *Game) FEN() string {
	return g.game.Position().String()
}

func (g *Game) Draw() string {
	return g.game.Position().Board().Draw()
}

func (g *Game) White() common.Address {
	return g.white
}

func (g *Game) Black() common.Address {
	return g.black
}
