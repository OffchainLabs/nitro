package arbtest

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
)

const (
	SenderAccount            = "Sender"
	ReceiverAccount          = "Receiver"
	TransferAmount           = 1000000
	NewUncompressedSizeLimit = params.DefaultMaxUncompressedBatchSize * 2
)

func TestTooBigBatchGetsRejected(t *testing.T) {
	builder, ctx, cleanup := setupNodeForTestingBatchSizeLimit(t, false)
	defer cleanup()

	checkReceiverAccountBalance(t, ctx, builder, 0)
	batchesProcessed := numberOfProcessedBatches(t, builder)

	bigBatch := buildBigBatch(t, builder.L2Info)
	postBatch(t, ctx, builder, bigBatch)

	ensureMoreBatchesWereProcessed(t, builder, batchesProcessed)
	checkReceiverAccountBalance(t, ctx, builder, 0)
}

func TestCanIncreaseBatchSizeLimit(t *testing.T) {
	builder, ctx, cleanup := setupNodeForTestingBatchSizeLimit(t, true)
	defer cleanup()

	checkReceiverAccountBalance(t, ctx, builder, 0)
	batchesProcessed := numberOfProcessedBatches(t, builder)

	bigBatch := buildBigBatch(t, builder.L2Info)
	postBatch(t, ctx, builder, bigBatch)

	ensureMoreBatchesWereProcessed(t, builder, batchesProcessed)
	checkReceiverAccountBalance(t, ctx, builder, TransferAmount)
}

// setupNodeForTestingBatchSizeLimit initializes a test node with the option to set a higher uncompressed batch size limit.
// Also, it creates genesis accounts for sender and receiver with appropriate balances.
// It returns the NodeBuilder and a cleanup function to be called after the test.
func setupNodeForTestingBatchSizeLimit(t *testing.T, setHighLimit bool) (*NodeBuilder, context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = false
	builder.L2Info.GenerateGenesisAccount(SenderAccount, big.NewInt(1e18))
	builder.L2Info.GenerateGenesisAccount(ReceiverAccount, big.NewInt(0))

	if setHighLimit {
		builder.chainConfig.ArbitrumChainParams.MaxUncompressedBatchSize = NewUncompressedSizeLimit
	}

	cleanup := builder.Build(t)

	return builder, ctx, func() {
		cancel()
		cleanup()
	}
}

// buildBigBatch builds a batch that:
// - consists of a valid transfer tx followed by highly compressible trash data
// - has an uncompressed size larger than DefaultMaxUncompressedBatchSize but less than NewUncompressedSizeLimit
// - has a compressed size smaller than the allowed calldata batch size for the test batch poster
// - is already compressed and has the appropriate header byte
func buildBigBatch(t *testing.T, l2Info *BlockchainTestInfo) []byte {
	batchBuffer := bytes.NewBuffer([]byte{})

	// 1. The first tx in the batch is a standard transfer tx used as an indicator whether the batch was processed successfully.
	standardTx := l2Info.PrepareTx(SenderAccount, ReceiverAccount, 1000000, big.NewInt(TransferAmount), []byte{})
	err := writeTxToBatch(batchBuffer, standardTx)
	Require(t, err)

	// 2. The rest of the batch is filled with highly compressible trash data.
	batchBuffer.Write(bytes.Repeat([]byte{0xff}, params.DefaultMaxUncompressedBatchSize))

	// 3. Compress the batch (as the batch poster would do).
	compressed, err := arbcompress.CompressWell(batchBuffer.Bytes())
	Require(t, err)

	// 4. Ensure compressed and uncompressed sizes are as expected.
	uncompressedSize, compressedSize := len(batchBuffer.Bytes()), len(compressed)
	require.Greater(t, uncompressedSize, params.DefaultMaxUncompressedBatchSize)
	require.Less(t, uncompressedSize, NewUncompressedSizeLimit)
	require.Less(t, compressedSize, arbnode.TestBatchPosterConfig.MaxCalldataBatchSize)

	// 5. Return the compressed batch with the appropriate header byte.
	return append([]byte{daprovider.BrotliMessageHeaderByte}, compressed...)
}

// postBatch posts the given batch directly to the L1 SequencerInbox contract.
func postBatch(t *testing.T, ctx context.Context, builder *NodeBuilder, batch []byte) {
	seqNum := new(big.Int).Lsh(common.Big1, 256)
	seqNum.Sub(seqNum, common.Big1)

	seqInboxAddr := builder.L1Info.GetAddress("SequencerInbox")
	seqInbox, err := bridgegen.NewSequencerInbox(seqInboxAddr, builder.L1.Client)
	Require(t, err)

	sequencer := builder.L1Info.GetDefaultTransactOpts("Sequencer", ctx)

	tx, err := seqInbox.AddSequencerL2BatchFromOrigin8f111f3c(&sequencer, seqNum, batch, big.NewInt(1), common.Address{}, big.NewInt(0), big.NewInt(0))
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L1.Client, tx)
	Require(t, err)
}

// checkReceiverAccountBalance ensures that the receiver account has the expected balance.
func checkReceiverAccountBalance(t *testing.T, ctx context.Context, builder *NodeBuilder, expectedBalance int64) {
	balanceBefore, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress(ReceiverAccount), nil)
	Require(t, err)
	require.True(t, balanceBefore.Cmp(big.NewInt(expectedBalance)) == 0)
}

// numberOfProcessedBatches retrieves the number of batches processed by the L2 node's inbox tracker.
func numberOfProcessedBatches(t *testing.T, builder *NodeBuilder) uint64 {
	batchesProcessed, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)
	return batchesProcessed
}

// ensureMoreBatchesWereProcessed waits until the number of processed batches exceeds the given earlier count.
func ensureMoreBatchesWereProcessed(t *testing.T, builder *NodeBuilder, processedEarlier uint64) {
	require.Eventuallyf(t, func() bool {
		return numberOfProcessedBatches(t, builder) > processedEarlier
	}, 5*time.Second, time.Second, "new batch processed")
}
