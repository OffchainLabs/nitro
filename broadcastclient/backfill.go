// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcastclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/contracts"
	"github.com/offchainlabs/nitro/util/httpclient"
	"github.com/offchainlabs/nitro/util/httperror"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	backfillRunsCounter            = metrics.NewRegisteredCounter("arb/feed/backfill/runs", nil)
	backfillChunksCounter          = metrics.NewRegisteredCounter("arb/feed/backfill/chunks", nil)
	backfillMessagesCounter        = metrics.NewRegisteredCounter("arb/feed/backfill/messages", nil)
	backfillChunksMissingCounter   = metrics.NewRegisteredCounter("arb/feed/backfill/chunks_missing", nil)
	backfillErrorsCounter          = metrics.NewRegisteredCounter("arb/feed/backfill/errors", nil)
	backfillPermanentErrorsCounter = metrics.NewRegisteredCounter("arb/feed/backfill/permanent_errors", nil)
)

const (
	feedInfoPath            = "/feed/v1/info"
	feedChunkPathPrefix     = "/feed/v1/chunk/"
	feedBackfillHeaderValue = "1"
)

// Upper sanity bound on the archive's advertised chunk size; the zero check beside it keeps the
// head alignment from dividing by zero.
const maxChunkSize = 10_000

const (
	backfillRequestAttempts = 3
	backfillRetryDelay      = 500 * time.Millisecond
	backfillPollInterval    = time.Second
	backfillMaxBackoff      = 64 * time.Second
)

// errChunkMissing is a 404: the archive does not have the chunk, because it has not settled yet or
// because it was evicted. The contract does not distinguish the two, and both end a run.
var errChunkMissing = errors.New("feed chunk not available from this archive")

// errArchiveMismatch wraps failures that repeat on every run until the archive or the
// configuration changes: another chain or feed version, an unusable chunk size, a history that
// disagrees with the database, or a url that does not serve the feed info.
var errArchiveMismatch = errors.New("archive does not serve the expected feed backlog")

// Returned by AddBroadcastBackfillMessages when a chunk cannot be written and refetching it will
// not help until something else changes.
var (
	ErrBackfillFeedReorgPending = errors.New("the live feed is reorging and the parent chain has to settle it")
	ErrBackfillDiverged         = fmt.Errorf("%w: backfilled chunk disagrees with a stored message", errArchiveMismatch)
)

// statusCode is the HTTP status behind err, or 0 when err is not an HTTP status error.
func statusCode(err error) int {
	var status *httperror.HTTPError
	if errors.As(err, &status) {
		return status.StatusCode
	}
	return 0
}

// isPermanentError also covers a 406, which means this client's requests never get a chunk body.
func isPermanentError(err error) bool {
	return errors.Is(err, errArchiveMismatch) || statusCode(err) == http.StatusNotAcceptable
}

// reportBackfillError logs and counts a failed run step as transient or as permanent.
func reportBackfillError(msg string, err error, ctx ...interface{}) {
	ctx = append(ctx, "err", err)
	if isPermanentError(err) {
		backfillPermanentErrorsCounter.Inc(1)
		log.Error(msg, ctx...)
		return
	}
	backfillErrorsCounter.Inc(1)
	log.Warn(msg, ctx...)
}

type RestConfig struct {
	Enable  bool          `koanf:"enable"`
	URL     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout" reload:"hot"`
}

var DefaultRestConfig = RestConfig{
	Enable:  false,
	URL:     "",
	Timeout: 10 * time.Second,
}

func RestConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultRestConfig.Enable, "backfill feed gaps from the feed's REST chunk API, and advertise that capability when connecting to the feed")
	f.String(prefix+".url", DefaultRestConfig.URL, "base URL of the REST API serving the feed backlog, e.g. https://archive.example:9642; defaults to the host of the first feed url")
	f.Duration(prefix+".timeout", DefaultRestConfig.Timeout, "per-request timeout for feed backfill requests")
}

