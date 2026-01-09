package melreplay

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
)

type txFetcherForBlock struct {
	header           *types.Header
	preimageResolver PreimageResolver
}

func NewTransactionFetcher(header *types.Header, preimageResolver PreimageResolver) melextraction.TransactionFetcher {
	return &txFetcherForBlock{header, preimageResolver}
}

// TransactionByLog fetches the tx for a specific transaction index by walking
// the tx trie of the block header. It uses the preimage resolver to fetch the preimages
// of the trie nodes as needed.
func (tf *txFetcherForBlock) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	tx, err := fetchObjectFromTrie[types.Transaction](tf.header.TxHash, log.TxIndex, tf.preimageResolver)
	if err != nil {
		return nil, err
	}
	return tx, err
}
