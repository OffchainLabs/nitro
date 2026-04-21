// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package pruner

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/offchainlabs/nitro/arbnode/mel"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
)

const (
	testChildFinalizedBlock = uint64(4242)
	testPollInterval        = 10 * time.Second
)

type fakeHeaderReader struct {
	header *types.Header
	err    error
}

func (f *fakeHeaderReader) HeaderByNumber(_ context.Context, _ *big.Int) (*types.Header, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.header, nil
}

type lookupCall struct {
	from uint64
	to   uint64
}

type fakeBridge struct {
	calls    []lookupCall
	messages []*mel.DelayedInboxMessage
	err      error
}

func (f *fakeBridge) LookupMessagesInRange(
	_ context.Context,
	from, to *big.Int,
	_ arbostypes.FallibleBatchFetcherWithParentBlock,
) ([]*mel.DelayedInboxMessage, error) {
	f.calls = append(f.calls, lookupCall{from: from.Uint64(), to: to.Uint64()})
	if f.err != nil {
		return nil, f.err
	}
	var out []*mel.DelayedInboxMessage
	for _, m := range f.messages {
		block := m.ParentChainBlockNumber
		if block >= from.Uint64() && block <= to.Uint64() {
			out = append(out, m)
		}
	}
	return out, nil
}

type fakeManager struct {
	filtered       map[common.Hash]bool
	deleted        []common.Hash
	isFilteredErr  error
	deleteErr      error
	callBlockSeen  []*big.Int
	deleteCallHash []common.Hash
}

func (f *fakeManager) IsTransactionFiltered(opts *bind.CallOpts, txHash [32]byte) (bool, error) {
	if opts != nil {
		f.callBlockSeen = append(f.callBlockSeen, opts.BlockNumber)
	}
	if f.isFilteredErr != nil {
		return false, f.isFilteredErr
	}
	return f.filtered[common.Hash(txHash)], nil
}

func (f *fakeManager) DeleteFilteredTransaction(_ *bind.TransactOpts, txHash [32]byte) (*types.Transaction, error) {
	f.deleteCallHash = append(f.deleteCallHash, common.Hash(txHash))
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	f.deleted = append(f.deleted, common.Hash(txHash))
	return types.NewTx(&types.LegacyTx{}), nil
}

func newHeader(t *testing.T, blockNumber uint64, nonce uint64) *types.Header {
	t.Helper()
	return &types.Header{
		Number: new(big.Int).SetUint64(blockNumber),
		Nonce:  types.EncodeNonce(nonce),
	}
}

func newDelayedMsg(t *testing.T, idx uint64, parentBlock uint64) *mel.DelayedInboxMessage {
	t.Helper()
	requestID := common.BigToHash(new(big.Int).SetUint64(idx))
	return &mel.DelayedInboxMessage{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EndOfBlock,
				Poster:      common.Address{},
				BlockNumber: parentBlock,
				Timestamp:   0,
				RequestId:   &requestID,
				L1BaseFee:   common.Big0,
			},
			L2msg: nil,
		},
		ParentChainBlockNumber: parentBlock,
	}
}

// newDepositMsg builds a DelayedInboxMessage carrying a single EthDeposit
// transaction. ParseL2Transactions produces an ArbitrumDepositTx for this
// kind, which gives the test a deterministic tx hash to assert against.
func newDepositMsg(t *testing.T, idx, parentBlock uint64, to common.Address, value *big.Int) *mel.DelayedInboxMessage {
	t.Helper()
	requestID := common.BigToHash(new(big.Int).SetUint64(idx))
	l2msg := append(append([]byte{}, to.Bytes()...), common.BigToHash(value).Bytes()...)
	return &mel.DelayedInboxMessage{
		Message: &arbostypes.L1IncomingMessage{
			Header: &arbostypes.L1IncomingMessageHeader{
				Kind:        arbostypes.L1MessageType_EthDeposit,
				Poster:      common.Address{},
				BlockNumber: parentBlock,
				Timestamp:   0,
				RequestId:   &requestID,
				L1BaseFee:   common.Big0,
			},
			L2msg: l2msg,
		},
		ParentChainBlockNumber: parentBlock,
	}
}