func (c *RestConfig) Validate() error {
	if !c.Enable {
		return nil
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("invalid feed backfill timeout %v, must be positive", c.Timeout)
	}
	if c.URL == "" {
		return nil
	}
	parsed, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid feed backfill url %v: %w", c.URL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid feed backfill url %v: scheme must be http or https", c.URL)
	}
	return nil
}

func restBaseURL(config *Config) (*url.URL, error) {
	if config.Rest.URL != "" {
		return url.Parse(config.Rest.URL)
	}
	if len(config.URL) == 0 || config.URL[0] == "" {
		return nil, errors.New("feed backfill needs a rest url or a feed url")
	}
	feedURL, err := url.Parse(config.URL[0])
	if err != nil {
		return nil, err
	}
	scheme, ok := map[string]string{"ws": "http", "wss": "https", "http": "http", "https": "https"}[feedURL.Scheme]
	if !ok {
		return nil, fmt.Errorf("cannot derive a feed backfill url from feed url %v", config.URL[0])
	}
	return &url.URL{Scheme: scheme, Host: feedURL.Host}, nil
}

// feedInfo is the /feed/v1/info body. It is static per process and advertises no range, so which
// chunks an archive holds is discovered by asking for them.
type feedInfo struct {
	ChainId     uint64 `json:"chainId"`
	FeedVersion int    `json:"feedVersion"`
	ChunkSize   uint64 `json:"chunkSize"`
}

// BackfillTransactionStreamerInterface is the streamer as the backfiller needs it: the head to
// fill from, and a write path for REST-fetched messages that leaves the queued live feed in place
// until it is reached. written counts only the given messages that were newly stored.
type BackfillTransactionStreamerInterface interface {
	GetMessageCount() (arbutil.MessageIndex, error)
	AddBroadcastBackfillMessages(feedMessages []*message.BroadcastFeedMessage) (written int, err error)
}

// Backfiller fills gaps between the node's message head and the live feed by reading chunks from
// the feed's REST API. Messages it applies are feed messages like any other: unconfirmed, verified
// the same way, and never able to reorg the chain.
type Backfiller struct {
	stopwaiter.StopWaiter

	config       ConfigFetcher
	chainId      uint64
	streamer     BackfillTransactionStreamerInterface
	latestSeqNum func() arbutil.MessageIndex
	sigVerifier  *signature.Verifier
	client       *http.Client
	baseURL      *url.URL
	trigger      chan struct{}
}

func NewBackfiller(
	config ConfigFetcher,
	chainId uint64,
	streamer BackfillTransactionStreamerInterface,
	addrVerifier contracts.AddressVerifierInterface,
	latestSeqNum func() arbutil.MessageIndex,
) (*Backfiller, error) {
	if err := config().Rest.Validate(); err != nil {
		return nil, err
	}
	baseURL, err := restBaseURL(config())
	if err != nil {
		return nil, err
	}
	sigVerifier, err := signature.NewVerifier(&config().Verify, addrVerifier)
	if err != nil {
		return nil, err
	}
	return &Backfiller{
		config:       config,
		chainId:      chainId,
		streamer:     streamer,
		latestSeqNum: latestSeqNum,
		sigVerifier:  sigVerifier,
		// Leave Accept-Encoding to the transport: it sends gzip (which the chunk endpoint
		// requires) and decodes the response. Setting it here returns undecoded bytes instead.
		client:  &http.Client{},
		baseURL: baseURL,
		trigger: make(chan struct{}, 1),
	}, nil
}

// Head is the node's message count, the point every run fills from.
func (b *Backfiller) Head() (arbutil.MessageIndex, error) {
	return b.streamer.GetMessageCount()
}

// Trigger asks for a backfill run. It never blocks: a run already pending absorbs the request.
func (b *Backfiller) Trigger() {
	select {
	case b.trigger <- struct{}{}:
	default:
	}
}

