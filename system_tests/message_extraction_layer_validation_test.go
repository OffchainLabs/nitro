package arbtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/offchainlabs/nitro/arbnode/mel"
	melextraction "github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/staker"
)

func TestMELValidator_Recording_Preimages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.L2Info.GenerateAccount("User2")
	builder.nodeConfig.BatchPoster.Post4844Blobs = true
	builder.nodeConfig.BatchPoster.IgnoreBlobPrice = true
	builder.nodeConfig.BatchPoster.MaxDelay = time.Hour     // set high max-delay so we can test the delay buffer
	builder.nodeConfig.BatchPoster.PollInterval = time.Hour // set a high poll interval to avoid continuous polling
	cleanup := builder.Build(t)
	defer cleanup()

	// Post a blob batch with a bunch of txs
	startBlock, err := builder.L1.Client.BlockNumber(ctx)
	Require(t, err)
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{})
	defer cleanupB()
	initialBatchCount := GetBatchCount(t, builder)
	var txs types.Transactions
	for i := 0; i < 20; i++ {
		tx, _ := builder.L2.TransferBalance(t, "Faucet", "User2", big.NewInt(1e12), builder.L2Info)
		txs = append(txs, tx)
	}
	builder.nodeConfig.BatchPoster.MaxDelay = 0
	builder.L2.ConsensusConfigFetcher.Set(builder.nodeConfig)
	_, err = builder.L2.ConsensusNode.BatchPoster.MaybePostSequencerBatch(ctx)
	Require(t, err)
	for _, tx := range txs {
		_, err := testClientB.EnsureTxSucceeded(tx)
		Require(t, err, "tx not found on second node")
	}
	CheckBatchCount(t, builder, initialBatchCount+1)

	// Post delayed messages
	forceDelayedBatchPosting(t, ctx, builder, testClientB, 10, 0)

	// MEL Validator: create validation entry
	blobReaderRegistry := daprovider.NewDAProviderRegistry()
	Require(t, blobReaderRegistry.SetupBlobReader(daprovider.NewReaderForBlobReader(builder.L1.L1BlobReader)))
	melValidator := staker.NewMELValidator(builder.L2.ConsensusNode.ArbDB, builder.L1.Client, builder.L2.ConsensusNode.MessageExtractor, blobReaderRegistry)
	extractedMsgCount, err := builder.L2.ConsensusNode.TxStreamer.GetMessageCount()
	Require(t, err)
	entry, err := melValidator.CreateNextValidationEntry(ctx, startBlock, uint64(extractedMsgCount))
	Require(t, err)

	// Represents running of MEL validation using preimages in wasm mode. TODO: remove this once we have validation wired
	state, err := builder.L2.ConsensusNode.MessageExtractor.GetState(ctx, startBlock)
	Require(t, err)
	preimagesBasedDelayedDb := &delayedMessageDatabase{
		preimageResolver: &testPreimageResolver{
			preimages: entry.Preimages[arbutil.Keccak256PreimageType],
		},
	}
	preimagesBasedDapReaders := daprovider.NewDAProviderRegistry()
	Require(t, preimagesBasedDapReaders.SetupBlobReader(daprovider.NewReaderForBlobReader(&blobPreimageReader{entry.Preimages})))
	for state.MsgCount < uint64(extractedMsgCount) {
		header, err := builder.L1.Client.HeaderByNumber(ctx, new(big.Int).SetUint64(state.ParentChainBlockNumber+1))
		Require(t, err)
		// Awaiting recording implementations of logsFetcher and txsFetcher
		txsAndLogsFetcher := &staker.DummyTxsAndLogsFetcher{L1client: builder.L1.Client}
		postState, _, _, _, err := melextraction.ExtractMessages(ctx, state, header, preimagesBasedDapReaders, preimagesBasedDelayedDb, txsAndLogsFetcher, txsAndLogsFetcher)
		Require(t, err)
		wantState, err := builder.L2.ConsensusNode.MessageExtractor.GetState(ctx, state.ParentChainBlockNumber+1)
		Require(t, err)
		if postState.Hash() != wantState.Hash() {
			t.Fatalf("MEL state mismatch")
		}
		state = postState
	}
}

