package arbtest

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGenesisBlockHashUnchanged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	cleanup := builder.Build(t)
	defer cleanup()

	genesisHeader, err := builder.L2.Client.HeaderByNumber(ctx, big.NewInt(0))
	Require(t, err)
	t.Log(genesisHeader)

	expectedHash := common.HexToHash("0xe00eb9c04ca8df66fc70c7fca2d3755505123bd764140d335178b5d7d47fe529")
	if genesisHeader.Hash() != expectedHash {
		Fatal(t, "genesis hash changed, have: ", genesisHeader.Hash(), "want", expectedHash)
	}
}
