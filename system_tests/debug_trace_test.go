package arbtest

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestDebugTraceCallForRecentBlock(t *testing.T) {
	threads := 128
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Caching.Archive = true
	cleanup := builder.Build(t)
	defer cleanup()
	builder.L2Info.GenerateAccount("User2")
	builder.L2Info.GenerateAccount("User3")

	errors := make(chan error, threads+1)
	senderDone := make(chan struct{})
	go func() {
		defer close(senderDone)
		for ctx.Err() == nil {
			tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, new(big.Int).Lsh(big.NewInt(1), 128), nil)
			err := builder.L2.Client.SendTransaction(ctx, tx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				errors <- err
				return
			}
			_, err = builder.L2.EnsureTxSucceeded(tx)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				errors <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	type TransactionArgs struct {
		From                 *common.Address   `json:"from"`
		To                   *common.Address   `json:"to"`
		Gas                  *hexutil.Uint64   `json:"gas"`
		GasPrice             *hexutil.Big      `json:"gasPrice"`
		MaxFeePerGas         *hexutil.Big      `json:"maxFeePerGas"`
		MaxPriorityFeePerGas *hexutil.Big      `json:"maxPriorityFeePerGas"`
		Value                *hexutil.Big      `json:"value"`
		Nonce                *hexutil.Uint64   `json:"nonce"`
		SkipL1Charging       *bool             `json:"skipL1Charging"`
		Data                 *hexutil.Bytes    `json:"data"`
		Input                *hexutil.Bytes    `json:"input"`
		AccessList           *types.AccessList `json:"accessList,omitempty"`
		ChainID              *hexutil.Big      `json:"chainId,omitempty"`
	}
	rpcClient := builder.L2.ConsensusNode.Stack.Attach()
	sometx := builder.L2Info.PrepareTx("User2", "User3", builder.L2Info.TransferGas, common.Big1, nil)
	from := builder.L2Info.GetAddress("User2")
	to := sometx.To()
	gas := sometx.Gas()
	maxFeePerGas := sometx.GasFeeCap()
	value := sometx.Value()
	nonce := sometx.Nonce()
	data := sometx.Data()
	txargs := TransactionArgs{
		From:         &from,
		To:           to,
		Gas:          (*hexutil.Uint64)(&gas),
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
		Value:        (*hexutil.Big)(value),
		Nonce:        (*hexutil.Uint64)(&nonce),
		Data:         (*hexutil.Bytes)(&data),
	}
	db := builder.L2.ExecNode.Backend.ChainDb()

	i := 1
	var mtx sync.RWMutex
	var wgTrace sync.WaitGroup
	for j := 0; j < threads && ctx.Err() == nil; j++ {
		wgTrace.Add(1)
		go func() {
			defer wgTrace.Done()
			mtx.RLock()
			blockNumber := i
			mtx.RUnlock()
			for blockNumber < 300 && ctx.Err() == nil {
				var err error
				prefix := make([]byte, 8)
				binary.BigEndian.PutUint64(prefix, uint64(blockNumber))
				prefix = append([]byte("b"), prefix...)
				it := db.NewIterator(prefix, nil)
				defer it.Release()
				if it.Next() {
					key := it.Key()
					if len(key) != len(prefix)+common.HashLength {
						Fatal(t, "Wrong key length, have:", len(key), "want:", len(prefix)+common.HashLength)
					}
					blockHash := common.BytesToHash(key[len(prefix):])
					start := time.Now()
					for ctx.Err() == nil {
						var res json.RawMessage
						err = rpcClient.CallContext(ctx, &res, "debug_traceCall", txargs, blockHash, nil)
						if err == nil {
							mtx.Lock()
							if blockNumber == i {
								i++
							}
							mtx.Unlock()
							break
						}
						if ctx.Err() != nil {
							return
						}
						if !strings.Contains(err.Error(), "not currently canonical") && !strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "missing trie node") {
							errors <- err
							return
						}
						if time.Since(start) > 5*time.Second {
							errors <- fmt.Errorf("timeout - failed to trace call for more then 5 seconds, block: %d, err: %w", blockNumber, err)
							return
						}
					}
				}
				it.Release()
				mtx.RLock()
				blockNumber = i
				mtx.RUnlock()
			}
		}()
	}
	traceDone := make(chan struct{})
	go func() {
		wgTrace.Wait()
		close(traceDone)
	}()

	select {
	case <-traceDone:
		cancel()
	case <-senderDone:
		cancel()
	case err := <-errors:
		t.Error(err)
		cancel()
	}
	<-traceDone
	<-senderDone
	close(errors)
	for err := range errors {
		if err != nil {
			t.Error(err)
		}
	}
}
