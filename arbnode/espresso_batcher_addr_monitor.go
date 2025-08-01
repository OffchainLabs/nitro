package arbnode

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/bold/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/dbutil"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

const (
	eventKey               = "espresso-batcher-addr-event"
	initAddressesKey       = "espresso-batcher-addr-init-addresses"
	lastProcessedHeightKey = "espresso-last-processed-height"
)

var ownerFunctionCalledID common.Hash
var seqInboxABI abi.ABI

func init() {
	parsedSeqInboxABI, err := bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	seqInboxABI = *parsedSeqInboxABI
	ownerFunctionCalledID = parsedSeqInboxABI.Events["OwnerFunctionCalled"].ID
}

// BatcherAddrUpdate represents a batch poster address status change, equivalent in effect to the `BatchPosterSet` event.
// For compatibility, we do not directly search for `BatchPosterSet` events.
// Instead, we search for `OwnerFunctionCalled(1)` events and parse the transaction input data to reconstruct the updates.
type BatcherAddrUpdate struct {
	// From this L1 height, the batcher address becomes functional or non-functional
	L1Height     uint64         `koanf:"l1-height"`
	ParentHeight uint64         `koanf:"parent-height"`
	Addr         common.Address `koanf:"addr"`
	IsBatcher    bool           `koanf:"is-batcher"`
}

type BatcherAddrMonitor struct {
	stopwaiter.StopWaiter
	// This is corresponding L1 height to the parent height.
	// If the parent chain is Ethereum, this is equal to the parent height.
	lastProcessedL1Height     uint64
	lastEventL1Height         uint64
	lastProcessedParentHeight uint64

	// Cache for the latest valid addresses.
	// Since batcher address changes are infrequent and callers typically
	// process HotShot blocks sequentially, caching improves performance.
	cached          bool
	cachedAddresses []common.Address

	updates []BatcherAddrUpdate
	db      ethdb.Database

	// Init addresses are the addresses that were set as batcher when the rollup was deployed.
	initAddresses []common.Address

	l1Reader *headerreader.HeaderReader

	seqInboxAddr      common.Address
	seqInboxInterface *bridgegen.SequencerInbox
	deployAt          uint64
}

func NewBatcherAddrMonitor(
	initAddresses []common.Address,
	db ethdb.Database,
	l1Reader *headerreader.HeaderReader,
	seqInboxAddr common.Address,
	deployAt uint64,
	fromBlock uint64,
) *BatcherAddrMonitor {
	seqInboxInterface, err := bridgegen.NewSequencerInbox(seqInboxAddr, l1Reader.Client())
	if err != nil {
		panic(err)
	}
	if fromBlock < deployAt+1 {
		fromBlock = deployAt + 1
	}
	return &BatcherAddrMonitor{
		initAddresses:             initAddresses,
		db:                        db,
		l1Reader:                  l1Reader,
		seqInboxAddr:              seqInboxAddr,
		seqInboxInterface:         seqInboxInterface,
		deployAt:                  deployAt,
		lastProcessedParentHeight: fromBlock - 1,
	}
}

func (b *BatcherAddrMonitor) AddBatchPosterSetEvents(events []BatcherAddrUpdate) error {
	if len(events) == 0 {
		return nil
	}
	b.updates = append(b.updates, events...)
	// Sort events by l1Height to ensure correct processing order.
	// Since BatcherAddr events are infrequent, the performance impact of sorting is negligible.
	sort.Slice(b.updates, func(i, j int) bool {
		return b.updates[i].L1Height < b.updates[j].L1Height
	})
	b.lastEventL1Height = b.updates[len(b.updates)-1].L1Height
	b.cached = false
	return b.Store()
}

