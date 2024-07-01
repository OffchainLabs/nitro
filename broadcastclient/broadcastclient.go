// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcastclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/httphead"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsflate"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	m "github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

var (
	sourcesConnectedGauge    = metrics.NewRegisteredGauge("arb/feed/sources/connected", nil)
	sourcesDisconnectedGauge = metrics.NewRegisteredGauge("arb/feed/sources/disconnected", nil)
)

type FeedConfig struct {
	Output wsbroadcastserver.BroadcasterConfig `koanf:"output" reload:"hot"`
	Input  Config                              `koanf:"input" reload:"hot"`
}

func (fc *FeedConfig) Validate() error {
	return fc.Output.Validate()
}

func FeedConfigAddOptions(prefix string, f *flag.FlagSet, feedInputEnable bool, feedOutputEnable bool) {
	if feedInputEnable {
		ConfigAddOptions(prefix+".input", f)
	}
	if feedOutputEnable {
		wsbroadcastserver.BroadcasterConfigAddOptions(prefix+".output", f)
	}
}

var FeedConfigDefault = FeedConfig{
	Output: wsbroadcastserver.DefaultBroadcasterConfig,
	Input:  DefaultConfig,
}

type Config struct {
	ReconnectInitialBackoff time.Duration            `koanf:"reconnect-initial-backoff" reload:"hot"`
	ReconnectMaximumBackoff time.Duration            `koanf:"reconnect-maximum-backoff" reload:"hot"`
	RequireChainId          bool                     `koanf:"require-chain-id" reload:"hot"`
	RequireFeedVersion      bool                     `koanf:"require-feed-version" reload:"hot"`
	Timeout                 time.Duration            `koanf:"timeout" reload:"hot"`
	URL                     []string                 `koanf:"url"`
	SecondaryURL            []string                 `koanf:"secondary-url"`
	Verify                  signature.VerifierConfig `koanf:"verify"`
	EnableCompression       bool                     `koanf:"enable-compression" reload:"hot"`
}

func (c *Config) Enable() bool {
	return len(c.URL) > 0 && c.URL[0] != ""
}

type ConfigFetcher func() *Config

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Duration(prefix+".reconnect-initial-backoff", DefaultConfig.ReconnectInitialBackoff, "initial duration to wait before reconnect")
	f.Duration(prefix+".reconnect-maximum-backoff", DefaultConfig.ReconnectMaximumBackoff, "maximum duration to wait before reconnect")
	f.Bool(prefix+".require-chain-id", DefaultConfig.RequireChainId, "require chain id to be present on connect")
	f.Bool(prefix+".require-feed-version", DefaultConfig.RequireFeedVersion, "require feed version to be present on connect")
	f.Duration(prefix+".timeout", DefaultConfig.Timeout, "duration to wait before timing out connection to sequencer feed")
	f.StringSlice(prefix+".url", DefaultConfig.URL, "list of primary URLs of sequencer feed source")
	f.StringSlice(prefix+".secondary-url", DefaultConfig.SecondaryURL, "list of secondary URLs of sequencer feed source. Would be started in the order they appear in the list when primary feeds fails")
	signature.FeedVerifierConfigAddOptions(prefix+".verify", f)
	f.Bool(prefix+".enable-compression", DefaultConfig.EnableCompression, "enable per message deflate compression support")
}

var DefaultConfig = Config{
	ReconnectInitialBackoff: time.Second * 1,
	ReconnectMaximumBackoff: time.Second * 64,
	RequireChainId:          false,
	RequireFeedVersion:      false,
	Verify:                  signature.DefultFeedVerifierConfig,
	URL:                     []string{},
	SecondaryURL:            []string{},
	Timeout:                 20 * time.Second,
	EnableCompression:       true,
}

var DefaultTestConfig = Config{
	ReconnectInitialBackoff: 0,
	ReconnectMaximumBackoff: 0,
	RequireChainId:          false,
	RequireFeedVersion:      false,
	Verify:                  signature.DefultFeedVerifierConfig,
	URL:                     []string{""},
	SecondaryURL:            []string{},
	Timeout:                 200 * time.Millisecond,
	EnableCompression:       true,
}

type TransactionStreamerInterface interface {
	AddBroadcastMessages(feedMessages []*m.BroadcastFeedMessage) error
}