// TODO: Code from cmd/mel-replay and cmd/replay packages for verification of preimages, should be deleted once we have validation wired
type blobPreimageReader struct {
	preimages daprovider.PreimagesMap
}

func (r *blobPreimageReader) Initialize(ctx context.Context) error { return nil }

func (r *blobPreimageReader) GetBlobs(
	ctx context.Context,
	batchBlockHash common.Hash,
	versionedHashes []common.Hash,
) ([]kzg4844.Blob, error) {
	var blobs []kzg4844.Blob
	for _, h := range versionedHashes {
		var blob kzg4844.Blob
		if _, ok := r.preimages[arbutil.EthVersionedHashPreimageType]; !ok {
			return nil, errors.New("no blobs found in preimages")
		}
		preimage, ok := r.preimages[arbutil.EthVersionedHashPreimageType][h]
		if !ok {
			return nil, errors.New("no blobs found in preimages")
		}
		if len(preimage) != len(blob) {
			return nil, fmt.Errorf("for blob %v got back preimage of length %v but expected blob length %v", h, len(preimage), len(blob))
		}
		copy(blob[:], preimage)
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

type testPreimageResolver struct {
	preimages map[common.Hash][]byte
}

func (r *testPreimageResolver) ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error) {
	if preimageType != arbutil.Keccak256PreimageType {
		return nil, fmt.Errorf("unsupported preimageType: %d", preimageType)
	}
	if preimage, ok := r.preimages[hash]; ok {
		return preimage, nil
	}
	return nil, fmt.Errorf("preimage not found for hash: %v", hash)
}

type preimageResolver interface {
	ResolveTypedPreimage(preimageType arbutil.PreimageType, hash common.Hash) ([]byte, error)
}

type delayedMessageDatabase struct {
	preimageResolver preimageResolver
}

func (d *delayedMessageDatabase) ReadDelayedMessage(
	ctx context.Context,
	state *mel.State,
	msgIndex uint64,
) (*mel.DelayedInboxMessage, error) {
	originalMsgIndex := msgIndex
	totalMsgsSeen := state.DelayedMessagesSeen
	if msgIndex >= totalMsgsSeen {
		return nil, fmt.Errorf("index %d out of range, total delayed messages seen: %d", msgIndex, totalMsgsSeen)
	}
	treeSize := nextPowerOfTwo(totalMsgsSeen)
	merkleDepth := bits.TrailingZeros64(treeSize)

	// Start traversal from root, which is the delayed messages seen root.
	merkleRoot := state.DelayedMessagesSeenRoot
	currentHash := merkleRoot
	currentDepth := merkleDepth

	// Traverse down the Merkle tree to find the leaf at the given index.
	for currentDepth > 0 {
		// Resolve the preimage to get left and right children.
		result, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
		if err != nil {
			return nil, err
		}
		if len(result) != 64 {
			return nil, fmt.Errorf("invalid preimage result length: %d, wanted 64", len(result))
		}
		// Split result into left and right halves.
		mid := len(result) / 2
		left := result[:mid]
		right := result[mid:]

		// Calculate which subtree contains our index.
		subtreeSize := uint64(1) << (currentDepth - 1)
		if msgIndex < subtreeSize {
			// Go left.
			currentHash = common.BytesToHash(left)
		} else {
			// Go right.
			currentHash = common.BytesToHash(right)
			msgIndex -= subtreeSize
		}
		currentDepth--
	}
	// At this point, currentHash should be the hash of the delayed message.
	delayedMsgBytes, err := d.preimageResolver.ResolveTypedPreimage(arbutil.Keccak256PreimageType, currentHash)
	if err != nil {
		return nil, err
	}
	delayedMessage := new(mel.DelayedInboxMessage)
	if err = rlp.Decode(bytes.NewBuffer(delayedMsgBytes), &delayedMessage); err != nil {
		return nil, fmt.Errorf("failed to decode delayed message at index %d: %w", originalMsgIndex, err)
	}
	return delayedMessage, nil
}

func nextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	if n&(n-1) == 0 {
		return n
	}
	return 1 << bits.Len64(n)
}