func (b *BatcherAddrMonitor) GetValidAddresses(targetL1Height uint64) []common.Address {
	if targetL1Height > b.lastProcessedL1Height {
		// If the target L1 height is greater than the latest known L1 height,
		// return an empty slice. The caller should wait until the monitor has
		// observed at least this L1 height before calling this function.
		return []common.Address{}
	}

	if len(b.updates) == 0 || b.updates[0].L1Height > targetL1Height {
		return b.initAddresses
	}

	// If the target L1 height is within the latest cached window, return the cached result.
	// In a practical scenario, this is the most common case. Here means that during the time
	// from `lastEventL1Height` to `l1Height`, the `events` are not changed. It is not needed to
	// calculate valid batcher addresses.
	latestCachedWindow := targetL1Height >= b.lastEventL1Height && targetL1Height <= b.lastProcessedL1Height
	if b.cached && latestCachedWindow {
		return b.cachedAddresses
	}

	result := map[common.Address]bool{}
	for _, addr := range b.initAddresses {
		result[addr] = true
	}

	for _, event := range b.updates {
		if event.L1Height > targetL1Height {
			break
		}

		result[event.Addr] = event.IsBatcher
	}

	var validAddrs []common.Address
	for addr, isBatcher := range result {
		if isBatcher {
			validAddrs = append(validAddrs, addr)
		}
	}

	if latestCachedWindow {
		b.cached = true
		b.cachedAddresses = validAddrs
	}

	return validAddrs
}

func (b *BatcherAddrMonitor) SetParentHeight(height uint64) {
	b.lastProcessedParentHeight = height
}

func (b *BatcherAddrMonitor) SetL1Height(height uint64) {
	b.lastProcessedL1Height = height
}

func (b *BatcherAddrMonitor) GetLastProcessedParentHeight() uint64 {
	return b.lastProcessedParentHeight
}

func (b *BatcherAddrMonitor) LookupAddressUpdates(ctx context.Context, fromBlock, toBlock uint64) ([]BatcherAddrUpdate, error) {
	from := big.NewInt(0).SetUint64(fromBlock)
	to := big.NewInt(0).SetUint64(toBlock)
	query := ethereum.FilterQuery{
		BlockHash: nil,
		FromBlock: from,
		ToBlock:   to,
		Addresses: []common.Address{b.seqInboxAddr},
		Topics: [][]common.Hash{
			{ownerFunctionCalledID},
			{common.BigToHash(big.NewInt(1))},
		},
	}
	logs, err := b.l1Reader.Client().FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	return b.logsToBatcherAddrEvents(ctx, logs)
}

func (b *BatcherAddrMonitor) logsToBatcherAddrEvents(ctx context.Context, logs []types.Log) ([]BatcherAddrUpdate, error) {
	if len(logs) == 0 {
		return nil, nil
	}
	events := []BatcherAddrUpdate{}
	for _, ethLog := range logs {
		l1Height := ethLog.BlockNumber
		if b.l1Reader.IsParentChainArbitrum() {
			header, err := b.l1Reader.Client().HeaderByNumber(ctx, big.NewInt(0).SetUint64(ethLog.BlockNumber))
			if err != nil {
				return nil, err
			}
			l1Height = types.DeserializeHeaderExtraInformation(header).L1BlockNumber
		}
		txHash := ethLog.TxHash
		tx, _, err := b.l1Reader.Client().TransactionByHash(ctx, txHash)
		if err != nil {
			return nil, err
		}
		// Parse data to get arguments from tx
		data := tx.Data()
		if len(data) < 4 {
			return nil, fmt.Errorf("failed to parse a log: invalid data")
		}
		if !bytes.Equal(data[:4], seqInboxABI.Methods["setIsBatchPoster"].ID) {
			// Encountering an unknown method, caff node needs to update
			return nil, fmt.Errorf("failed to parse a log: invalid method: %x", data[:4])
		}
		args, err := seqInboxABI.Methods["setIsBatchPoster"].Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		batchPoster, ok := args[0].(common.Address)
		if !ok {
			return nil, fmt.Errorf("failed to parse a log: invalid batch poster address")
		}
		isBatcher, ok := args[1].(bool)
		if !ok {
			return nil, fmt.Errorf("failed to parse a log: invalid isBatchPoster")
		}

		event := BatcherAddrUpdate{
			Addr:         batchPoster,
			IsBatcher:    isBatcher,
			L1Height:     l1Height,
			ParentHeight: ethLog.BlockNumber,
		}
		log.Info("adding event for batch poster updates", "event", event)
		events = append(events, event)
	}
	return events, nil
}