type BroadcastClient struct {
	stopwaiter.StopWaiter

	config       ConfigFetcher
	websocketUrl string
	nextSeqNum   arbutil.MessageIndex
	sigVerifier  *signature.Verifier

	chainId uint64

	// Protects conn and shuttingDown
	connMutex sync.Mutex
	conn      net.Conn

	retryCount int64

	retrying                        bool
	shuttingDown                    bool
	confirmedSequenceNumberListener chan arbutil.MessageIndex
	txStreamer                      TransactionStreamerInterface
	fatalErrChan                    chan error
	adjustCount                     func(int32)
}

var ErrIncorrectFeedServerVersion = errors.New("incorrect feed server version")
var ErrIncorrectChainId = errors.New("incorrect chain id")
var ErrMissingChainId = errors.New("missing chain id")
var ErrMissingFeedServerVersion = errors.New("missing feed server version")

func NewBroadcastClient(
	config ConfigFetcher,
	websocketUrl string,
	chainId uint64,
	currentMessageCount arbutil.MessageIndex,
	txStreamer TransactionStreamerInterface,
	confirmedSequencerNumberListener chan arbutil.MessageIndex,
	fatalErrChan chan error,
	addrVerifier contracts.AddressVerifierInterface,
	adjustCount func(int32),
) (*BroadcastClient, error) {
	sigVerifier, err := signature.NewVerifier(&config().Verify, addrVerifier)
	if err != nil {
		return nil, err
	}
	return &BroadcastClient{
		config:                          config,
		websocketUrl:                    websocketUrl,
		chainId:                         chainId,
		nextSeqNum:                      currentMessageCount,
		txStreamer:                      txStreamer,
		confirmedSequenceNumberListener: confirmedSequencerNumberListener,
		fatalErrChan:                    fatalErrChan,
		sigVerifier:                     sigVerifier,
		adjustCount:                     adjustCount,
	}, err
}