func (b *Backfiller) Start(ctxIn context.Context) {
	b.StopWaiter.Start(ctxIn, b)
	b.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-b.trigger:
			}
			// A gap is only reported on the message that opens it, so once one is seen the run
			// is retried on a timer until it closes, backing off while nothing lands.
			backoff := backfillPollInterval
			for {
				filled, progressed := b.runBackfill(ctx)
				if ctx.Err() != nil {
					return
				}
				if filled {
					break
				}
				if progressed {
					backoff = backfillPollInterval
				}
				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-b.trigger:
					timer.Stop()
					backoff = backfillPollInterval
				case <-timer.C:
					if !progressed {
						backoff = min(backoff*2, backfillMaxBackoff)
					}
				}
			}
		}
	})
}

// runBackfill applies chunks upward from the node's message head, each as it arrives so partial
// progress is durable. It reports whether there is nothing left to fill, and whether any chunk
// landed; the poll loop backs off only while nothing lands.
func (b *Backfiller) runBackfill(ctx context.Context) (filled bool, progressed bool) {
	backfillRunsCounter.Inc(1)

	info, err := b.fetchInfo(ctx)
	if err != nil {
		if ctx.Err() == nil {
			reportBackfillError("feed backfill could not read the archive info", err, "url", b.baseURL)
		}
		return false, false
	}

	var applied int
	for ctx.Err() == nil {
		head, err := b.streamer.GetMessageCount()
		if err != nil {
			if ctx.Err() == nil {
				log.Error("feed backfill cannot read the message count", "err", err)
				backfillErrorsCounter.Inc(1)
			}
			return false, applied > 0
		}
		target := b.latestSeqNum()
		if head > target {
			if applied > 0 {
				log.Info("feed backfill filled the gap", "chunks", applied, "head", head)
			}
			return true, applied > 0
		}

		start := uint64(head) - uint64(head)%info.ChunkSize
		messages, err := b.fetchChunk(ctx, info.ChunkSize, start)
		if err != nil {
			if errors.Is(err, errChunkMissing) {
				backfillChunksMissingCounter.Inc(1)
				// head against feedHead tells an operator whether this is the frontier or eviction.
				log.Info("feed backfill stopping, the archive does not have the next chunk",
					"start", start, "head", head, "feedHead", target, "chunksApplied", applied)
				return false, applied > 0
			}
			if ctx.Err() == nil {
				reportBackfillError("feed backfill could not read a chunk", err, "start", start, "head", head)
			}
			return false, applied > 0
		}

		written, err := b.streamer.AddBroadcastBackfillMessages(messages)
		switch {
		case errors.Is(err, TransactionStreamerBlockCreationStopped):
			// The node will not sync past its configured block, so there is nothing left to fill.
			log.Info("feed backfill stopping, the node has all the messages it will sync", "head", head)
			return true, applied > 0
		case errors.Is(err, ErrBackfillDiverged):
			reportBackfillError("feed backfill stopping, the archive serves a different history than the database", err, "start", start, "head", head)
			return false, applied > 0
		case errors.Is(err, ErrBackfillFeedReorgPending):
			log.Info("feed backfill waiting for the parent chain to settle a feed reorg", "head", head)
			return false, applied > 0
		case err != nil:
			if ctx.Err() == nil {
				log.Error("feed backfill could not add messages", "start", start, "head", head, "err", err)
				backfillErrorsCounter.Inc(1)
			}
			return false, applied > 0
		}
		if written == 0 {
			// Nothing in the chunk was new. Only another writer moving the head makes a retry
			// worthwhile; otherwise the same chunk would be refetched for the same answer.
			if newHead, err := b.streamer.GetMessageCount(); err == nil && newHead != head {
				continue
			}
			log.Warn("feed backfill is not advancing the message head, giving up this run",
				"start", start, "head", head, "feedHead", target, "chunksApplied", applied)
			return false, applied > 0
		}
		backfillChunksCounter.Inc(1)
		backfillMessagesCounter.Inc(int64(written))
		applied++
	}
	return false, applied > 0
}

