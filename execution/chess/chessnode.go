package chess

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// msg_spec = 1 byte (op)
// if op == 1:
//     msg_spec += 20 bytes (white address) + 20 bytes (black address)
// elif op == 2:
//     msg_spec += 8 bytes (gid) + (1 + 1 + 1 + 1) bytes (move)

type ChessOp byte

const (
	CreateGame ChessOp = iota
	MakeMove
	Invalid
)

type ChessTx []byte

func (tx ChessTx) GetOp() (ChessOp, error) {
	if len(tx) < 1 {
		return Invalid, fmt.Errorf("invalid tx: no op")
	}
	op := tx[0]
	if op >= byte(Invalid) {
		return Invalid, fmt.Errorf("invalid op: %v", op)
	}
	return ChessOp(op), nil
}

func (tx ChessTx) DecodeCreateGame() (white common.Address, black common.Address, err error) {
	if len(tx) != 1+2*common.AddressLength {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid create game tx: wrong len")
	}

	op := tx[0]
	tx = tx[1:]
	if op != byte(CreateGame) {
		return common.Address{}, common.Address{}, fmt.Errorf("invalid create game tx: wrong op")
	}

	white = common.Address(tx[:common.AddressLength])
	tx = tx[common.AddressLength:]

	black = common.Address(tx[:common.AddressLength])
	tx = tx[common.AddressLength:]

	return white, black, nil
}

func (tx ChessTx) DecodeMakeMove() (id uint64, move string, err error) {
	if len(tx) != 1+8+4 {
		return 0, "", fmt.Errorf("invalid make move tx: wrong len")
	}

	op := tx[0]
	tx = tx[1:]
	if op != byte(MakeMove) {
		return 0, "", fmt.Errorf("invalid make move tx: wrong op")
	}

	id = binary.BigEndian.Uint64(tx[:8])
	tx = tx[8:]

	move = string(tx[:4])
	tx = tx[4:]

	return id, move, nil
}

type ChessNode struct {
	games []*Game
}

func NewChessNode() *ChessNode {
	return &ChessNode{}
}

func (n *ChessNode) Process(from common.Address, tx ChessTx) error {
	op, err := tx.GetOp()
	if err != nil {
		return err
	}
	switch op {
	case CreateGame:
		white, black, err := tx.DecodeCreateGame()
		if err != nil {
			return err
		}
		id := len(n.games)
		n.games = append(n.games, NewGame(white, black))
		log.Info("Created chess game", "id", id, "white", white, "black", black)
	case MakeMove:
		id, move, err := tx.DecodeMakeMove()
		if err != nil {
			return err
		}
		if id >= uint64(len(n.games)) {
			return fmt.Errorf("game not found: %v", id)
		}
		err = n.games[id].MakeMove(from, move)
		if err != nil {
			return err
		}
		log.Info("Made a move", "id", id, "player", from, "move", move)
	}
	return nil
}

func (n *ChessNode) Status() string {
	status := []string{}
	for i, game := range n.games {
		if i > 0 {
			status = append(status, "---")
		}
		status = append(status, fmt.Sprintf("Game:  %v", i))
		status = append(status, fmt.Sprintf("White: %v", game.White()))
		status = append(status, fmt.Sprintf("Black: %v", game.Black()))
		status = append(status, fmt.Sprintf("FEN:   %v", game.FEN()))
		status = append(status, game.Draw())
	}
	return strings.Join(status, "\n")
}