func (bc *BroadcastClient) Start(ctxIn context.Context) {
	bc.StopWaiter.Start(ctxIn, bc)
	if bc.StopWaiter.Stopped() {
		log.Info("broadcast client has already been stopped, not starting")
		return
	}
	bc.LaunchThread(func(ctx context.Context) {
		backoffDuration := bc.config().ReconnectInitialBackoff
		for {
			earlyFrameData, err := bc.connect(ctx, bc.nextSeqNum)
			if errors.Is(err, ErrMissingChainId) ||
				errors.Is(err, ErrIncorrectChainId) ||
				errors.Is(err, ErrMissingFeedServerVersion) ||
				errors.Is(err, ErrIncorrectFeedServerVersion) {
				bc.fatalErrChan <- fmt.Errorf("failed connecting to server feed due to %w", err)
				return
			}
			if err == nil {
				bc.startBackgroundReader(earlyFrameData)
				break
			}
			log.Warn("failed connect to sequencer broadcast, waiting and retrying", "url", bc.websocketUrl, "err", err)
			timer := time.NewTimer(backoffDuration)
			if backoffDuration < bc.config().ReconnectMaximumBackoff {
				backoffDuration *= 2
			}
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	})
}

func (bc *BroadcastClient) connect(ctx context.Context, nextSeqNum arbutil.MessageIndex) (io.Reader, error) {
	if len(bc.websocketUrl) == 0 {
		// Nothing to do
		return nil, nil
	}

	header := ws.HandshakeHeaderHTTP(http.Header{
		wsbroadcastserver.HTTPHeaderFeedClientVersion:       []string{strconv.Itoa(wsbroadcastserver.FeedClientVersion)},
		wsbroadcastserver.HTTPHeaderRequestedSequenceNumber: []string{strconv.FormatUint(uint64(nextSeqNum), 10)},
	})

	log.Info("connecting to arbitrum inbox message broadcaster", "url", bc.websocketUrl)
	var foundChainId bool
	var foundFeedServerVersion bool
	var chainId uint64
	var feedServerVersion uint64

	config := bc.config()
	var extensions []httphead.Option
	deflateExt := wsflate.DefaultParameters.Option()
	if config.EnableCompression {
		extensions = []httphead.Option{deflateExt}
	}
	timeoutDialer := ws.Dialer{
		Header: header,
		OnHeader: func(key, value []byte) (err error) {
			headerName := string(key)
			headerValue := string(value)
			if headerName == wsbroadcastserver.HTTPHeaderFeedServerVersion {
				foundFeedServerVersion = true
				feedServerVersion, err = strconv.ParseUint(headerValue, 0, 64)
				if err != nil {
					return err
				}
				if feedServerVersion != wsbroadcastserver.FeedServerVersion {
					log.Error(
						"incorrect feed server version",
						"expectedFeedServerVersion",
						wsbroadcastserver.FeedServerVersion,
						"actualFeedServerVersion",
						feedServerVersion,
					)
					return ErrIncorrectFeedServerVersion
				}
			} else if headerName == wsbroadcastserver.HTTPHeaderChainId {
				foundChainId = true
				chainId, err = strconv.ParseUint(headerValue, 0, 64)
				if err != nil {
					return err
				}
				if chainId != bc.chainId {
					log.Error(
						"incorrect chain id when connecting to server feed",
						"expectedChainId",
						bc.chainId,
						"actualChainId",
						chainId,
					)
					return ErrIncorrectChainId
				}
			}
			return nil
		},
		Timeout: 10 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		Extensions: extensions,
	}

	if bc.isShuttingDown() {
		return nil, nil
	}

	conn, br, _, err := timeoutDialer.Dial(ctx, bc.websocketUrl)
	if errors.Is(err, ErrIncorrectFeedServerVersion) || errors.Is(err, ErrIncorrectChainId) {
		return nil, err
	}
	if strings.Contains(err.Error(), wsbroadcastserver.ErrRateLimited.Error()) {
		log.Error("Failed connecting to feed due to hitting rate limit", "Status", http.StatusTooManyRequests, "err", err)
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("broadcast client unable to connect: %w", err)
	}
	if config.RequireChainId && !foundChainId {
		err := conn.Close()
		if err != nil {
			return nil, fmt.Errorf("error closing connection when missing chain id: %w", err)
		}
		return nil, ErrMissingChainId
	}
	if config.RequireFeedVersion && !foundFeedServerVersion {
		err := conn.Close()
		if err != nil {
			return nil, fmt.Errorf("error closing connection when missing feed server version: %w", err)
		}
		return nil, ErrMissingFeedServerVersion
	}

	var earlyFrameData io.Reader
	if br != nil {
		// Depending on how long the client takes to read the response, there may be
		// data after the WebSocket upgrade response in a single read from the socket,
		// ie WebSocket frames sent by the server. If this happens, Dial returns
		// a non-nil bufio.Reader so that data isn't lost. But beware, this buffered
		// reader is still hooked up to the socket; trying to read past what had already
		// been buffered will do a blocking read on the socket, so we have to wrap it
		// in a LimitedReader.
		earlyFrameData = io.LimitReader(br, int64(br.Buffered()))
	}

	bc.connMutex.Lock()
	bc.conn = conn
	bc.connMutex.Unlock()
	log.Info("Feed connected", "feedServerVersion", feedServerVersion, "chainId", chainId, "requestedSeqNum", nextSeqNum)

	return earlyFrameData, nil
}

func (bc *BroadcastClient) startBackgroundReader(earlyFrameData io.Reader) {
	bc.LaunchThread(func(ctx context.Context) {
		connected := false
		sourcesDisconnectedGauge.Inc(1)
		backoffDuration := bc.config().ReconnectInitialBackoff
		flateReader := wsbroadcastserver.NewFlateReader()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			var msg []byte
			var op ws.OpCode
			var err error
			config := bc.config()
			msg, op, err = wsbroadcastserver.ReadData(ctx, bc.conn, earlyFrameData, config.Timeout, ws.StateClientSide, config.EnableCompression, flateReader)
			if err != nil {
				if bc.isShuttingDown() {
					return
				}
				if strings.Contains(err.Error(), "i/o timeout") {
					log.Error("Server connection timed out without receiving data", "url", bc.websocketUrl, "err", err)
				} else if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					log.Warn("readData returned EOF", "url", bc.websocketUrl, "opcode", int(op), "err", err)
				} else {
					log.Error("error calling readData", "url", bc.websocketUrl, "opcode", int(op), "err", err)
				}
				if connected {
					connected = false
					bc.adjustCount(-1)
					sourcesConnectedGauge.Dec(1)
					sourcesDisconnectedGauge.Inc(1)
				}
				_ = bc.conn.Close()
				timer := time.NewTimer(backoffDuration)
				if backoffDuration < bc.config().ReconnectMaximumBackoff {
					backoffDuration *= 2
				}
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				earlyFrameData = bc.retryConnect(ctx)
				continue
			}
			backoffDuration = bc.config().ReconnectInitialBackoff

			if msg != nil {
				res := m.BroadcastMessage{}
				err = json.Unmarshal(msg, &res)
				if err != nil {
					log.Error("error unmarshalling message", "msg", msg, "err", err)
					continue
				}

				if !connected {
					connected = true
					sourcesDisconnectedGauge.Dec(1)
					sourcesConnectedGauge.Inc(1)
					bc.adjustCount(1)
				}
				if len(res.Messages) > 0 {
					log.Debug("received batch item", "count", len(res.Messages), "first seq", res.Messages[0].SequenceNumber)
				} else if res.ConfirmedSequenceNumberMessage != nil {
					log.Debug("confirmed sequence number", "seq", res.ConfirmedSequenceNumberMessage.SequenceNumber)
				} else {
					log.Debug("received broadcast with no messages populated", "length", len(msg))
				}
				if res.Version == 1 {
					if len(res.Messages) > 0 {
						for _, message := range res.Messages {
							if message == nil {
								log.Warn("ignoring nil feed message")
								continue
							}

							err := bc.isValidSignature(ctx, message)
							if err != nil {
								log.Error("error validating feed signature", "error", err, "sequence number", message.SequenceNumber)
								bc.fatalErrChan <- fmt.Errorf("error validating feed signature %v: %w", message.SequenceNumber, err)
								continue
							}

							bc.nextSeqNum = message.SequenceNumber + 1
						}
						if err := bc.txStreamer.AddBroadcastMessages(res.Messages); err != nil {
							log.Error("Error adding message from Sequencer Feed", "err", err)
						}
					}
					if res.ConfirmedSequenceNumberMessage != nil && bc.confirmedSequenceNumberListener != nil {
						bc.confirmedSequenceNumberListener <- res.ConfirmedSequenceNumberMessage.SequenceNumber
					}
				}
			}
		}
	})
}

