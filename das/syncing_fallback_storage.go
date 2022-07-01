package das

import (
	"context"
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

var sequencerBridgeABI *abi.ABI
var batchDeliveredID common.Hash
var addSequencerL2BatchFromOriginCallABI abi.Method
var sequencerBatchDataABI abi.Event

const sequencerBatchDataEvent = "SequencerBatchData"

// TODO: can we use the generated ABI for BatchDataLocation enum?
type batchDataLocation uint8

const (
	batchDataTxInput batchDataLocation = iota
	batchDataSeparateEvent
)

func init() {
	var err error
	sequencerBridgeABI, err = bridgegen.SequencerInboxMetaData.GetAbi()
	if err != nil {
		panic(err)
	}
	batchDeliveredID = sequencerBridgeABI.Events["SequencerBatchDelivered"].ID
	sequencerBatchDataABI = sequencerBridgeABI.Events[sequencerBatchDataEvent]
	addSequencerL2BatchFromOriginCallABI = sequencerBridgeABI.Methods["addSequencerL2BatchFromOrigin"]
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
		lastBatchCount: nil,
	}, nil
}

func (s *l1SyncService) processBatchDelivered(ctx context.Context, log *types.Log) error {
	data := []byte{}
	deliveredEvent := new(bridgegen.SequencerInboxSequencerBatchDelivered)
	err := sequencerBridgeABI.UnpackIntoInterface(deliveredEvent, sequencerBatchDataEvent, log.Data)
	if err != nil {
		return err
	}
	// TODO: retention time should start on log event, not on current time
	storeUntil := arbmath.SaturatingUAdd(uint64(time.Now().Unix()), uint64(s.config.RetentionPeriod.Seconds())) // TODO: support limited retention period
	if deliveredEvent.DataLocation == uint8(batchDataSeparateEvent) {
		query := ethereum.FilterQuery{
			BlockHash: &log.BlockHash,
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
		dataEvent := new(bridgegen.SequencerInboxSequencerBatchData)
		err = sequencerBridgeABI.UnpackIntoInterface(dataEvent, sequencerBatchDataEvent, log.Data)
		if err != nil {
			return err
		}
		data = dataEvent.Data
	} else if deliveredEvent.DataLocation == uint8(batchDataTxInput) {
		tx, err := s.l1Reader.Client().TransactionInBlock(ctx, log.BlockHash, log.TxIndex)
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
	if len(data) < 41 {
		// no data - nothing to do
		return nil
	}
	if !arbstate.IsDASMessageHeaderByte(data[40]) {
		return nil
	}
	preimages := make(map[common.Hash][]byte)
	if _, err = arbstate.RecoverPayloadFromDasBatch(ctx, data, s.dataSource, preimages); err != nil {
		return err
	}
	for hash, contents := range preimages {
		_, err := s.syncTo.GetByHash(ctx, hash.Bytes())
		if errors.Is(err, ErrNotFound) {
			if err := s.syncTo.Put(ctx, contents, storeUntil); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	updatedBatchCount := new(big.Int).Add(deliveredEvent.BatchSequenceNumber, common.Big1)
	if s.lastBatchCount.Cmp(updatedBatchCount) <= 0 {
		s.lastBatchCount.Set(deliveredEvent.BatchSequenceNumber)
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
	for _, log := range logs {
		thisLog := log
		if err := s.processBatchDelivered(ctx, &thisLog); err != nil {
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
	s.StopWaiter.Start(ctxIn)

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
