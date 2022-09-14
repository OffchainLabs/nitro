// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/headerreader"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

var sequencerInboxABI *abi.ABI
var batchDeliveredID common.Hash
var addSequencerL2BatchFromOriginCallABI abi.Method
var sequencerBatchDataABI abi.Event

const sequencerBatchDataEvent = "SequencerBatchData"
const sequencerBatchDeliveredEvent = "SequencerBatchDelivered"

// TODO: can we use the generated ABI for BatchDataLocation enum?
type batchDataLocation uint8

const (
	batchDataTxInput batchDataLocation = iota
	batchDataSeparateEvent
)

func init() {
	var err error
	sequencerInboxABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerInboxABI.Events[sequencerBatchDeliveredEvent].ID
	sequencerBatchDataABI = sequencerInboxABI.Events[sequencerBatchDataEvent]
	addSequencerL2BatchFromOriginCallABI = sequencerInboxABI.Methods["addSequencerL2BatchFromOrigin"]
}

type SyncToStorageConfig struct {
	Eager                bool          `koanf:"eager"`
	EagerLowerBoundBlock uint64        `koanf:"eager-lower-bound-block"`
	RetentionPeriod      time.Duration `koanf:"retention-period"`
	DelayOnError         time.Duration `koanf:"delay-on-error"`
	IgnoreWriteErrors    bool          `koanf:"ignore-write-errors"`
	L1BlocksPerRead      uint64        `koanf:"l1-blocks-per-read"`
}

var DefaultSyncToStorageConfig = SyncToStorageConfig{
	Eager:                false,
	EagerLowerBoundBlock: 0,
	RetentionPeriod:      time.Duration(math.MaxInt64),
	DelayOnError:         time.Second,
	IgnoreWriteErrors:    true,
	L1BlocksPerRead:      100,
}

func SyncToStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".eager", DefaultSyncToStorageConfig.Eager, "eagerly sync batch data to this DAS's storage from the rest endpoints, using L1 as the index of batch data hashes; otherwise only sync lazily")
	f.Uint64(prefix+".eager-lower-bound-block", DefaultSyncToStorageConfig.EagerLowerBoundBlock, "when eagerly syncing, start indexing forward from this L1 block")
	f.Uint64(prefix+".l1-blocks-per-read", DefaultSyncToStorageConfig.L1BlocksPerRead, "when eagerly syncing, max l1 blocks to read per poll")
	f.Duration(prefix+".retention-period", DefaultSyncToStorageConfig.RetentionPeriod, "period to retain synced data (defaults to forever)")
	f.Duration(prefix+".delay-on-error", DefaultSyncToStorageConfig.DelayOnError, "time to wait if encountered an error before retrying")
	f.Bool(prefix+".ignore-write-errors", DefaultSyncToStorageConfig.IgnoreWriteErrors, "log only on failures to write when syncing; otherwise treat it as an error")
}

type l1SyncService struct {
	stopwaiter.StopWaiter

	config     SyncToStorageConfig
	syncTo     StorageService
	dataSource arbstate.DataAvailabilityReader

	l1Reader      *headerreader.HeaderReader
	inboxContract *bridgegen.SequencerInbox
	inboxAddr     common.Address

	catchingUp     bool
	lowBlockNr     uint64
	lastBatchCount *big.Int
	lastBatchAcc   common.Hash
}

func newl1SyncService(config *SyncToStorageConfig, syncTo StorageService, dataSource arbstate.DataAvailabilityReader, l1Reader *headerreader.HeaderReader, inboxAddr common.Address) (*l1SyncService, error) {
	l1Client := l1Reader.Client()
	inboxContract, err := bridgegen.NewSequencerInbox(inboxAddr, l1Client)
	if err != nil {
		return nil, err
	}
	// make sure that as we sync, any Keysets missing from dataSource will fetched from the L1 chain
	dataSource, err = NewChainFetchReader(dataSource, l1Client, inboxAddr)
	if err != nil {
		return nil, err
	}
	return &l1SyncService{
		config:         *config,
		syncTo:         syncTo,
		dataSource:     dataSource,
		l1Reader:       l1Reader,
		inboxContract:  inboxContract,
		inboxAddr:      inboxAddr,
		catchingUp:     true,
		lowBlockNr:     config.EagerLowerBoundBlock,
		lastBatchCount: big.NewInt(0),
	}, nil
}