func (b *BatcherAddrMonitor) StoreLastProcessedHeight(batch ethdb.Batch, height uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	return batch.Put([]byte(lastProcessedHeightKey), buf)
}

func (b *BatcherAddrMonitor) Store() error {
	eventsBytes, err := rlp.EncodeToBytes(b.updates)
	if err != nil {
		return fmt.Errorf("failed to encode events: %w", err)
	}

	initAddressesBytes, err := rlp.EncodeToBytes(b.initAddresses)
	if err != nil {
		return fmt.Errorf("failed to encode init addresses: %w", err)
	}

	newBatch := b.db.NewBatch()

	err = newBatch.Put([]byte(eventKey), eventsBytes)
	if err != nil {
		return fmt.Errorf("failed to put events: %w", err)
	}

	err = newBatch.Put([]byte(initAddressesKey), initAddressesBytes)
	if err != nil {
		return fmt.Errorf("failed to put init addresses: %w", err)
	}

	err = b.StoreLastProcessedHeight(newBatch, b.lastProcessedParentHeight)
	if err != nil {
		return fmt.Errorf("failed to put last processed height: %w", err)
	}

	return newBatch.Write()
}

func (b *BatcherAddrMonitor) Restore() error {
	initAddressesBytes, err := b.db.Get([]byte(initAddressesKey))
	if err != nil && !dbutil.IsErrNotFound(err) {
		return fmt.Errorf("failed to get init addresses: %w", err)
	}

	if initAddressesBytes != nil {
		err = rlp.DecodeBytes(initAddressesBytes, &b.initAddresses)
		if err != nil {
			return fmt.Errorf("failed to decode init addresses: %w", err)
		}
	}

	lastProcessedHeightBytes, err := b.db.Get([]byte(lastProcessedHeightKey))
	if err != nil && !dbutil.IsErrNotFound(err) {
		return fmt.Errorf("failed to get last processed height: %w", err)
	}
	if lastProcessedHeightBytes != nil {
		b.lastProcessedParentHeight = binary.BigEndian.Uint64(lastProcessedHeightBytes)
	}

	eventsBytes, err := b.db.Get([]byte(eventKey))
	if err != nil && !dbutil.IsErrNotFound(err) {
		return fmt.Errorf("failed to get events: %w", err)
	}

	if eventsBytes != nil {
		var events []BatcherAddrUpdate
		err = rlp.DecodeBytes(eventsBytes, &events)
		if err != nil {
			return fmt.Errorf("failed to decode events: %w", err)
		}
		b.updates = events
		b.cached = false
		b.cachedAddresses = []common.Address{}
		if len(events) > 0 {
			b.lastEventL1Height = events[len(events)-1].L1Height
		}
	} else {
		b.updates = []BatcherAddrUpdate{}
		b.cached = false
		b.cachedAddresses = []common.Address{}
		b.lastEventL1Height = 0
	}
	return nil
}

