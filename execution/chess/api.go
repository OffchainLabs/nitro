package chess

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

type ChessAPI struct {
	engine *ChessEngine
}

func (a *ChessAPI) Status() string {
	return a.engine.Status()
}

func CreateChessAPI(engine *ChessEngine) rpc.API {
	return rpc.API{
		Namespace: "chess",
		Service: &ChessAPI{
			engine,
		},
	}
}

func RegisterChessAPI(stack *node.Node, engine *ChessEngine) {
	apis := []rpc.API{CreateChessAPI(engine)}
	stack.RegisterAPIs(apis)
}
