package arbnode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util"
	flag "github.com/spf13/pflag"
)

type L1Reader struct {
	util.StopWaiter
	config              L1ReaderConfig
	client              arbutil.L1Interface
	outChannels         map[chan<- *types.Header]struct{}
	outChannelsBehind   map[chan<- *types.Header]struct{}
	chanMutex           sync.Mutex
	lastBroadcastHash   common.Hash
	lastBroadcastHeader *types.Header
}

type L1ReaderConfig struct {
	Enable       bool          `koanf:"enable"`
	PollOnly     bool          `koanf:"poll-only"`
	PollInterval time.Duration `koanf:"poll-interval"`
	TxTimeout    time.Duration `koanf:"tx-timeout"`
}

var DefaultL1ReaderConfig = L1ReaderConfig{
	Enable:       true,
	PollOnly:     false,
	PollInterval: 15 * time.Second,
	TxTimeout:    time.Minute,
}

func L1ReaderAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultL1ReaderConfig.Enable, "enable l1 connection")
	f.Bool(prefix+".poll-only", DefaultL1ReaderConfig.PollOnly, "do not attempt to subscribe to L1 events")
	f.Duration(prefix+".poll-interval", DefaultL1ReaderConfig.PollInterval, "interval when polling L1")
	f.Duration(prefix+".tx-timeout", DefaultL1ReaderConfig.TxTimeout, "timeout when waiting for a transaction")
}

var TestL1ReaderConfig = L1ReaderConfig{
	Enable:       true,
	PollOnly:     false,
	PollInterval: time.Second,
	TxTimeout:    time.Second * 4,
}

func NewL1Reader(client arbutil.L1Interface, config L1ReaderConfig) *L1Reader {
	return &L1Reader{
		client:            client,
		config:            config,
		outChannels:       make(map[chan<- *types.Header]struct{}),
		outChannelsBehind: make(map[chan<- *types.Header]struct{}),
	}
}

func (s *L1Reader) Subscribe() (<-chan *types.Header, func()) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	result := make(chan *types.Header)
	outchannel := (chan<- *types.Header)(result)
	s.outChannelsBehind[outchannel] = struct{}{}
	unsubscribeFunc := func() { s.unsubscribe(outchannel) }
	return result, unsubscribeFunc
}

func (s *L1Reader) unsubscribe(from chan<- *types.Header) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	if _, ok := s.outChannels[from]; ok {
		delete(s.outChannels, from)
		close(from)
	}
	if _, ok := s.outChannelsBehind[from]; ok {
		delete(s.outChannelsBehind, from)
		close(from)
	}
}

func (s *L1Reader) closeAll() {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	for ch := range s.outChannels {
		delete(s.outChannels, ch)
		close(ch)
	}
}

func (s *L1Reader) possiblyBroadcast(h *types.Header) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	headerHash := h.Hash()

	if headerHash != s.lastBroadcastHash {
		for ch := range s.outChannels {
			select {
			case ch <- h:
			default:
				delete(s.outChannels, ch)
				s.outChannelsBehind[ch] = struct{}{}
			}
		}
		s.lastBroadcastHash = headerHash
		s.lastBroadcastHeader = h
	}

	for ch := range s.outChannelsBehind {
		select {
		case ch <- h:
			delete(s.outChannelsBehind, ch)
			s.outChannels[ch] = struct{}{}
		default:
		}
	}
}

func (s *L1Reader) pollHeader(ctx context.Context) time.Duration {
	lastHeader, err := s.client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Warn("failed reading l1 header", "err", err)
		return s.config.PollInterval
	}
	s.possiblyBroadcast(lastHeader)
	return s.config.PollInterval
}

func (s *L1Reader) subscribeLoop(ctx context.Context) {
	inputChannel := make(chan *types.Header)
	if err := ctx.Err(); err != nil {
		return
	}
	headerSubscription, err := s.client.SubscribeNewHead(ctx, inputChannel)
	if err != nil {
		log.Error("failed subscribing to header", "err", err)
		return
	}
	ticker := time.NewTicker(s.config.PollInterval)
	for {
		select {
		case h := <-inputChannel:
			s.possiblyBroadcast(h)
		case <-ticker.C:
			h, err := s.client.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Warn("failed reading l1 header", "err", err)
			} else {
				s.possiblyBroadcast(h)
			}
		case err := <-headerSubscription.Err():
			if ctx.Err() == nil {
				return
			}
			log.Warn("error in subscription to L1 headers", "err", err)
			for {
				headerSubscription, err = s.client.SubscribeNewHead(ctx, inputChannel)
				if err == nil {
					break
				}
				log.Warn("error re-subscribing to L1 headers", "err", err)
				timer := time.NewTimer(s.pollHeader(ctx))
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		case <-ctx.Done():
			headerSubscription.Unsubscribe()
			return
		}
	}
}

func (s *L1Reader) WaitForTxApproval(ctxIn context.Context, tx *types.Transaction) (*types.Receipt, error) {
	headerchan, unsubscribe := s.Subscribe()
	defer unsubscribe()
	ctx, cancel := context.WithTimeout(ctxIn, s.config.TxTimeout)
	defer cancel()
	txHash := tx.Hash()
	for {
		receipt, err := s.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			callBlockNr, err := arbutil.GetCallMsgBlockNumber(ctx, s.client)
			if err != nil {
				return nil, err
			}
			if callBlockNr.Cmp(receipt.BlockNumber) >= 0 {
				return receipt, arbutil.DetailTxError(ctx, s.client, tx, receipt)
			}
		}
		select {
		case _, ok := <-headerchan:
			if !ok {
				return nil, fmt.Errorf("waiting for %v: channel closed", txHash)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (s *L1Reader) LastHeader(ctx context.Context) (*types.Header, error) {
	s.chanMutex.Lock()
	storedHeader := s.lastBroadcastHeader
	s.chanMutex.Unlock()
	if storedHeader != nil {
		return storedHeader, nil
	}
	return s.client.HeaderByNumber(ctx, nil)
}

func (s *L1Reader) Client() arbutil.L1Interface {
	return s.client
}

func (s *L1Reader) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn)
	if s.config.PollOnly {
		s.CallIteratively(s.pollHeader)
	} else {
		s.LaunchThread(s.subscribeLoop)
	}
}

func (s *L1Reader) StopAndWait() {
	s.closeAll()
	s.StopWaiter.StopAndWait()
}