func newTestPruner(
	t *testing.T,
	childReader *fakeHeaderReader,
	parentReader *fakeHeaderReader,
	bridge *fakeBridge,
	manager *fakeManager,
	startIdx uint64,
	scanRange uint64,
) *Pruner {
	t.Helper()
	return &Pruner{
		config: &Config{
			Enable:                   true,
			StartDelayedMessageIndex: startIdx,
			PollInterval:             testPollInterval,
			ParentChainScanRange:     scanRange,
		},
		chainID: big.NewInt(412346),
		parent:  parentReader,
		child:   childReader,
		bridge:  bridge,
		manager: manager,
		txOpts:  &bind.TransactOpts{},
		nextIdx: startIdx,
	}
}

func TestStepIdleWhenNothingFinalized(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 0)}
	parent := &fakeHeaderReader{header: newHeader(t, 100, 0)}
	bridge := &fakeBridge{}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	got := p.step(context.Background())
	if got != testPollInterval {
		t.Fatalf("expected PollInterval when nothing is finalized, got %v", got)
	}
	if len(bridge.calls) != 0 {
		t.Fatalf("expected no bridge lookup when nothing to do, got %d", len(bridge.calls))
	}
	if p.nextIdx != 0 {
		t.Fatalf("nextIdx should not advance, got %d", p.nextIdx)
	}
}

func TestStepIdleWhenChildFinalizedErrors(t *testing.T) {
	child := &fakeHeaderReader{err: errors.New("boom")}
	parent := &fakeHeaderReader{header: newHeader(t, 100, 0)}
	bridge := &fakeBridge{}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	got := p.step(context.Background())
	if got != testPollInterval {
		t.Fatalf("expected PollInterval on child header error, got %v", got)
	}
	if len(bridge.calls) != 0 {
		t.Fatalf("expected no bridge lookup on error, got %d", len(bridge.calls))
	}
}

func TestStepProcessesUnfilteredTransaction(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 1)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	bridge := &fakeBridge{
		messages: []*mel.DelayedInboxMessage{newDelayedMsg(t, 0, 10)},
	}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	_ = p.step(context.Background())
	if p.nextIdx != 1 {
		t.Fatalf("expected nextIdx=1, got %d", p.nextIdx)
	}
	if len(manager.deleted) != 0 {
		t.Fatalf("expected no delete calls, got %d", len(manager.deleted))
	}
}

func expectedTxHash(t *testing.T, msg *mel.DelayedInboxMessage, chainID *big.Int) common.Hash {
	t.Helper()
	txs, err := arbos.ParseL2Transactions(msg.Message, chainID, 0)
	if err != nil {
		t.Fatalf("ParseL2Transactions: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 parsed tx, got %d", len(txs))
	}
	return txs[0].Hash()
}

func TestStepDeletesFilteredTransaction(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 1)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	msg := newDepositMsg(t, 0, 10, common.HexToAddress("0xabc"), big.NewInt(12345))
	bridge := &fakeBridge{messages: []*mel.DelayedInboxMessage{msg}}
	expected := expectedTxHash(t, msg, big.NewInt(412346))
	manager := &fakeManager{filtered: map[common.Hash]bool{expected: true}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	_ = p.step(context.Background())
	if p.nextIdx != 1 {
		t.Fatalf("expected nextIdx=1, got %d", p.nextIdx)
	}
	if len(manager.deleted) != 1 || manager.deleted[0] != expected {
		t.Fatalf("expected single delete for %v, got %v", expected.Hex(), manager.deleted)
	}
	if len(manager.callBlockSeen) == 0 || manager.callBlockSeen[0] == nil {
		t.Fatalf("expected IsTransactionFiltered with block override")
	}
	if manager.callBlockSeen[0].Uint64() != testChildFinalizedBlock {
		t.Fatalf("expected block override %d, got %d", testChildFinalizedBlock, manager.callBlockSeen[0].Uint64())
	}
}

