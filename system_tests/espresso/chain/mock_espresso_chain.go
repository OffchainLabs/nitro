package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	espresso_client "github.com/EspressoSystems/espresso-network/sdks/go/client"
	tagged_base64 "github.com/EspressoSystems/espresso-network/sdks/go/tagged-base64"
	espresso_types "github.com/EspressoSystems/espresso-network/sdks/go/types"
	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
)

// MockEspressoChain is a mock implementation of an Espresso Blockchain for
// testing purposes.
// It's meant to simulate a basic, likely incorrect and incomplete,
// implementation of an Espresso chain.
//
// This implementation is not concerned with specifics or details about the
// Espresso chain, and is not meant to be utilized as a sufficient substitute
// for a real Espresso chain.
//
// What matters most about this implementation is that it allows the user to
// control when new blocks are produced, and allows the user to submit, and
// check for the existence of, transactions within the chain.
type MockEspressoChain struct {
	UnimplementedEspressoClient
	Lock sync.RWMutex

	Height       uint64
	BlockBuilder BlockBuilder

	BlockHeightStore map[uint64]BlockDetail
	BlockHashStore   map[string]BlockDetail // This is not used in the mock, but kept for interface compatibility
	TxnHashStore     map[string]TransactionDetail
}

// Ensure MockEspressoChain implements the EspressoClient interface.
var _ espresso_client.EspressoClient = &MockEspressoChain{}

// MockEspressoChainOption is a functional option type for configuring
// MockEspressoChain instances.
type MockEspressoChainOption func(*MockEspressoChain)

// WithBlockHeight sets the initial block height for the MockEspressoChain.
func WithBlockHeight(height uint64) MockEspressoChainOption {
	return func(chain *MockEspressoChain) {
		chain.Height = height
	}
}

// WithBuilder sets the BlockBuilder for the MockEspressoChain.
func WithBuilder(builder BlockBuilder) MockEspressoChainOption {
	return func(chain *MockEspressoChain) {
		chain.BlockBuilder = builder
	}
}

// NewMockEspressoChain creates a new instance of MockEspressoChain with
// initialized stores for blocks and transactions.
func NewMockEspressoChain(options ...MockEspressoChainOption) *MockEspressoChain {
	chain := &MockEspressoChain{
		BlockBuilder:     NewIdealBlockBuilder(),
		BlockHeightStore: make(map[uint64]BlockDetail),
		BlockHashStore:   make(map[string]BlockDetail),
		TxnHashStore:     make(map[string]TransactionDetail),
	}

	for _, option := range options {
		option(chain)
	}

	return chain
}

// ErrorBlockNotFoundForHeight is an error type that indicates a block was not
// found at a given height in the mock Espresso chain.
type ErrorBlockNotFoundForHeight struct {
	Height uint64
}

// Error implements error
func (e ErrorBlockNotFoundForHeight) Error() string {
	return fmt.Sprintf("block not found at height: %d", e.Height)
}

// ErrorFailedToComputeBlockHash is an error type that indicates a failure to
// compute the hash of a block in the mock Espresso chain.
type ErrorFailedToComputeBlockHash struct {
	Cause error
}

// Error implements error
func (e ErrorFailedToComputeBlockHash) Error() string {
	return fmt.Sprintf("failed to compute block hash: %v", e.Cause)
}

// Advance simulates the advancement of the Espresso chain by creating a new
// block with the list of pending transactions, advancing the height, and
// storing the block and transaction details in their respective stores.
func (m *MockEspressoChain) Advance() {
	m.Lock.Lock()
	defer m.Lock.Unlock()

	// Let's "build" a new block
	height := m.Height
	m.Height++

	pendingTxs, err := m.BlockBuilder.NextTransactions()
	if err != nil {
		panic(fmt.Sprintf("failed to get next transactions: %v", err))
	}

	block := BlockDetail{
		Height:       height,
		NumTxns:      uint64(len(pendingTxs)),
		Transactions: pendingTxs,
	}

	blockTag, err := block.TaggedBase64()
	if err != nil {
		panic(fmt.Sprintf("failed to create tagged base64 for block: %v", err))
	}

	m.BlockHeightStore[height] = block
	m.BlockHashStore[blockTag.String()] = block

	for i, l := uint64(0), uint64(len(block.Transactions)); i < l; i++ {
		tx := block.Transactions[i]
		txTag, err := TransactionTaggedBase64(tx)
		if err != nil {
			panic(fmt.Sprintf("failed to create tagged base64 for transaction: %v", err))
		}

		txDetail := TransactionDetail{
			Block:       height,
			Index:       uint64(i),    // Index is not used in this mock
			Namespace:   tx.Namespace, // Namespace is not used in this mock
			Size:        uint64(len(tx.Payload)),
			Transaction: tx,
		}

		m.TxnHashStore[txTag.String()] = txDetail
	}
}