func (b *BatcherAddrMonitor) backfill(ctx context.Context) error {
	latestParentHeader, err := b.l1Reader.Client().HeaderByNumber(ctx, new(big.Int).SetInt64(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return fmt.Errorf("failed to get latest parent height: %w", err)
	}
	lastProcessedHeight := b.GetLastProcessedParentHeight()

	if lastProcessedHeight <= b.deployAt {
		// Verify init addresses are batchers
		for _, addr := range b.initAddresses {
			isBatcher, err := b.seqInboxInterface.IsBatchPoster(&bind.CallOpts{}, addr)
			if err != nil {
				return fmt.Errorf("failed to get batcher status: %w", err)
			}
			if !isBatcher {
				return fmt.Errorf("init address %s is not a batcher", addr)
			}
		}

		lastProcessedHeight = b.deployAt
		b.lastProcessedParentHeight = lastProcessedHeight
	}

	blocksToRead := uint64(100)
	allowedRetry := 10
	retry := 0
	latestParentHeight := latestParentHeader.Number.Uint64()
	log.Info("batcher addr monitor backfilling")
	for retry < allowedRetry {
		if lastProcessedHeight >= latestParentHeight {
			break
		}

		events, err := b.LookupAddressUpdates(ctx, lastProcessedHeight+1, lastProcessedHeight+blocksToRead)
		if err != nil {
			retry++
			log.Error("failed to lookup events", "err", err)
			continue
		}
		err = b.AddBatchPosterSetEvents(events)
		if err != nil {
			retry++
			log.Error("failed to add events", "err", err)
			continue
		}
		lastProcessedHeight += blocksToRead
		latestParentHeader, err = b.l1Reader.Client().HeaderByNumber(ctx, new(big.Int).SetInt64(int64(rpc.FinalizedBlockNumber)))
		if err != nil {
			retry++
			log.Error("failed to get latest parent height", "err", err)
			continue
		}
		latestParentHeight = latestParentHeader.Number.Uint64()
	}
	b.lastProcessedParentHeight = latestParentHeight
	b.lastProcessedL1Height = latestParentHeight
	if b.l1Reader.IsParentChainArbitrum() {
		b.lastProcessedL1Height = types.DeserializeHeaderExtraInformation(latestParentHeader).L1BlockNumber
	}
	log.Info("batcher addr monitor backfilled", "parentHeight", b.lastProcessedParentHeight, "l1Height", b.lastProcessedL1Height)

	return nil
}

func (b *BatcherAddrMonitor) Process(ctx context.Context) error {
	latestHeader, err := b.l1Reader.LastHeader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block header: %w", err)
	}
	latestBlockNumber := latestHeader.Number.Uint64()
	parentHeight := b.GetLastProcessedParentHeight()
	// The latest finalized block doesn't change
	if parentHeight >= latestBlockNumber {
		log.Debug("processing", "parentHeight", parentHeight, "latestBlockNumber", latestBlockNumber)
		return nil
	}

	newHeight := latestBlockNumber
	events, err := b.LookupAddressUpdates(ctx, parentHeight+1, newHeight)
	log.Debug("looking up events", "from", parentHeight+1, "to", newHeight)
	if err != nil {
		return err
	}
	err = b.AddBatchPosterSetEvents(events)
	if err != nil {
		return err
	}
	l1Height := newHeight
	if b.l1Reader.IsParentChainArbitrum() {
		l1Height = types.DeserializeHeaderExtraInformation(latestHeader).L1BlockNumber
	}
	b.SetL1Height(l1Height)
	b.SetParentHeight(newHeight)
	if len(events) == 0 {
		// If no events are found, we still need to update the last processed height
		batch := b.db.NewBatch()
		err = b.StoreLastProcessedHeight(batch, newHeight)
		if err != nil {
			return fmt.Errorf("failed to store last processed height: %w", err)
		}
		return batch.Write()
	}
	return nil
}

func (b *BatcherAddrMonitor) GetEvents() []BatcherAddrUpdate {
	return b.updates
}

func (b *BatcherAddrMonitor) Start(ctx context.Context) error {
	log.Info("starting the batch poster address monitor")
	b.StopWaiter.Start(ctx, b)

	err := b.Restore()
	if err != nil && !dbutil.IsErrNotFound(err) {
		return fmt.Errorf("failed to restore batcher address monitor: %w", err)
	}

	err = b.backfill(ctx)
	if err != nil {
		return fmt.Errorf("failed to backfill batcher address monitor: %w", err)
	}

	headerchan, unsubscribe := b.l1Reader.Subscribe(false)

	b.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				unsubscribe()
				return
			case <-headerchan:
				err := b.Process(ctx)
				if err != nil {
					log.Error("failed to process", "err", err)
					continue
				}
			}
		}
	})

	return nil
}
