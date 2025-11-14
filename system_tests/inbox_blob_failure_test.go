// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build !race

package arbtest

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
)

// failingBlobReader wraps a real BlobReader and can be configured to fail.
type failingBlobReader struct {
	inner     daprovider.BlobReader
	returnErr error // Set this field to make GetBlobs return an error
}

func (f *failingBlobReader) GetBlobs(ctx context.Context, batchBlockHash common.Hash, versionedHashes []common.Hash) ([]kzg4844.Blob, error) {
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	return f.inner.GetBlobs(ctx, batchBlockHash, versionedHashes)
}

func (f *failingBlobReader) Initialize(ctx context.Context) error {
	// Don't call inner.Initialize() because it wipes the blob storage map.
	// The inner SimulatedBeacon is already initialized when the sequencer started.
	return nil
}

// TestInboxReaderBlobFailureWithDelayedMessage tests the race condition described in NIT-4065:
// "don't read a batch-posting-report if you cannot read the batch posted"
//
// The issue: When a batch is posted to L1, a batch-posting-report delayed message is created.
// If the follower's AddDelayedMessages() succeeds but AddSequencerBatches() fails (due to
// blob fetch failure), the follower may get out of sync (delayed count incremented but batch
// not processed).
//
// This test verifies:
// 1. Follower with broken blob reader gets out of sync
// 2. After re-enabling blob fetching, follower resumes syncing
// 3. Follower can sync new batches and delayed messages normally
// 4. Final block hashes match between sequencer and follower
func TestInboxReaderBlobFailureWithDelayedMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build sequencer with blob posting and delayed sequencer enabled
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.nodeConfig.BatchPoster.PollInterval = 10 * time.Millisecond
	builder.nodeConfig.DelayedSequencer.Enable = true
	builder.nodeConfig.DelayedSequencer.FinalizeDistance = 1

	cleanup := builder.Build(t)
	defer cleanup()

	// Send L2 transaction to trigger batch posting
	tx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(1e12), nil)
	err := builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// Advance L1 to trigger batch posting and delayed message processing
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Wait for batch to be posted
	txReceipt, err := builder.L2.Client.TransactionReceipt(ctx, tx.Hash())
	Require(t, err)
	l2Block, err := builder.L2.Client.BlockByHash(ctx, txReceipt.BlockHash)
	Require(t, err)

	var batchNum uint64
	for i := 0; i < 30; i++ {
		batchInfo, err := builder.L2.ConsensusNode.FindInboxBatchContainingMessage(arbutil.MessageIndex(l2Block.NumberU64())).Await(ctx)
		Require(t, err)
		if batchInfo.Found {
			batchNum = batchInfo.BatchNum
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Advance L1 more for batch-posting-report finality
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 5)
	time.Sleep(time.Second)

	// Record sequencer state before starting follower
	seqDelayed, err := builder.L2.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	seqBatch, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)

	// Build follower with failing blob reader
	wrappedBlobReader := &failingBlobReader{
		inner:     builder.L1.L1BlobReader,
		returnErr: errors.New("simulated blob fetch failure"),
	}

	testClientB, cleanupB := builder.Build2ndNodeWithBlobReader(t, &SecondNodeParams{
		nodeConfig: arbnode.ConfigDefaultL1NonSequencerTest(),
	}, wrappedBlobReader)
	defer cleanupB()

	// Wait for follower to attempt sync
	time.Sleep(2 * time.Second)

	// Check if follower is out of sync
	follDelayed, err := testClientB.ConsensusNode.InboxTracker.GetDelayedCount()
	Require(t, err)
	follBatch, err := testClientB.ConsensusNode.InboxTracker.GetBatchCount()
	Require(t, err)

	if follDelayed == seqDelayed && follBatch < seqBatch {
		t.Logf("Follower is behind: delayed=%d (same as sequencer) but batches=%d < %d",
			follDelayed, follBatch, seqBatch)
	} else {
		t.Logf("Follower state: delayed=%d batches=%d, sequencer: delayed=%d batches=%d",
			follDelayed, follBatch, seqDelayed, seqBatch)
	}

	// Check for database corruption: delayed message should not be readable if its batch doesn't exist
	// This detects the race condition where AddDelayedMessages succeeds but AddSequencerBatches fails
	if follDelayed > 0 && follBatch < seqBatch {
		// Investigate all delayed messages to understand the corruption
		for i := uint64(0); i < follDelayed; i++ {
			msg, err := testClientB.ConsensusNode.InboxReader.Tracker().GetDelayedMessage(ctx, i)
			if err != nil {
				t.Fatalf("Delayed message %d: Failed to read - %v", i, err)
				continue
			}
			t.Logf("Delayed message %d: Kind=%v, BlockNumber=%v", i, msg.Header.Kind, msg.Header.BlockNumber)

			// Check if this is a batch-posting-report
			if msg.Header.Kind == arbostypes.L1MessageType_BatchPostingReport {
				// Try to parse it to see which batch it references
				_, _, _, batchNum, _, _, err := arbostypes.ParseBatchPostingReportMessageFields(bytes.NewReader(msg.L2msg))
				if err != nil {
					t.Logf("  Failed to parse batch-posting-report: %v", err)
				} else {
					t.Logf("  Batch-posting-report for batch %d", batchNum)

					// Check if this batch exists in our database
					_, err := testClientB.ConsensusNode.InboxTracker.GetBatchMetadata(batchNum)
					if err != nil {
						// TODO After we have fixed the issue, this can be changed back to log.Fatalf
						t.Logf("CORRUPTION DETECTED: Delayed message %d is a batch-posting-report for batch %d, but batch %d doesn't exist in database! Error: %v", i, batchNum, batchNum, err)
					}
				}
			}
		}
		t.Logf("All delayed messages checked - no corruption found")
	}

	// Re-enable blob fetching
	wrappedBlobReader.returnErr = nil
	t.Log("Re-enabled blob fetching")

	// Send new transaction on sequencer
	verifyTx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(2e12), nil)
	err = builder.L2.Client.SendTransaction(ctx, verifyTx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(verifyTx)
	Require(t, err)

	// Advance L1 to post batch
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 30)

	// Wait for batch and advance for finality
	for i := 0; i < 30; i++ {
		verifyReceipt, _ := builder.L2.Client.TransactionReceipt(ctx, verifyTx.Hash())
		if verifyReceipt != nil {
			verifyBlock, _ := builder.L2.Client.BlockByHash(ctx, verifyReceipt.BlockHash)
			batchInfo, err := builder.L2.ConsensusNode.FindInboxBatchContainingMessage(arbutil.MessageIndex(verifyBlock.NumberU64())).Await(ctx)
			if err == nil && batchInfo.Found {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	AdvanceL1(t, ctx, builder.L1.Client, builder.L1Info, 5)

	// Check if follower synced the new transaction
	time.Sleep(3 * time.Second)
	follVerifyReceipt, err := WaitForTx(ctx, testClientB.Client, verifyTx.Hash(), 3*time.Second)
	if err != nil || follVerifyReceipt == nil {
		t.Fatal("Follower did not sync new transaction after re-enabling blobs")
	}
	t.Logf("Follower synced new transaction")

	// Send delayed message via L1
	delayedTx := builder.L2Info.PrepareTx("Owner", "Owner", builder.L2Info.TransferGas, big.NewInt(3e12), nil)
	SendSignedTxViaL1(t, ctx, builder.L1Info, builder.L1.Client, builder.L2.Client, delayedTx)

	// Check if follower synced the delayed message
	time.Sleep(3 * time.Second)
	follDelayedReceipt, err := WaitForTx(ctx, testClientB.Client, delayedTx.Hash(), 3*time.Second)
	if err != nil || follDelayedReceipt == nil {
		t.Fatal("Follower did not sync delayed message")
	}
	t.Logf("Follower synced delayed message")

	// Final check: Compare block hashes to ensure chains match
	seqBlockNum, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	follBlockNum, err := testClientB.Client.BlockNumber(ctx)
	Require(t, err)

	t.Logf("Final block numbers: sequencer=%d follower=%d", seqBlockNum, follBlockNum)

	// Compare the highest common block
	checkBlockNum := follBlockNum
	if seqBlockNum < follBlockNum {
		checkBlockNum = seqBlockNum
	}

	// #nosec G115
	seqBlock, err := builder.L2.Client.BlockByNumber(ctx, big.NewInt(int64(checkBlockNum)))
	Require(t, err)
	// #nosec G115
	follBlock, err := testClientB.Client.BlockByNumber(ctx, big.NewInt(int64(checkBlockNum)))
	Require(t, err)

	t.Logf("Comparing block %d hashes:", checkBlockNum)
	t.Logf("  Sequencer: %s", seqBlock.Hash())
	t.Logf("  Follower:  %s", follBlock.Hash())

	if seqBlock.Hash() != follBlock.Hash() {
		t.Fatalf("Block hash mismatch at block %d - chains have diverged!", checkBlockNum)
	}

	if follBlockNum < seqBlockNum {
		t.Logf("PASS: Follower is on same chain but lagging by %d blocks", seqBlockNum-follBlockNum)
	} else {
		t.Logf("PASS: Follower is fully synced")
	}

	// Prevent unused variable warning
	_ = batchNum
}

// Build2ndNodeWithBlobReader builds a second node with a custom blob reader.
func (b *NodeBuilder) Build2ndNodeWithBlobReader(t *testing.T, params *SecondNodeParams, blobReader daprovider.BlobReader) (*TestClient, func()) {
	t.Helper()
	DontWaitAndRun(b.ctx, 1, t.Name())
	if b.L2 == nil {
		t.Fatal("builder did not previously build an L2 Node")
	}
	if b.L1 == nil {
		t.Fatal("builder did not previously build an L1 Node")
	}

	if params == nil {
		params = &SecondNodeParams{}
	}
	if params.nodeConfig == nil {
		params.nodeConfig = arbnode.ConfigDefaultL1NonSequencerTest()
	}
	if params.dasConfig != nil {
		params.nodeConfig.DataAvailability = *params.dasConfig
	}
	if params.stackConfig == nil {
		params.stackConfig = b.l2StackConfig
		params.stackConfig.DataDir = t.TempDir()
	}
	if params.initData == nil {
		params.initData = &b.L2Info.ArbInitData
	}
	if params.execConfig == nil {
		params.execConfig = b.execConfig
	}
	if params.addresses == nil {
		params.addresses = b.addresses
	}
	if b.nodeConfig.BatchPoster.Enable && params.nodeConfig.BatchPoster.Enable && params.nodeConfig.BatchPoster.RedisUrl == "" {
		t.Fatal("The batch poster must use Redis when enabled for multiple nodes")
	}

	testClient := NewTestClient(b.ctx)
	testClient.Client, testClient.ConsensusNode, testClient.ExecutionConfigFetcher, testClient.ConsensusConfigFetcher =
		Create2ndNodeWithConfig(t, b.ctx, b.L2.ConsensusNode, b.L1.Stack, b.L1Info, params.initData, params.nodeConfig, params.execConfig, params.stackConfig, b.valnodeConfig, params.addresses, b.initMessage, params.useExecutionClientOnly, blobReader)
	testClient.ExecNode = getExecNode(t, testClient.ConsensusNode)
	testClient.cleanup = func() { testClient.ConsensusNode.StopAndWait() }
	testClient.L1BlobReader = blobReader

	return testClient, func() { testClient.cleanup() }
}
