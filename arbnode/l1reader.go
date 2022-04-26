package arbnode

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
)

type L1Reader struct {
	stopwaiter.StopWaiter
	config L1ReaderConfig
	client arbutil.L1Interface

	chanMutex sync.Mutex
	// All fields below require the chanMutex
	outChannels                map[chan<- *types.Header]struct{}
	outChannelsBehind          map[chan<- *types.Header]struct{}
	lastBroadcastHash          common.Hash
	lastBroadcastHeader        *types.Header
	lastPendingCallBlockNr     uint64
	requiresPendingCallUpdates int
}

type L1ReaderConfig struct {
	Enable               bool          `koanf:"enable"`
	PollOnly             bool          `koanf:"poll-only"`
	PollInterval         time.Duration `koanf:"poll-interval"`
	SubscribeErrInterval time.Duration `koanf:"subscribe-err-interval"`
	TxTimeout            time.Duration `koanf:"tx-timeout"`
}

var DefaultL1ReaderConfig = L1ReaderConfig{
	Enable:               true,
	PollOnly:             false,
	PollInterval:         15 * time.Second,
	SubscribeErrInterval: 5 * time.Minute,
	TxTimeout:            5 * time.Minute,
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
	PollInterval: time.Millisecond * 10,
	TxTimeout:    time.Second * 5,
}

func NewL1Reader(client arbutil.L1Interface, config L1ReaderConfig) *L1Reader {
	return &L1Reader{
		client:            client,
		config:            config,
		outChannels:       make(map[chan<- *types.Header]struct{}),
		outChannelsBehind: make(map[chan<- *types.Header]struct{}),
	}
}

// Subscribers are notified when there is a change.
// Channel could be missing headers and have duplicates.
// Listening to the channel will make sure listenere is notified when header changes.
func (s *L1Reader) Subscribe(requireBlockNrUpdates bool) (<-chan *types.Header, func()) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	if requireBlockNrUpdates {
		s.requiresPendingCallUpdates++
	}
	result := make(chan *types.Header)
	outchannel := (chan<- *types.Header)(result)
	s.outChannelsBehind[outchannel] = struct{}{}
	unsubscribeFunc := func() { s.unsubscribe(requireBlockNrUpdates, outchannel) }
	return result, unsubscribeFunc
}

func (s *L1Reader) unsubscribe(requireBlockNrUpdates bool, from chan<- *types.Header) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	if requireBlockNrUpdates {
		s.requiresPendingCallUpdates--
	}

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

	s.requiresPendingCallUpdates = 0

	for ch := range s.outChannels {
		delete(s.outChannels, ch)
		close(ch)
	}
	for ch := range s.outChannelsBehind {
		delete(s.outChannelsBehind, ch)
		close(ch)
	}
}

func (s *L1Reader) possiblyBroadcast(h *types.Header) {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()

	headerHash := h.Hash()
	broadcastThis := false

	if headerHash != s.lastBroadcastHash {
		broadcastThis = true
		s.lastBroadcastHash = headerHash
		s.lastBroadcastHeader = h
	}

	if s.requiresPendingCallUpdates > 0 {
		pendingCallBlockNr, err := arbutil.GetPendingCallBlockNumber(s.GetContext(), s.client)
		if err == nil && pendingCallBlockNr.IsUint64() {
			pendingU64 := pendingCallBlockNr.Uint64()
			if pendingU64 > s.lastPendingCallBlockNr {
				broadcastThis = true
				s.lastPendingCallBlockNr = pendingU64
			}
		} else {
			log.Warn("GetPendingCallBlockNr: bad result", "err", err, "number", pendingCallBlockNr)
		}
	}

	if broadcastThis {
		for ch := range s.outChannels {
			select {
			case ch <- h:
			default:
				delete(s.outChannels, ch)
				s.outChannelsBehind[ch] = struct{}{}
			}
		}
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

func (s *L1Reader) broadcastLoop(ctx context.Context) {
	var clientSubscription ethereum.Subscription = nil
	defer func() {
		if clientSubscription != nil {
			clientSubscription.Unsubscribe()
		}
	}()
	inputChannel := make(chan *types.Header)
	if err := ctx.Err(); err != nil {
		return
	}
	ticker := time.NewTicker(s.config.PollInterval)
	nextSubscribeErr := time.Now().Add(-time.Second)
	var errChannel <-chan error
	for {
		if clientSubscription != nil {
			errChannel = clientSubscription.Err()
		} else {
			errChannel = nil
		}
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
			if !s.config.PollOnly && clientSubscription == nil {
				clientSubscription, err = s.client.SubscribeNewHead(ctx, inputChannel)
				if err != nil {
					clientSubscription = nil
					if errors.Is(err, rpc.ErrNotificationsUnsupported) {
						s.config.PollOnly = true
					} else if time.Now().After(nextSubscribeErr) {
						log.Warn("failed subscribing to header", "err", err)
						nextSubscribeErr = time.Now().Add(s.config.SubscribeErrInterval)
					}
				}
			}
		case err := <-errChannel:
			if ctx.Err() != nil {
				return
			}
			clientSubscription = nil
			log.Warn("error in subscription to L1 headers", "err", err)
		case <-ctx.Done():
			return
		}
	}
}

func (s *L1Reader) WaitForTxApproval(ctxIn context.Context, tx *types.Transaction) (*types.Receipt, error) {
	headerchan, unsubscribe := s.Subscribe(true)
	defer unsubscribe()
	ctx, cancel := context.WithTimeout(ctxIn, s.config.TxTimeout)
	defer cancel()
	txHash := tx.Hash()
	for {
		receipt, err := s.client.TransactionReceipt(ctx, txHash)
		if err == nil && receipt.BlockNumber.IsUint64() {
			receiptBlockNr := receipt.BlockNumber.Uint64()
			callBlockNr := s.LastPendingCallBlockNr()
			if callBlockNr > receiptBlockNr {
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

func (s *L1Reader) UpdatingPendingCallBlockNr() bool {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	return s.requiresPendingCallUpdates > 0
}

// blocknumber used by pending calls.
// only updated if UpdatingPendingCallBlockNr returns true
func (s *L1Reader) LastPendingCallBlockNr() uint64 {
	s.chanMutex.Lock()
	defer s.chanMutex.Unlock()
	return s.lastPendingCallBlockNr
}

func (s *L1Reader) Client() arbutil.L1Interface {
	return s.client
}

func (s *L1Reader) Start(ctxIn context.Context) {
	s.StopWaiter.Start(ctxIn)
	s.LaunchThread(s.broadcastLoop)
}

func (s *L1Reader) StopAndWait() {
	s.StopWaiter.StopAndWait()
	s.closeAll()
}
