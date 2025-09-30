//go:build !debugblock

package arbos

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func debugBlockStateUpdate(_ *state.StateDB, _ *big.Int, _ *params.ChainConfig) {
	log.Warn("debugBlockStateUpdate is not supported in this build")
}