// FetchTransactionsInBlock retrieves the transactions in a block at a given
// height and namespace. It returns an error if the block is not found.
func (m *MockEspressoChain) FetchTransactionsInBlock(ctx context.Context, blockHeight uint64, namespace uint64) (espresso_client.TransactionsInBlock, error) {
	m.Lock.RLock()
	block, blockOk := m.BlockHeightStore[blockHeight]
	m.Lock.RUnlock()
	if !blockOk {
		return espresso_client.TransactionsInBlock{}, ErrorBlockNotFoundForHeight{Height: blockHeight}
	}

	txns := make([]espresso_types.Bytes, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txns = append(txns, espresso_types.Bytes(tx.Payload))
	}
	return espresso_client.TransactionsInBlock{
		Transactions: txns,
	}, nil
}

// ErrorInvalidHash is an error type that indicates an invalid hash was provided
// when trying to fetch a transaction by its hash in the mock Espresso chain.
type ErrorInvalidHash struct {
	Hash *espresso_common.TaggedBase64
}

// Error implements error
func (e ErrorInvalidHash) Error() string {
	if e.Hash == nil {
		return "invalid hash: nil"
	}

	return fmt.Sprintf("invalid hash: %s", e.Hash.String())
}

// ErrorTransactionNotFoundForHash is an error type that indicates a transaction
// was not found for a given hash in the mock Espresso chain.
type ErrorTransactionNotFoundForHash struct {
	Hash espresso_common.TaggedBase64
}

// Error implements error
func (e ErrorTransactionNotFoundForHash) Error() string {
	return fmt.Sprintf("transaction not found for hash: %s", e.Hash.String())
}

// FetchTransactionByHash retrieves a transaction by its hash. It returns an
// error if the hash is nil or if the transaction is not found in the mock
// Espresso chain.
func (m *MockEspressoChain) FetchTransactionByHash(ctx context.Context, hash *espresso_common.TaggedBase64) (espresso_types.TransactionQueryData, error) {
	if hash == nil {
		return espresso_types.TransactionQueryData{}, ErrorInvalidHash{
			Hash: hash,
		}
	}

	key := hash.String()
	m.Lock.RLock()
	txDetail, txDetailOk := m.TxnHashStore[key]
	m.Lock.RUnlock()
	if !txDetailOk {
		return espresso_types.TransactionQueryData{}, ErrorTransactionNotFoundForHash{
			Hash: *hash,
		}
	}

	blockDetail, blockDetailOk := m.BlockHeightStore[txDetail.Block]
	if !blockDetailOk {
		return espresso_types.TransactionQueryData{}, ErrorBlockNotFoundForHeight{Height: txDetail.Block}
	}

	blockHash, err := blockDetail.TaggedBase64()
	if err != nil {
		return espresso_types.TransactionQueryData{}, ErrorFailedToComputeBlockHash{Cause: err}
	}

	return espresso_types.TransactionQueryData{
		Transaction: txDetail.Transaction,
		Hash:        hash,
		Index:       txDetail.Index,
		Proof:       json.RawMessage(`[]`), // Mocking proof as empty JSON array
		BlockHash:   blockHash,
		BlockHeight: blockDetail.Height,
	}, nil
}

