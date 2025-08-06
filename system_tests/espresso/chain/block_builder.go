package chain

import (
	"context"
	"fmt"
	"sync"

	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
)

// BlockBuilder is an interface that allows for the definition of different
// Block Builders.
//
// This is utilized by the Mock Espresso Chain to determine which transactions
// are included in each block being built.
//
// Having this interface allows for us to swap out different BlockBuilder
// implementations, so that we can emulate ideal, and realistic scenarios.
type BlockBuilder interface {
	// SubmitTransaction submits a transaction to the block builder.
	SubmitTransaction(ctx context.Context, tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error)

	// NextTransactions returns the next set of transactions to be included in
	// the block being built.
	NextTransactions() ([]espresso_common.Transaction, error)
}

// IdealBlockBuilder is a BlockBuilder that simply collects transactions
// and returns them when NextTransactions is called.
//
// It is ideal in that it does not impose any restrictions on the size of
// the transactions, and simply returns all transactions that have been
// submitted so far.
type IdealBlockBuilder struct {
	lock         sync.Mutex
	transactions []espresso_common.Transaction
}

// NewIdealBlockBuilder creates a new IdealBlockBuilder instance.
func NewIdealBlockBuilder() *IdealBlockBuilder {
	return &IdealBlockBuilder{}
}

// SubmitTransaction implements BlockBuilder
func (b *IdealBlockBuilder) SubmitTransaction(ctx context.Context, tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error) {
	// Need to generate a unique hash for the transaction
	tag, err := TransactionTaggedBase64(tx)
	if err != nil {
		return nil, ErrorSubmitTransaction{
			Cause: err,
		}
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	b.transactions = append(b.transactions, tx)
	return tag, nil
}

// NextTransactions implements BlockBuilder
func (b *IdealBlockBuilder) NextTransactions() ([]espresso_common.Transaction, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// Return the transactions that have been submitted so far.
	txs := b.transactions
	b.transactions = nil // Clear the transactions after returning them.
	return txs, nil
}

// MaxSizeRestrictedBuilder is a BlockBuilder that restricts the size combined
// transactions that are returned when NextTransactions is called.
//
// It additionally ensures that each individual transaction submitted does not
// exceed the maximum size allowed for a transaction.
type MaxSizeRestrictedBuilder struct {
	lock                sync.Mutex
	pendingTransactions []espresso_common.Transaction
	maxSize             uint64
}

// NewMaxSizeRestrictedBuilder creates a new MaxSizeRestrictedBuilder with the
// specified maximum size for transactions.
func NewMaxSizeRestrictedBuilder(maxSize uint64) *MaxSizeRestrictedBuilder {
	return &MaxSizeRestrictedBuilder{
		maxSize: maxSize,
	}
}

// ErrorTransactionTooLarge is an error that indicates that an individual
// submitted transaction exceeds the maximum allowed size for a block by
// itself.
type ErrorTransactionTooLarge struct {
	Transaction espresso_common.Transaction
	MaxSize     uint64
}

// Error implements error
func (e ErrorTransactionTooLarge) Error() string {
	return fmt.Sprintf("transaction size %d exceeds maximum allowed size %d", len(e.Transaction.Payload), e.MaxSize)
}

// SubmitTransaction implements BlockBuilder
func (b *MaxSizeRestrictedBuilder) SubmitTransaction(ctx context.Context, tx espresso_common.Transaction) (*espresso_common.TaggedBase64, error) {
	// Need to generate a unique hash for the transaction
	tag, err := TransactionTaggedBase64(tx)
	if err != nil {
		return nil, ErrorSubmitTransaction{
			Cause: err,
		}
	}

	if uint64(len(tx.Payload)) > b.maxSize {
		return nil, ErrorTransactionTooLarge{
			Transaction: tx,
			MaxSize:     b.maxSize,
		}
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	b.pendingTransactions = append(b.pendingTransactions, tx)
	return tag, nil
}

// NextTransactions implements BlockBuilder
func (b *MaxSizeRestrictedBuilder) NextTransactions() ([]espresso_common.Transaction, error) {
	if len(b.pendingTransactions) == 0 {
		return nil, nil // No transactions to return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	nextPendingTxns := make([]espresso_common.Transaction, 0, len(b.pendingTransactions))
	selectedTransactions := make([]espresso_common.Transaction, 0, len(b.pendingTransactions))

	var totalSize uint64

	for _, tx := range b.pendingTransactions {
		txSize := uint64(len(tx.Payload))
		if totalSize+txSize > b.maxSize {
			nextPendingTxns = append(nextPendingTxns, tx)
			continue
		}

		selectedTransactions = append(selectedTransactions, tx)
		totalSize += txSize
	}

	// Clear the transactions after returning them.
	b.pendingTransactions = nextPendingTxns
	return selectedTransactions, nil
}