func TestStepSkipsUnfilteredTransaction(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 1)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	msg := newDepositMsg(t, 0, 10, common.HexToAddress("0xabc"), big.NewInt(12345))
	bridge := &fakeBridge{messages: []*mel.DelayedInboxMessage{msg}}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	_ = p.step(context.Background())
	if p.nextIdx != 1 {
		t.Fatalf("expected nextIdx=1, got %d", p.nextIdx)
	}
	if len(manager.deleted) != 0 {
		t.Fatalf("expected no deletes, got %d", len(manager.deleted))
	}
}

func TestStepStopsAtUnfinalizedMessage(t *testing.T) {
	// delayedMessagesRead=1 means only idx=0 is finalized on L2; idx=1 must wait.
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 1)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	bridge := &fakeBridge{
		messages: []*mel.DelayedInboxMessage{
			newDelayedMsg(t, 0, 10),
			newDelayedMsg(t, 1, 42),
		},
	}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 150)

	got := p.step(context.Background())
	if got != testPollInterval {
		t.Fatalf("expected PollInterval after stopping at unfinalized message, got %v", got)
	}
	if p.nextIdx != 1 {
		t.Fatalf("expected nextIdx=1, got %d", p.nextIdx)
	}
	if p.scanBlock != 42 {
		t.Fatalf("expected scanBlock=42 so the unfinalized msg is re-scanned, got %d", p.scanBlock)
	}
}

func TestStepScansWindowRollover(t *testing.T) {
	// delayedMessagesRead promises messages beyond what any window returns.
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 1)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	bridge := &fakeBridge{}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 25)

	got := p.step(context.Background())
	if got != 0 {
		t.Fatalf("expected immediate re-run when work remains, got %v", got)
	}
	if p.scanBlock != 26 {
		t.Fatalf("expected scanBlock=26 after first empty window, got %d", p.scanBlock)
	}

	got = p.step(context.Background())
	if got != 0 {
		t.Fatalf("expected immediate re-run on second empty window, got %v", got)
	}
	if p.scanBlock != 52 {
		t.Fatalf("expected scanBlock=52 after second empty window, got %d", p.scanBlock)
	}
	if len(bridge.calls) != 2 {
		t.Fatalf("expected two bridge lookups, got %d", len(bridge.calls))
	}
	if bridge.calls[0].from != 0 || bridge.calls[0].to != 25 {
		t.Fatalf("unexpected first window: %+v", bridge.calls[0])
	}
	if bridge.calls[1].from != 26 || bridge.calls[1].to != 51 {
		t.Fatalf("unexpected second window: %+v", bridge.calls[1])
	}
}

func TestStepSkipsMessagesBelowStartIndex(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 5)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	msgs := []*mel.DelayedInboxMessage{
		newDelayedMsg(t, 0, 5),
		newDelayedMsg(t, 1, 6),
		newDelayedMsg(t, 2, 7),
		newDelayedMsg(t, 3, 8),
	}
	bridge := &fakeBridge{messages: msgs}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 3, 50)

	_ = p.step(context.Background())
	if p.nextIdx != 4 {
		t.Fatalf("expected nextIdx=4 (only idx=3 processed; idx=4 not yet emitted), got %d", p.nextIdx)
	}
}

func TestStepDetectsGap(t *testing.T) {
	child := &fakeHeaderReader{header: newHeader(t, testChildFinalizedBlock, 10)}
	parent := &fakeHeaderReader{header: newHeader(t, 200, 0)}
	// idx=0 is missing but idx=1 is in-range, so the pruner sees a gap.
	bridge := &fakeBridge{messages: []*mel.DelayedInboxMessage{newDelayedMsg(t, 1, 10)}}
	manager := &fakeManager{filtered: map[common.Hash]bool{}}
	p := newTestPruner(t, child, parent, bridge, manager, 0, 50)

	got := p.step(context.Background())
	if got != testPollInterval {
		t.Fatalf("expected PollInterval after detecting gap, got %v", got)
	}
	if p.nextIdx != 0 {
		t.Fatalf("nextIdx must not advance through a gap, got %d", p.nextIdx)
	}
}