func (bc *BroadcastClient) GetRetryCount() int64 {
	return atomic.LoadInt64(&bc.retryCount)
}

func (bc *BroadcastClient) isShuttingDown() bool {
	bc.connMutex.Lock()
	defer bc.connMutex.Unlock()
	return bc.shuttingDown
}

func (bc *BroadcastClient) retryConnect(ctx context.Context) io.Reader {
	maxWaitDuration := 15 * time.Second
	waitDuration := 500 * time.Millisecond
	bc.retrying = true

	for !bc.isShuttingDown() {
		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}

		atomic.AddInt64(&bc.retryCount, 1)
		earlyFrameData, err := bc.connect(ctx, bc.nextSeqNum)
		if err == nil {
			bc.retrying = false
			return earlyFrameData
		}

		if waitDuration < maxWaitDuration {
			waitDuration += 500 * time.Millisecond
		}
	}
	return nil
}

func (bc *BroadcastClient) StopAndWait() {
	log.Debug("closing broadcaster client connection")
	bc.StopWaiter.StopAndWait()
	bc.connMutex.Lock()
	defer bc.connMutex.Unlock()

	if !bc.shuttingDown {
		bc.shuttingDown = true
		if bc.conn != nil {
			_ = bc.conn.Close()
		}
	}
}

func (bc *BroadcastClient) isValidSignature(ctx context.Context, message *m.BroadcastFeedMessage) error {
	if bc.config().Verify.Dangerous.AcceptMissing && bc.sigVerifier == nil {
		// Verifier disabled
		return nil
	}
	hash, err := message.Hash(bc.chainId)
	if err != nil {
		return fmt.Errorf("error getting message hash for sequence number %v: %w", message.SequenceNumber, err)
	}
	return bc.sigVerifier.VerifyHash(ctx, message.Signature, hash)
}
