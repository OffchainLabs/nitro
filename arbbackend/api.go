package arbbackend

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type ArbEthAPI struct {
	b *ArbBackend
}

func NewArbEthAPI(backend *ArbBackend) *ArbEthAPI {
	return &ArbEthAPI{
		b: backend,
	}
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (a *ArbEthAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}

	a.b.EnqueueL2Message(tx)

	return tx.Hash(), nil
}

func createAPIs(backend *ArbBackend) []rpc.API {
	var apis []rpc.API

	apis = append(apis,
		rpc.API{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewArbEthAPI(backend),
			Public:    true,
		})

	return apis
}
