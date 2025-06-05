package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/offchainlabs/nitro/arbutil"
)

type txsFetcherForBlock struct {
	header           *types.Header
	preimageResolver preimageResolver
}

func (tf *txsFetcherForBlock) TransactionsByHeader(
	ctx context.Context,
	parentChainHeaderHash common.Hash,
) (types.Transactions, error) {
	preimageDB := &DB{
		Hooks{
			Get: func(key []byte) []byte {
				if len(key) != 32 {
					panic(fmt.Sprintf("expected 32 byte key query, but got %d bytes: %x", len(key), key))
				}
				preimage, err := tf.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, common.BytesToHash(key))
				if err != nil {
					panic(fmt.Errorf("error resolving preimage for %#x: %w", key, err))
				}
				return preimage
			},
			Put: func(key []byte, value []byte) {
				panic("put not supported")
			},
			Delete: func(key []byte) {
				panic("delete not supported")
			},
		},
	}
	tdb := triedb.NewDatabase(preimageDB, nil)
	tr, err := trie.New(trie.TrieID(tf.header.TxHash), tdb)
	if err != nil {
		panic(err)
	}
	iter, err := tr.NodeIterator(nil)
	if err != nil {
		panic(err)
	}
	var values [][]byte
	var keys []uint64
	for iter.Next(true) {
		if iter.Leaf() {
			k := iter.LeafKey()
			var x uint64
			err := rlp.DecodeBytes(k, &x)
			if err != nil {
				panic(fmt.Errorf("invalid key: %w", err))
			}
			keys = append(keys, x)
			values = append(values, iter.LeafBlob())
		}
	}
	out := make([]hexutil.Bytes, len(values))
	for i, x := range keys {
		if x >= uint64(len(values)) {
			panic(fmt.Sprintf("bad key: %d", x))
		}
		if out[x] != nil {
			panic(fmt.Sprintf("duplicate key %d", x))
		}
		out[x] = values[i]
	}
	txs := make(types.Transactions, 0, len(out))
	for _, v := range out {
		tx := new(types.Transaction)
		if err := rlp.Decode(bytes.NewBuffer(v), &tx); err != nil {
			return nil, fmt.Errorf("error decoding transaction: %w", err)
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