// FetchExplorerTransactionByHash retrieves transaction details for a given
// hash from the mock Espresso chain.
func (m *MockEspressoChain) FetchExplorerTransactionByHash(ctx context.Context, hash *espresso_types.TaggedBase64) (espresso_types.ExplorerTransactionQueryData, error) {
	if hash == nil {
		return espresso_types.ExplorerTransactionQueryData{}, ErrorInvalidHash{
			Hash: hash,
		}
	}

	key := hash.String()
	m.Lock.RLock()
	txDetail, txDetailOk := m.TxnHashStore[key]
	m.Lock.RUnlock()
	if !txDetailOk {
		return espresso_types.ExplorerTransactionQueryData{}, ErrorTransactionNotFoundForHash{
			Hash: *hash,
		}
	}

	return espresso_types.ExplorerTransactionQueryData{
		TransactionsDetails: espresso_common.ExplorerTransactionsDetails{
			ExplorerDetails: espresso_common.ExplorerDetails{
				BlockHeight: txDetail.Block,
				Hash:        *hash,
			},
		},
	}, nil
}

// ErrorSubmitTransaction is an error type that indicates a failure to submit
// a transaction to the mock Espresso chain.
type ErrorSubmitTransaction struct {
	Cause error
}

// Error implements error
func (e ErrorSubmitTransaction) Error() string {
	return fmt.Sprintf("failed to submit transaction: %v", e.Cause)
}

// SubmitTransaction simulates the submission of a transaction to the mock
// Espresso chain. It generates a unique hash for the transaction, appends it
// to the list of pending transactions, and returns the generated hash.
func (m *MockEspressoChain) SubmitTransaction(ctx context.Context, tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error) {
	return m.BlockBuilder.SubmitTransaction(ctx, tx)
}

// FetchLatestBlockHeight retrieves the latest block height from the mock
// Espresso chain. It returns the current height and does not produce an error.
// This method is used to simulate the behavior of fetching the latest block
// height in a real Espresso chain.
//
// NOTE: This method does not acquire a lock, as it only reads the current
// height, which should be safe to do.
func (m *MockEspressoChain) FetchLatestBlockHeight(ctx context.Context) (uint64, error) {
	return m.Height, nil
}

// ProduceEspressoBlocksAtInterval is a convenience function that calls
// Advance on the provided MockEspressoChain at a specified interval.
//
// It is meant to be spawned in a separate goroutine to simulate the
// production of blocks in the Espresso chain at regular intervals.
//
// NOTE: This can be used to simulate a live Espresso chain that produces blocks
// at a given interval, which can be useful for testing purposes.
func ProduceEspressoBlocksAtInterval(ctx context.Context, chain *MockEspressoChain, interval time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			chain.Advance()
		}
	}
}

// BlockDetail represents the minimal amount of information needed to represent
// a Block in the Espresso chain for testing purposes.
type BlockDetail struct {
	Height       uint64
	NumTxns      uint64
	Transactions []espresso_common.Transaction
}

// Commit returns a commitment for the BlockDetail which is comprised of its
// contents.
func (b BlockDetail) Commit() espresso_common.Commitment {
	builder := espresso_common.NewRawCommitmentBuilder("BlockDetail").
		Uint64Field("height", b.Height)

	for _, tx := range b.Transactions {
		tCommit := tx.Commit()
		builder.VarSizeBytes(espresso_common.Bytes(tCommit[:]))
	}

	return builder.
		Finalize()
}

// TaggedBase64 is a convenience method to create a TaggedBase64
func (b BlockDetail) TaggedBase64() (*espresso_common.TaggedBase64, error) {
	commitment := b.Commit()
	tag, err := tagged_base64.New("MOCK-BLOCK", commitment[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create tagged base64 for block: %w", err)
	}
	return tag, nil
}

// TransactionDetail represents the minimal amount of information needed to
// represent a Transaction in the Espresso chain for testing purposes.
type TransactionDetail struct {
	Block       uint64
	Index       uint64
	Namespace   uint64
	Size        uint64
	Transaction espresso_common.Transaction
}

// TransactionTaggedBase64 is a convenience method to create a TaggedBase64
// for a TransactionDetail.
func TransactionTaggedBase64(tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error) {
	commitment := tx.Commit()
	tag, err := tagged_base64.New("MOCK-TXN", commitment[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create tagged base64 for transaction: %w", err)
	}

	return tag, nil
}