func (s *l1SyncService) processBatchDelivered(ctx context.Context, batchDeliveredLog types.Log) error {
	data := []byte{}
	deliveredEvent, err := s.inboxContract.ParseSequencerBatchDelivered(batchDeliveredLog)
	if err != nil {
		return err
	}
	log.Info("BatchDelivered", "log", batchDeliveredLog, "event", deliveredEvent)
	storeUntil := arbmath.SaturatingUAdd(deliveredEvent.TimeBounds.MaxTimestamp, uint64(s.config.RetentionPeriod.Seconds()))
	if storeUntil < uint64(time.Now().Unix()) {
		// old batch - no need to store
		return nil
	}
	if deliveredEvent.DataLocation == uint8(batchDataSeparateEvent) {
		query := ethereum.FilterQuery{
			BlockHash: &batchDeliveredLog.BlockHash,
			Addresses: []common.Address{s.inboxAddr},
			Topics:    [][]common.Hash{{sequencerBatchDataABI.ID}, {common.BigToHash(deliveredEvent.BatchSequenceNumber)}},
		}
		logs, err := s.l1Reader.Client().FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		if len(logs) != 1 {
			return fmt.Errorf("found %d data logs for sequence 0x%x (expected 1)", len(logs), deliveredEvent.BatchSequenceNumber)
		}
		dataEvent, err := s.inboxContract.ParseSequencerBatchData(logs[0])
		if err != nil {
			return err
		}
		data = dataEvent.Data
	} else if deliveredEvent.DataLocation == uint8(batchDataTxInput) {
		tx, err := s.l1Reader.Client().TransactionInBlock(ctx, batchDeliveredLog.BlockHash, batchDeliveredLog.TxIndex)
		if err != nil {
			return err
		}
		args := make(map[string]interface{})
		err = addSequencerL2BatchFromOriginCallABI.Inputs.UnpackIntoMap(args, tx.Data()[4:])
		if err != nil {
			return err
		}
		var ok bool
		data, ok = args["data"].([]byte)
		if !ok {
			return fmt.Errorf("couldn't parse data for sequence 0x%x", deliveredEvent.BatchSequenceNumber)
		}
	}
	if len(data) < 1 {
		// no data - nothing to do
		log.Warn("BatchDelivered - no data found", "data", data)
		return nil
	}
	if !arbstate.IsDASMessageHeaderByte(data[0]) {
		log.Warn("BatchDelivered - data not DAS")
		return nil
	}

	header := make([]byte, 40)
	binary.BigEndian.PutUint64(header[:8], deliveredEvent.TimeBounds.MinTimestamp)
	binary.BigEndian.PutUint64(header[8:16], deliveredEvent.TimeBounds.MaxTimestamp)
	binary.BigEndian.PutUint64(header[16:24], deliveredEvent.TimeBounds.MinBlockNumber)
	binary.BigEndian.PutUint64(header[24:32], deliveredEvent.TimeBounds.MaxBlockNumber)
	binary.BigEndian.PutUint64(header[32:40], deliveredEvent.AfterDelayedMessagesRead.Uint64())

	data = append(header, data...)
	preimages := make(map[common.Hash][]byte)
	if _, err = arbstate.RecoverPayloadFromDasBatch(ctx, deliveredEvent.BatchSequenceNumber.Uint64(), data, s.dataSource, preimages, arbstate.KeysetValidate); err != nil {
		log.Error("recover payload failed", "txhash", batchDeliveredLog.TxHash, "data", data)
		return err
	}
	for hash, contents := range preimages {
		_, err := s.syncTo.GetByHash(ctx, hash)
		if errors.Is(err, ErrNotFound) {
			if err := s.syncTo.Put(ctx, contents, storeUntil); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	seqNumber := deliveredEvent.BatchSequenceNumber
	if seqNumber == nil {
		seqNumber = common.Big0
	}
	updatedBatchCount := new(big.Int).Add(seqNumber, common.Big1)
	if s.lastBatchCount.Cmp(updatedBatchCount) <= 0 {
		s.lastBatchCount.Set(seqNumber)
		s.lastBatchAcc = deliveredEvent.AfterAcc
	}
	return nil
}

func (s *l1SyncService) processBlockRange(ctx context.Context, lowerBound, higherBound uint64) error {
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(lowerBound),
		ToBlock:   new(big.Int).SetUint64(higherBound),
		Addresses: []common.Address{s.inboxAddr},
		Topics:    [][]common.Hash{{batchDeliveredID}},
	}
	logs, err := s.l1Reader.Client().FilterLogs(ctx, query)
	if err != nil {
		return err
	}
	for _, deliveredLog := range logs {
		if err := s.processBatchDelivered(ctx, deliveredLog); err != nil {
			return err
		}
	}
	return nil
}

func (s *l1SyncService) readMore(ctx context.Context) error {
	header, err := s.l1Reader.LastHeader(ctx)
	if err != nil {
		return err
	}
	highBlockNr := header.Number.Uint64()
	finalizedHighBlockNr := highBlockNr - 12 // TODO
	callOpts := &bind.CallOpts{
		Context:     ctx,
		BlockNumber: header.Number,
	}
	if s.lastBatchCount != nil {
		currentBatchCount, err := s.inboxContract.BatchCount(callOpts)
		if err != nil {
			return err
		}
		if currentBatchCount.Cmp(s.lastBatchCount) == 0 {
			accBytes, err := s.inboxContract.InboxAccs(callOpts, new(big.Int).Sub(currentBatchCount, common.Big1))
			if err != nil {
				return err
			}
			var lastAccHash common.Hash
			copy(lastAccHash[:], accBytes[:])
			if lastAccHash == s.lastBatchAcc {
				// we're up to date
				s.lowBlockNr = finalizedHighBlockNr
				s.catchingUp = false
				return nil
			}
		}
	}
	if highBlockNr > s.lowBlockNr+s.config.L1BlocksPerRead {
		s.catchingUp = true
		highBlockNr = s.lowBlockNr + s.config.L1BlocksPerRead
		if finalizedHighBlockNr > highBlockNr {
			finalizedHighBlockNr = highBlockNr
		}
	} else {
		s.catchingUp = false
	}
	err = s.processBlockRange(ctx, s.lowBlockNr, highBlockNr)
	if err != nil {
		return err
	}
	s.lowBlockNr = finalizedHighBlockNr + 1
	return nil
}

func (s *l1SyncService) mainThread(ctx context.Context) {
	headerChan, unsubscribe := s.l1Reader.Subscribe(false)
	defer unsubscribe()
	for {
		err := s.readMore(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error("error trying to sync from L1", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.config.DelayOnError):
			}
			continue
		}
		if s.catchingUp {
			// we're behind. Don't wait.
			continue
		}
		select {
		case <-headerChan:
		case <-ctx.Done():
			return
		}
	}
}

func (s *l1SyncService) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn, s)

	s.LaunchThread(s.mainThread)
}

type SyncingFallbackStorageService struct {
	FallbackStorageService

	syncService *l1SyncService
}

func NewSyncingFallbackStorageService(ctx context.Context,
	primary StorageService,
	backup arbstate.DataAvailabilityReader,
	l1Reader *headerreader.HeaderReader,
	inboxAddr common.Address,
	syncConf *SyncToStorageConfig) (*SyncingFallbackStorageService, error) {
	syncService, err := newl1SyncService(syncConf, primary, backup, l1Reader, inboxAddr)
	if err != nil {
		return nil, err
	}
	syncService.Start(ctx)
	return &SyncingFallbackStorageService{
		FallbackStorageService{
			primary,
			backup,
			uint64(syncConf.RetentionPeriod.Seconds()),
			syncConf.IgnoreWriteErrors,
			true,
			make(map[[32]byte]bool),
			sync.RWMutex{},
		},
		syncService,
	}, nil
}

func (s *SyncingFallbackStorageService) Close(ctx context.Context) error {
	s.syncService.StopOnly()
	s.FallbackStorageService.Close(ctx)
	return nil
}