func (b *Backfiller) fetchInfo(ctx context.Context) (*feedInfo, error) {
	endpoint := b.baseURL.JoinPath(feedInfoPath).String()
	body, err := b.get(ctx, endpoint)
	if err != nil {
		if statusCode(err) == http.StatusNotFound {
			// The archive always serves its info, so this url points at something else.
			return nil, fmt.Errorf("%w: %v: %w", errArchiveMismatch, endpoint, err)
		}
		return nil, err
	}
	var info feedInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("cannot parse the feed info: %w", err)
	}
	if info.ChainId != b.chainId {
		return nil, fmt.Errorf("%w: it serves chain %v, expected %v", errArchiveMismatch, info.ChainId, b.chainId)
	}
	if info.FeedVersion != message.V1 {
		return nil, fmt.Errorf("%w: it serves feed version %v, expected %v", errArchiveMismatch, info.FeedVersion, message.V1)
	}
	if info.ChunkSize == 0 || info.ChunkSize > maxChunkSize {
		return nil, fmt.Errorf("%w: it reports an unusable chunk size %v", errArchiveMismatch, info.ChunkSize)
	}
	return &info, nil
}

func (b *Backfiller) fetchChunk(ctx context.Context, chunkSize, start uint64) ([]*message.BroadcastFeedMessage, error) {
	body, err := b.get(ctx, b.baseURL.JoinPath(feedChunkPathPrefix+strconv.FormatUint(start, 10)).String())
	if err != nil {
		if statusCode(err) == http.StatusNotFound {
			return nil, errChunkMissing
		}
		return nil, err
	}
	var chunk message.BroadcastMessage
	if err := json.Unmarshal(body, &chunk); err != nil {
		return nil, fmt.Errorf("cannot parse chunk %v: %w", start, err)
	}
	if err := validateChunk(&chunk, start, chunkSize); err != nil {
		return nil, err
	}
	for _, msg := range chunk.Messages {
		if err := b.sigVerifier.VerifyHash(ctx, msg.Signature, msg.SignatureHash(b.chainId)); err != nil {
			return nil, fmt.Errorf("invalid signature at sequence number %v of chunk %v: %w", msg.SequenceNumber, start, err)
		}
	}
	return chunk.Messages, nil
}

func validateChunk(chunk *message.BroadcastMessage, start, chunkSize uint64) error {
	if chunk.Version != message.V1 {
		return fmt.Errorf("chunk %v has version %v, expected %v", start, chunk.Version, message.V1)
	}
	if uint64(len(chunk.Messages)) != chunkSize {
		return fmt.Errorf("chunk %v has %v messages, expected %v", start, len(chunk.Messages), chunkSize)
	}
	for i, msg := range chunk.Messages {
		if msg == nil {
			return fmt.Errorf("chunk %v has a nil message at offset %v", start, i)
		}
		if msg.Message.Message == nil || msg.Message.Message.Header == nil {
			return fmt.Errorf("chunk %v has an incomplete message at offset %v", start, i)
		}
		// #nosec G115
		if expected := start + uint64(i); uint64(msg.SequenceNumber) != expected {
			return fmt.Errorf("chunk %v has sequence number %v at offset %v, expected %v", start, msg.SequenceNumber, i, expected)
		}
	}
	return nil
}

// get fetches a body with a per-attempt timeout, retrying anything but a final status.
func (b *Backfiller) get(ctx context.Context, endpoint string) ([]byte, error) {
	for attempt := 1; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, b.config().Rest.Timeout)
		body, err := httpclient.Get(attemptCtx, b.client, endpoint)
		cancel()
		if err == nil || ctx.Err() != nil {
			return body, err
		}
		if code := statusCode(err); code != 0 && !retryableStatus(code) {
			return nil, err
		}
		if attempt >= backfillRequestAttempts {
			return nil, err
		}
		timer := time.NewTimer(backfillRetryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// retryableStatus is wider than HTTPError.IsRetryable: only the statuses the archive itself
// defines are final (a chunk it does not have, a misaligned start, a refused gzip contract), and
// anything else may come from a proxy in front of it, as in arb-relay's own client.
func retryableStatus(code int) bool {
	return code != http.StatusNotFound && code != http.StatusBadRequest && code != http.StatusNotAcceptable
}
