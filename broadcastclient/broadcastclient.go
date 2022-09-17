// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package broadcastclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
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
	Input  Config                              `koanf:"input"`
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
	RequireChainId     bool          `koanf:"require-chain-id"`
	RequireFeedVersion bool          `koanf:"require-feed-version"`
	RequireSignature   bool          `koanf:"require-signature"`
	Timeout            time.Duration `koanf:"timeout"`
	URLs               []string      `koanf:"url"`
}

func (c *Config) Enable() bool {
	return len(c.URLs) > 0 && c.URLs[0] != ""
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".require-chain-id", DefaultConfig.RequireChainId, "require chain id to be present on connect")
	f.Bool(prefix+".require-feed-version", DefaultConfig.RequireFeedVersion, "require feed version to be present on connect")
	f.Bool(prefix+".require-signature", DefaultConfig.RequireSignature, "require all feed messages to be signed")
	f.Duration(prefix+".timeout", DefaultConfig.Timeout, "duration to wait before timing out connection to sequencer feed")
	f.StringSlice(prefix+".url", DefaultConfig.URLs, "URL of sequencer feed source")
}

var DefaultConfig = Config{
	RequireChainId:     false,
	RequireFeedVersion: false,
	RequireSignature:   false,
	URLs:               []string{""},
	Timeout:            20 * time.Second,
}

var DefaultTestConfig = Config{
	RequireSignature: true,
	URLs:             []string{""},
	Timeout:          200 * time.Millisecond,
}

type TransactionStreamerInterface interface {
	AddBroadcastMessages(feedMessages []*broadcaster.BroadcastFeedMessage) error
}

type BroadcastClient struct {
	stopwaiter.StopWaiter

	config       Config
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
	ConfirmedSequenceNumberListener chan arbutil.MessageIndex
	txStreamer                      TransactionStreamerInterface
	fatalErrChan                    chan error
}

var ErrIncorrectFeedServerVersion = errors.New("incorrect feed server version")
var ErrIncorrectChainId = errors.New("incorrect chain id")
var ErrInvalidFeedSignature = errors.New("invalid feed signature")
var ErrMissingChainId = errors.New("missing chain id")
var ErrMissingFeedServerVersion = errors.New("missing feed server version")

func NewBroadcastClient(
	config Config,
	websocketUrl string,
	chainId uint64,
	currentMessageCount arbutil.MessageIndex,
	txStreamer TransactionStreamerInterface,
	fatalErrChan chan error,
	sigVerifier *signature.Verifier,
) *BroadcastClient {
	return &BroadcastClient{
		config:       config,
		websocketUrl: websocketUrl,
		chainId:      chainId,
		nextSeqNum:   currentMessageCount,
		txStreamer:   txStreamer,
		fatalErrChan: fatalErrChan,
		sigVerifier:  sigVerifier,
	}
}

func (bc *BroadcastClient) Start(ctxIn context.Context) {
	bc.StopWaiter.Start(ctxIn, bc)
	bc.LaunchThread(func(ctx context.Context) {
		for {
			earlyFrameData, err := bc.connect(ctx, bc.nextSeqNum)
			if errors.Is(err, ErrMissingChainId) ||
				errors.Is(err, ErrIncorrectChainId) ||
				errors.Is(err, ErrMissingFeedServerVersion) ||
				errors.Is(err, ErrIncorrectFeedServerVersion) {
				bc.fatalErrChan <- err
				return
			}
			if err == nil {
				bc.startBackgroundReader(earlyFrameData)
				break
			}
			log.Warn("failed connect to sequencer broadcast, waiting and retrying", "url", bc.websocketUrl, "err", err)
			timer := time.NewTimer(5 * time.Second)
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
	}

	if bc.isShuttingDown() {
		return nil, nil
	}

	conn, br, _, err := timeoutDialer.Dial(ctx, bc.websocketUrl)
	if errors.Is(err, ErrIncorrectFeedServerVersion) || errors.Is(err, ErrIncorrectChainId) {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrap(err, "broadcast client unable to connect")
	}
	if bc.config.RequireChainId && !foundChainId {
		err := conn.Close()
		if err != nil {
			return nil, errors.Wrap(err, "error closing connection when missing chain id")
		}
		return nil, ErrMissingChainId
	}
	if bc.config.RequireFeedVersion && !foundFeedServerVersion {
		err := conn.Close()
		if err != nil {
			return nil, errors.Wrap(err, "error closing connection when missing feed server version")
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
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg, op, err := wsbroadcastserver.ReadData(ctx, bc.conn, earlyFrameData, bc.config.Timeout, ws.StateClientSide)
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
					sourcesConnectedGauge.Dec(1)
					sourcesDisconnectedGauge.Inc(1)
				}
				_ = bc.conn.Close()
				earlyFrameData = bc.retryConnect(ctx)
				continue
			}

			if msg != nil {
				res := broadcaster.BroadcastMessage{}
				err = json.Unmarshal(msg, &res)
				if err != nil {
					log.Error("error unmarshalling message", "msg", msg, "err", err)
					continue
				}

				if !connected {
					connected = true
					sourcesDisconnectedGauge.Dec(1)
					sourcesConnectedGauge.Inc(1)
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

							valid, err := bc.isValidSignature(ctx, message)
							if err != nil {
								log.Error("error validating feed signature", "error", err, "sequence number", message.SequenceNumber)
								bc.fatalErrChan <- errors.Wrapf(err, "error validating feed signature %v", message.SequenceNumber)
								continue
							}

							if !valid {
								log.Error("invalid feed signature", "sequence number", message.SequenceNumber)
								bc.fatalErrChan <- ErrInvalidFeedSignature
								continue
							}
							bc.nextSeqNum = message.SequenceNumber
						}
						if err := bc.txStreamer.AddBroadcastMessages(res.Messages); err != nil {
							log.Error("Error adding message from Sequencer Feed", "err", err)
						}
					}
					if res.ConfirmedSequenceNumberMessage != nil && bc.ConfirmedSequenceNumberListener != nil {
						bc.ConfirmedSequenceNumberListener <- res.ConfirmedSequenceNumberMessage.SequenceNumber
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

	bc.shuttingDown = true
	if bc.conn != nil {
		_ = bc.conn.Close()
	}
}

func (bc *BroadcastClient) isValidSignature(ctx context.Context, message *broadcaster.BroadcastFeedMessage) (bool, error) {
	if !bc.config.RequireSignature && bc.sigVerifier == nil {
		// Verifier disabled
		return true, nil
	}
	hash, err := message.Hash(bc.chainId)
	if err != nil {
		return false, errors.Wrapf(err, "error getting message hash for sequence number %v", message.SequenceNumber)
	}
	return bc.sigVerifier.VerifyHash(ctx, message.Signature, hash)
}
