// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package systest

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	arbtest "github.com/offchainlabs/nitro/system_tests"
)

// stubEthAPI backs an in-proc eth namespace so handle methods run against a
// scripted chain: sent txs mine instantly into one block each.
type stubEthAPI struct {
	mu       sync.Mutex
	head     uint64
	nonce    hexutil.Uint64
	sends    int
	receipts map[common.Hash]*types.Receipt
}

func (s *stubEthAPI) mint(hash common.Hash, status uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head++
	s.receipts[hash] = &types.Receipt{
		Type:              types.DynamicFeeTxType,
		Status:            status,
		CumulativeGasUsed: 21000,
		GasUsed:           21000,
		TxHash:            hash,
		BlockHash:         common.BigToHash(new(big.Int).SetUint64(s.head)),
		BlockNumber:       new(big.Int).SetUint64(s.head),
		Logs:              []*types.Log{},
	}
}

func (s *stubEthAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receipts[hash], nil
}

func (s *stubEthAPI) GetBlockByNumber(ctx context.Context, num rpc.BlockNumber, full bool) (*types.Header, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &types.Header{
		Number:     new(big.Int).SetUint64(s.head),
		Difficulty: big.NewInt(0),
	}, nil
}

func (s *stubEthAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) error {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return err
	}
	s.mu.Lock()
	s.sends++
	s.mu.Unlock()
	s.mint(tx.Hash(), types.ReceiptStatusSuccessful)
	return nil
}

func (s *stubEthAPI) GetTransactionCount(ctx context.Context, addr common.Address, blk rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	return s.nonce, nil
}

func newStubHandle(t *testing.T, api *stubEthAPI) *ChainHandle {
	t.Helper()
	api.receipts = map[common.Hash]*types.Receipt{}
	srv := rpc.NewServer()
	if err := srv.RegisterName("eth", api); err != nil {
		t.Fatalf("RegisterName: %v", err)
	}
	t.Cleanup(srv.Stop)
	client := ethclient.NewClient(rpc.DialInProc(srv))
	t.Cleanup(client.Close)
	return &ChainHandle{
		Client: client,
		Info:   arbtest.NewArbTestInfo(t, big.NewInt(412346)),
		e:      newEnv(t, t.Context(), Spec{}),
		name:   "stub",
	}
}

func TestHandleHeaderByNumber(t *testing.T) {
	api := &stubEthAPI{head: 7}
	h := newStubHandle(t, api)
	if got := h.HeaderByNumber(nil).Number.Uint64(); got != 7 {
		t.Fatalf("head = %d, want 7", got)
	}
}

func TestHandlePendingNonceAt(t *testing.T) {
	api := &stubEthAPI{nonce: 42}
	h := newStubHandle(t, api)
	if got := h.PendingNonceAt(common.Address{}); got != 42 {
		t.Fatalf("nonce = %d, want 42", got)
	}
}

func TestHandleWaitForTx(t *testing.T) {
	api := &stubEthAPI{}
	h := newStubHandle(t, api)
	tx := h.Info.PrepareTx("Faucet", "Faucet", h.Info.TransferGas, common.Big1, nil)
	api.mint(tx.Hash(), types.ReceiptStatusSuccessful)
	receipt, err := h.WaitForTx(tx, time.Second)
	if err != nil {
		t.Fatalf("WaitForTx: %v", err)
	}
	if receipt.TxHash != tx.Hash() {
		t.Fatalf("receipt hash = %s, want %s", receipt.TxHash, tx.Hash())
	}

	missing := h.Info.PrepareTx("Faucet", "Faucet", h.Info.TransferGas, common.Big1, nil)
	if _, err := h.WaitForTx(missing, 50*time.Millisecond); err == nil {
		t.Fatal("want timeout error for a tx that never mines")
	}
}

func TestHandleEnsureTxFailed(t *testing.T) {
	api := &stubEthAPI{}
	h := newStubHandle(t, api)
	tx := h.Info.PrepareTx("Faucet", "Faucet", h.Info.TransferGas, common.Big1, nil)
	api.mint(tx.Hash(), types.ReceiptStatusFailed)
	if got := h.EnsureTxFailed(tx).Status; got != types.ReceiptStatusFailed {
		t.Fatalf("status = %d, want failed", got)
	}
}

func TestHandleAdvanceBlocks(t *testing.T) {
	api := &stubEthAPI{}
	h := newStubHandle(t, api)
	h.AdvanceBlocks(3)
	if api.sends != 3 || api.head != 3 {
		t.Fatalf("sends = %d, head = %d, want 3, 3", api.sends, api.head)
	}
}

func TestHandleSendWaitTestTransactions(t *testing.T) {
	api := &stubEthAPI{}
	h := newStubHandle(t, api)
	txs := []*types.Transaction{
		h.Info.PrepareTx("Faucet", "Faucet", h.Info.TransferGas, common.Big1, nil),
		h.Info.PrepareTx("Faucet", "Faucet", h.Info.TransferGas, common.Big1, nil),
	}
	receipts := h.SendWaitTestTransactions(txs)
	if len(receipts) != 2 || api.sends != 2 {
		t.Fatalf("receipts = %d, sends = %d, want 2, 2", len(receipts), api.sends)
	}
	for i, r := range receipts {
		if r.TxHash != txs[i].Hash() {
			t.Fatalf("receipt %d hash = %s, want %s", i, r.TxHash, txs[i].Hash())
		}
	}
}
