package staker

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbnode/mel/extraction"
	"github.com/offchainlabs/nitro/arbnode/mel/runner"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/daprovider"
	"github.com/offchainlabs/nitro/solgen/go/rollupgen"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// dummyTxsAndLogsFetcher is for testing purposes. TODO: remove once we have preimages recorder implementations
type DummyTxsAndLogsFetcher struct {
	L1client *ethclient.Client
	receipts types.Receipts
}

func (d *DummyTxsAndLogsFetcher) LogsForBlockHash(ctx context.Context, parentChainBlockHash common.Hash) ([]*types.Log, error) {
	receipts, err := d.L1client.BlockReceipts(ctx, rpc.BlockNumberOrHashWithHash(parentChainBlockHash, false))
	if err != nil {
		return nil, err
	}
	var logs []*types.Log
	for _, receipt := range receipts {
		logs = append(logs, receipt.Logs...)
	}
	d.receipts = receipts
	return logs, nil
}

func (d *DummyTxsAndLogsFetcher) LogsForTxIndex(ctx context.Context, parentChainBlockHash common.Hash, txIndex uint) ([]*types.Log, error) {
	// #nosec G115
	if d.receipts.Len() < int(txIndex+1) {
		return nil, fmt.Errorf("insufficient number of receipts: %d, txIndex: %d", d.receipts.Len(), txIndex)
	}
	receipt := d.receipts[txIndex]
	return receipt.Logs, nil
}

func (d *DummyTxsAndLogsFetcher) TransactionByLog(ctx context.Context, log *types.Log) (*types.Transaction, error) {
	tx, _, err := d.L1client.TransactionByHash(ctx, log.TxHash)
	return tx, err
}

type MELValidator struct {
	stopwaiter.StopWaiter

	arbDb    ethdb.KeyValueStore
	l1client *ethclient.Client

	boldStakerAddr common.Address
	rollupAddr     common.Address
	rollup         *rollupgen.RollupUserLogic

	messageExtractor *melrunner.MessageExtractor
	dapReaders       arbstate.DapReaderSource

	lastValidatedParentChainBlock uint64
}

func NewMELValidator(arbDb ethdb.KeyValueStore, l1client *ethclient.Client, messageExtractor *melrunner.MessageExtractor, dapReaders arbstate.DapReaderSource) *MELValidator {
	return &MELValidator{
		arbDb:            arbDb,
		l1client:         l1client,
		messageExtractor: messageExtractor,
		dapReaders:       dapReaders,
	}
}

func (mv *MELValidator) Start(ctx context.Context) {
	mv.CallIteratively(func(ctx context.Context) time.Duration {
		latestStaked, err := mv.rollup.LatestStakedAssertion(&bind.CallOpts{}, mv.boldStakerAddr)
		if err != nil {
			log.Error("MEL validator: Error fetching latest staked assertion hash", "err", err)
			return 0
		}
		latestStakedAssertion, err := ReadBoldAssertionCreationInfo(ctx, mv.rollup, mv.l1client, mv.rollupAddr, latestStaked)
		if err != nil {
			log.Error("MEL validator: Error fetching latest staked assertion creation info", "err", err)
			return 0
		}
		if latestStakedAssertion.InboxMaxCount == nil || !latestStakedAssertion.InboxMaxCount.IsUint64() {
			log.Error("MEL validator: latestStakedAssertion.InboxMaxCount is not uint64")
			return 0
		}

		// Create validation entry
		entry, err := mv.CreateNextValidationEntry(ctx, mv.lastValidatedParentChainBlock, latestStakedAssertion.InboxMaxCount.Uint64())
		if err != nil {
			log.Error("MEL validator: Error creating validation entry", "lastValidatedParentChainBlock", mv.lastValidatedParentChainBlock, "inboxMaxCount", latestStakedAssertion.InboxMaxCount.Uint64(), "err", err)
			return time.Minute // wait for latestStakedAssertion to progress by the blockValidator
		}

		// Send validation entry to validation nodes
		if err := mv.SendValidationEntry(ctx, entry); err != nil {
			log.Error("MEL validator: Error sending validation entry", "err", err)
		}

		// Advance validations
		return 0
	})
}

func (mv *MELValidator) CreateNextValidationEntry(ctx context.Context, lastValidatedParentChainBlock, toValidateMsgExtractionCount uint64) (*validationEntry, error) {
	if lastValidatedParentChainBlock == 0 { // TODO: last validated.
		// ending position- bold staker latest posted assertion on chain that it agrees with (l1blockhash)-
		return nil, errors.New("trying to create validation entry for zero block number")
	}
	preState, err := mv.messageExtractor.GetState(ctx, lastValidatedParentChainBlock)
	if err != nil {
		return nil, err
	}
	// We have already validated message extraction of messages till count toValidateMsgExtractionCount, so can return early
	// and wait for block validator to progress the toValidateMsgExtractionCount
	if preState.MsgCount >= toValidateMsgExtractionCount {
		return nil, nil
	}
	delayedMsgRecordingDB := melrunner.NewRecordingDatabase(mv.arbDb)
	recordingDAPReaders := melrunner.NewRecordingDAPReaderSource(ctx, mv.dapReaders)
	for i := lastValidatedParentChainBlock + 1; ; i++ {
		header, err := mv.l1client.HeaderByNumber(ctx, new(big.Int).SetUint64(i))
		if err != nil {
			return nil, err
		}
		// Awaiting recording implementations of logsFetcher and txsFetcher
		txsAndLogsFetcher := &DummyTxsAndLogsFetcher{L1client: mv.l1client}
		state, _, _, _, err := melextraction.ExtractMessages(ctx, preState, header, recordingDAPReaders, delayedMsgRecordingDB, txsAndLogsFetcher, txsAndLogsFetcher)
		if err != nil {
			return nil, fmt.Errorf("error calling melextraction.ExtractMessages in recording mode: %w", err)
		}
		wantState, err := mv.messageExtractor.GetState(ctx, i)
		if err != nil {
			return nil, err
		}
		if state.Hash() != wantState.Hash() {
			return nil, fmt.Errorf("calculated MEL state hash in recording mode doesn't match the one computed in native mode, parentchainBlocknumber: %d", i)
		}
		if state.MsgCount >= toValidateMsgExtractionCount {
			break
		}
		preState = state
	}
	preimages := recordingDAPReaders.Preimages()
	delayedPreimages := daprovider.PreimagesMap{
		arbutil.Keccak256PreimageType: delayedMsgRecordingDB.Preimages(),
	}
	daprovider.CopyPreimagesInto(preimages, delayedPreimages)
	return &validationEntry{
		Preimages: preimages,
	}, nil
}

func (mv *MELValidator) SendValidationEntry(ctx context.Context, entry *validationEntry) error {
	return nil
}
