// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package broadcastclient

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/broadcaster"
	"github.com/offchainlabs/nitro/broadcaster/message"
	"github.com/offchainlabs/nitro/util/httperror"
	"github.com/offchainlabs/nitro/util/signature"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func TestRestConfigValidate(t *testing.T) {
	t.Parallel()
	enabled := func(url string, timeout time.Duration) *RestConfig {
		return &RestConfig{Enable: true, URL: url, Timeout: timeout}
	}
	if err := enabled("http://archive.example", 0).Validate(); err == nil {
		t.Error("a non-positive timeout should be rejected when backfill is enabled")
	}
	Require(t, enabled("", time.Second).Validate())
	if err := enabled("ws://archive.example", time.Second).Validate(); err == nil {
		t.Error("a websocket url should be rejected")
	}
	Require(t, enabled("http://archive.example", time.Second).Validate())
	// A config built as a literal leaves these zero, which must stay valid while backfill is off.
	if err := (&RestConfig{}).Validate(); err != nil {
		t.Errorf("the zero value should be valid: %v", err)
	}
}

// mockArchive serves the feed's REST backlog API the way arb-relay does, so the client is tested
// against the real thing: gzipped bodies, gzip required, and cacheable 404s for both unsettled and
// evicted chunks.
type mockArchive struct {
	server *httptest.Server
	// chunks[start] holds the messages of a published chunk.
	chunks    map[uint64][]*message.BroadcastFeedMessage
	chainId   uint64
	chunkSize uint64
	// firstPublished is the lowest chunk this archive still holds.
	firstPublished uint64

	mu sync.Mutex
	// Set to serve a body verbatim instead of the real chunk, for malformed-input cases.
	badChunkBody []byte
	badInfoBody  []byte
	// chunkRequests counts well-formed chunk requests, whatever the answer.
	chunkRequests int
}

func newMockArchive(t *testing.T, chainId, chunkSize uint64) *mockArchive {
	t.Helper()
	a := &mockArchive{
		chunks:    make(map[uint64][]*message.BroadcastFeedMessage),
		chainId:   chainId,
		chunkSize: chunkSize,
	}
	a.server = httptest.NewServer(http.HandlerFunc(a.serve))
	t.Cleanup(a.server.Close)
	return a
}

func (a *mockArchive) url() string { return a.server.URL }

// publish stores the messages as chunks, which is what the archive will serve.
func (a *mockArchive) publish(messages []*message.BroadcastFeedMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for start := uint64(0); start+a.chunkSize <= uint64(len(messages)); start += a.chunkSize {
		a.chunks[start] = messages[start : start+a.chunkSize]
	}
}

// evictBelow drops the chunks before start, as the archive does once they finalize.
func (a *mockArchive) evictBelow(start uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for published := range a.chunks {
		if published < start {
			delete(a.chunks, published)
		}
	}
	a.firstPublished = start
}

func (a *mockArchive) chunkRequestCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chunkRequests
}

func (a *mockArchive) setChainId(chainId uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chainId = chainId
}

func (a *mockArchive) setBadInfoBody(body []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.badInfoBody = body
}

func (a *mockArchive) setBadChunkBody(body []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.badChunkBody = body
}

func (a *mockArchive) serve(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	chainId, badInfoBody, badChunkBody := a.chainId, a.badInfoBody, a.badChunkBody
	a.mu.Unlock()

	if r.URL.Path == feedInfoPath {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if badInfoBody != nil {
			_, _ = w.Write(badInfoBody)
			return
		}
		_ = json.NewEncoder(w).Encode(feedInfo{
			ChainId:     chainId,
			FeedVersion: message.V1,
			ChunkSize:   a.chunkSize,
		})
		return
	}

	rawStart, ok := strings.CutPrefix(r.URL.Path, feedChunkPathPrefix)
	if !ok {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	start, err := strconv.ParseUint(rawStart, 10, 64)
	if err != nil || strconv.FormatUint(start, 10) != rawStart || start%a.chunkSize != 0 {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	a.chunkRequests++
	a.mu.Unlock()
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	a.mu.Lock()
	messages, published := a.chunks[start]
	firstPublished := a.firstPublished
	a.mu.Unlock()
	if !published {
		// The two classes differ only in cache max-age, as on arb-relay; the client never reads it
		// and the contract does not promise it.
		if start < firstPublished {
			w.Header().Set("Cache-Control", "public, max-age=2")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=1")
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var body []byte
	if badChunkBody != nil {
		body = badChunkBody
	} else {
		body, err = json.Marshal(message.BroadcastMessage{Version: message.V1, Messages: messages})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Cache-Control", "public, max-age=600, immutable")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	_, _ = gz.Write(body)
}

// backfillStreamer is a stand-in for the transaction streamer: it accepts contiguous messages and
// reports a head that advances as they land.
type backfillStreamer struct {
	mu       sync.Mutex
	messages []*message.BroadcastFeedMessage
	head     arbutil.MessageIndex
	// refuse makes every backfill write a silent no-op; failWith makes it fail with that error.
	refuse   bool
	failWith error
}

func (s *backfillStreamer) GetMessageCount() (arbutil.MessageIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.head, nil
}

func (s *backfillStreamer) AddBroadcastBackfillMessages(feedMessages []*message.BroadcastFeedMessage) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return 0, s.failWith
	}
	if s.refuse {
		return 0, nil
	}
	var written int
	for _, msg := range feedMessages {
		if msg.SequenceNumber < s.head {
			continue // already stored, as the real streamer trims duplicates
		}
		if msg.SequenceNumber != s.head {
			return written, fmt.Errorf("message %v is not contiguous with head %v", msg.SequenceNumber, s.head)
		}
		s.messages = append(s.messages, msg)
		s.head++
		written++
	}
	return written, nil
}

func (s *backfillStreamer) getMessages() []*message.BroadcastFeedMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*message.BroadcastFeedMessage{}, s.messages...)
}

func (s *backfillStreamer) accept() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refuse = false
}

func (s *backfillStreamer) setHead(head arbutil.MessageIndex) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head = head
}

// signedMessages builds count feed messages signed the way the sequencer signs them.
func signedMessages(t *testing.T, chainId uint64, count int) ([]*message.BroadcastFeedMessage, *signature.VerifierConfig) {
	t.Helper()
	privateKey, err := crypto.GenerateKey()
	Require(t, err)
	sequencerAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	dataSigner := signature.DataSignerFromPrivateKey(privateKey)

	broadcasterConfig := wsbroadcastserver.DefaultTestBroadcasterConfig
	b := broadcaster.NewBroadcaster(func() *wsbroadcastserver.BroadcasterConfig { return &broadcasterConfig }, chainId, make(chan error, 10), dataSigner)

	messages := make([]*message.BroadcastFeedMessage, 0, count)
	for i := 0; i < count; i++ {
		msg, err := b.NewBroadcastFeedMessage(arbostypes.MessageWithMetadataAndBlockInfo{
			MessageWithMeta: arbostypes.EmptyTestMessageWithMetadata,
			BlockHash:       nil,
			BlockMetadata:   nil,
		}, arbutil.MessageIndex(i)) // #nosec G115
		Require(t, err)
		messages = append(messages, msg)
	}

	// Testing config rather than the default: the default accepts any signature when there is no
	// address verifier to consult, which would defeat the point of checking them here.
	verifierConfig := signature.TestingFeedVerifierConfig
	verifierConfig.AllowedAddresses = []string{sequencerAddr.Hex()}
	return messages, &verifierConfig
}

func newTestBackfiller(t *testing.T, chainId uint64, archiveURL string, verify *signature.VerifierConfig, streamer BackfillTransactionStreamerInterface, target func() arbutil.MessageIndex) *Backfiller {
	t.Helper()
	config := DefaultTestConfig
	config.Verify = *verify
	config.Rest.Enable = true
	config.Rest.URL = archiveURL
	backfiller, err := NewBackfiller(func() *Config { return &config }, chainId, streamer, nil, target)
	Require(t, err)
	return backfiller
}

// lastSeqNum is the feed head every test sits at: the sequence number of the last published message.
func lastSeqNum(messages []*message.BroadcastFeedMessage) func() arbutil.MessageIndex {
	last := messages[len(messages)-1].SequenceNumber
	return func() arbutil.MessageIndex { return last }
}

// runOnce runs a single pass outside the poll loop and checks what it reported and how many chunk
// requests it cost, so a run that should end is proven to end rather than assumed from a sleep.
func runOnce(t *testing.T, ctx context.Context, backfiller *Backfiller, archive *mockArchive, wantFilled, wantProgressed bool, wantChunkRequests int) {
	t.Helper()
	filled, progressed := backfiller.runBackfill(ctx)
	if filled != wantFilled || progressed != wantProgressed {
		t.Fatalf("run reported filled=%v progressed=%v, expected filled=%v progressed=%v", filled, progressed, wantFilled, wantProgressed)
	}
	if got := archive.chunkRequestCount(); got != wantChunkRequests {
		t.Fatalf("archive served %v chunk requests, expected %v", got, wantChunkRequests)
	}
}

func TestBackfillerFillsGap(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 20)
	archive := newMockArchive(t, chainId, 5)
	archive.publish(messages)

	streamer := &backfillStreamer{}
	// The feed head sits at the last message, so everything below it is a gap.
	backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))
	backfiller.Start(ctx)
	defer backfiller.StopAndWait()
	backfiller.Trigger()

	awaitHead(t, streamer, arbutil.MessageIndex(len(messages)), 5*time.Second)
	for i, msg := range streamer.getMessages() {
		if msg.SequenceNumber != messages[i].SequenceNumber {
			t.Fatalf("message %v has sequence number %v", i, msg.SequenceNumber)
		}
	}
}

// TestBackfillerAlignsMidChunkHead pins the chunk alignment: a head inside a chunk fetches that
// chunk from its start, and the part already stored is skipped rather than rewritten.
func TestBackfillerAlignsMidChunkHead(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 20)
	archive := newMockArchive(t, chainId, 5)
	archive.publish(messages)

	// The node holds messages 0 to 5, so the first chunk it needs starts below its head.
	streamer := &backfillStreamer{head: 6}
	backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

	// Chunks 5, 10, and 15 close the gap; chunk 0 is never asked for.
	runOnce(t, ctx, backfiller, archive, true, true, 3)
	stored := streamer.getMessages()
	if len(stored) != 14 || stored[0].SequenceNumber != 6 {
		t.Fatalf("stored %v messages starting at %v, expected 14 starting at 6", len(stored), stored[0].SequenceNumber)
	}
}

func TestBackfillerStopsOnMissingChunk(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 20)
	archive := newMockArchive(t, chainId, 5)
	// Only the first two chunks are published; the rest is past the archive's frontier.
	archive.publish(messages[:10])

	streamer := &backfillStreamer{}
	backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

	// Two chunks land, the third is a 404, and the run ends there having made progress.
	runOnce(t, ctx, backfiller, archive, false, true, 3)
	if head, _ := streamer.GetMessageCount(); head != 10 {
		t.Fatalf("head is %v, expected the run to stop at the archive's frontier", head)
	}

	// Once the archive publishes more, the pending run picks it up without a fresh trigger.
	backfiller.Start(ctx)
	defer backfiller.StopAndWait()
	backfiller.Trigger()
	archive.publish(messages)
	awaitHead(t, streamer, arbutil.MessageIndex(len(messages)), 10*time.Second)
}

func TestBackfillerEvictedChunkEndsRun(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 20)
	archive := newMockArchive(t, chainId, 5)
	archive.publish(messages)
	// The node is behind what the archive still holds.
	archive.evictBelow(10)

	streamer := &backfillStreamer{}
	backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

	// The first chunk the node needs is gone, so the run ends after one request with nothing applied.
	runOnce(t, ctx, backfiller, archive, false, false, 1)
	if head, _ := streamer.GetMessageCount(); head != 0 {
		t.Fatalf("head is %v, expected no messages from an evicted range", head)
	}

	// When the node catches up into the retained window, backfill resumes.
	streamer.setHead(10)
	backfiller.Start(ctx)
	defer backfiller.StopAndWait()
	backfiller.Trigger()
	awaitHead(t, streamer, arbutil.MessageIndex(len(messages)), 10*time.Second)
}

// TestBackfillerRejectsBadArchive pins that an unusable info body fails the run before any chunk is
// asked for, and which of those failures are permanent rather than transient.
func TestBackfillerRejectsBadArchive(t *testing.T) {
	t.Parallel()
	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 10)

	for _, tc := range []struct {
		name      string
		setup     func(a *mockArchive)
		permanent bool
	}{
		{
			name:      "wrong chain id",
			setup:     func(a *mockArchive) { a.setChainId(chainId + 1) },
			permanent: true,
		},
		{
			name: "wrong feed version",
			setup: func(a *mockArchive) {
				a.setBadInfoBody([]byte(fmt.Sprintf(`{"chainId":%d,"feedVersion":2,"chunkSize":5}`, chainId)))
			},
			permanent: true,
		},
		{
			name: "zero chunk size",
			setup: func(a *mockArchive) {
				a.setBadInfoBody([]byte(fmt.Sprintf(`{"chainId":%d,"feedVersion":1,"chunkSize":0}`, chainId)))
			},
			permanent: true,
		},
		{
			name: "absurd chunk size",
			setup: func(a *mockArchive) {
				a.setBadInfoBody([]byte(fmt.Sprintf(`{"chainId":%d,"feedVersion":1,"chunkSize":100000000}`, chainId)))
			},
			permanent: true,
		},
		{
			name:      "unparsable info body",
			setup:     func(a *mockArchive) { a.setBadInfoBody([]byte("not json")) },
			permanent: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			archive := newMockArchive(t, chainId, 5)
			archive.publish(messages)
			tc.setup(archive)

			streamer := &backfillStreamer{}
			backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

			_, err := backfiller.fetchInfo(ctx)
			if err == nil {
				t.Fatal("expected the info fetch to fail")
			}
			if got := isPermanentError(err); got != tc.permanent {
				t.Fatalf("isPermanentError = %v, expected %v (err: %v)", got, tc.permanent, err)
			}

			// The info check fails, so the run ends before asking for any chunk.
			runOnce(t, ctx, backfiller, archive, false, false, 0)
			if head, _ := streamer.GetMessageCount(); head != 0 {
				t.Fatalf("head is %v, expected nothing to be applied from an unusable archive", head)
			}
		})
	}
}

// TestBackfillerBaseURL pins how the configured base URL is used: paths are joined onto it, so a
// trailing slash does not become a double slash, and a 404 on the info path is reported as the
// status it is rather than as a missing chunk, so a wrong prefix reads as a wrong url.
func TestBackfillerBaseURL(t *testing.T) {
	t.Parallel()
	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 10)

	for _, tc := range []struct {
		name       string
		suffix     string
		wantStatus int // 0 expects success
	}{
		{name: "a trailing slash is joined", suffix: "/"},
		{name: "a wrong path prefix is a 404, not a missing chunk", suffix: "/relay", wantStatus: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			archive := newMockArchive(t, chainId, 5)
			archive.publish(messages)
			backfiller := newTestBackfiller(t, chainId, archive.url()+tc.suffix, verify, &backfillStreamer{}, lastSeqNum(messages))

			_, err := backfiller.fetchInfo(ctx)
			if tc.wantStatus == 0 {
				Require(t, err)
				return
			}
			if errors.Is(err, errChunkMissing) {
				t.Fatalf("info 404 was reported as a missing chunk: %v", err)
			}
			var status *httperror.HTTPError
			if !errors.As(err, &status) || status.StatusCode != tc.wantStatus {
				t.Fatalf("expected a %v status error, got %v", tc.wantStatus, err)
			}
		})
	}
}

// TestHTTPStatusPolicy pins how each status is handled: whether a request is retried, and whether a
// final answer is permanent. Only the archive's own final answers skip the retry budget, and of
// those only a refused gzip encoding is permanent.
func TestHTTPStatusPolicy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code      int
		retryable bool
		permanent bool
	}{
		{code: http.StatusNotFound, retryable: false, permanent: false},
		{code: http.StatusBadRequest, retryable: false, permanent: false},
		{code: http.StatusNotAcceptable, retryable: false, permanent: true},
		{code: http.StatusForbidden, retryable: true, permanent: false},
		{code: http.StatusRequestTimeout, retryable: true, permanent: false},
		{code: http.StatusTooManyRequests, retryable: true, permanent: false},
		{code: http.StatusInternalServerError, retryable: true, permanent: false},
		{code: http.StatusBadGateway, retryable: true, permanent: false},
	} {
		if got := retryableStatus(tc.code); got != tc.retryable {
			t.Errorf("status %v: retryable = %v, expected %v", tc.code, got, tc.retryable)
		}
		if got := isPermanentError(&httperror.HTTPError{StatusCode: tc.code}); got != tc.permanent {
			t.Errorf("status %v: isPermanentError = %v, expected %v", tc.code, got, tc.permanent)
		}
	}
	if isPermanentError(errChunkMissing) {
		t.Error("a missing chunk must not count as permanent")
	}
}

func TestBackfillerRejectsBadChunk(t *testing.T) {
	t.Parallel()
	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 10)
	otherMessages, _ := signedMessages(t, chainId, 10)

	marshal := func(t *testing.T, msg message.BroadcastMessage) []byte {
		t.Helper()
		body, err := json.Marshal(msg)
		Require(t, err)
		return body
	}

	for _, tc := range []struct {
		name    string
		body    func(t *testing.T) []byte
		wantErr string
	}{
		{
			name: "wrong version",
			body: func(t *testing.T) []byte {
				return marshal(t, message.BroadcastMessage{Version: 2, Messages: messages[:5]})
			},
			wantErr: "has version 2, expected 1",
		},
		{
			name: "short chunk",
			body: func(t *testing.T) []byte {
				return marshal(t, message.BroadcastMessage{Version: message.V1, Messages: messages[:3]})
			},
			wantErr: "has 3 messages, expected 5",
		},
		{
			name: "chunk shifted from the requested start",
			body: func(t *testing.T) []byte {
				return marshal(t, message.BroadcastMessage{Version: message.V1, Messages: messages[1:6]})
			},
			wantErr: "has sequence number 1 at offset 0, expected 0",
		},
		{
			name: "signed by someone else",
			body: func(t *testing.T) []byte {
				return marshal(t, message.BroadcastMessage{Version: message.V1, Messages: otherMessages[:5]})
			},
			wantErr: "invalid signature at sequence number 0",
		},
		{
			name: "nil message",
			body: func(t *testing.T) []byte {
				return []byte(`{"version":1,"messages":[null,null,null,null,null]}`)
			},
			wantErr: "has a nil message at offset 0",
		},
		{
			name: "nil inner message",
			body: func(t *testing.T) []byte {
				one := `{"sequenceNumber":%d,"message":{"message":null,"delayedMessagesRead":0},"signatureV2":null}`
				parts := make([]string, 5)
				for i := range parts {
					parts[i] = fmt.Sprintf(one, i)
				}
				return []byte(`{"version":1,"messages":[` + strings.Join(parts, ",") + `]}`)
			},
			wantErr: "has an incomplete message at offset 0",
		},
		{
			name: "nil header",
			body: func(t *testing.T) []byte {
				one := `{"sequenceNumber":%d,"message":{"message":{"header":null,"l2Msg":""},"delayedMessagesRead":0},"signatureV2":null}`
				parts := make([]string, 5)
				for i := range parts {
					parts[i] = fmt.Sprintf(one, i)
				}
				return []byte(`{"version":1,"messages":[` + strings.Join(parts, ",") + `]}`)
			},
			wantErr: "has an incomplete message at offset 0",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			archive := newMockArchive(t, chainId, 5)
			archive.publish(messages)
			archive.setBadChunkBody(tc.body(t))

			streamer := &backfillStreamer{}
			backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

			// A malformed chunk ends the run after one request, is not applied, and must not panic.
			runOnce(t, ctx, backfiller, archive, false, false, 1)
			if head, _ := streamer.GetMessageCount(); head != 0 {
				t.Fatalf("head is %v, expected a malformed chunk to be rejected", head)
			}
			// And it is rejected for the reason the row describes.
			if _, err := backfiller.fetchChunk(ctx, 5, 0); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got %v, expected an error containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestBackfillerGivesUpWhenNothingLands covers the no-progress guard: if a chunk lands nothing and
// the head has not moved, the run ends rather than refetching the same chunk. The run is called
// directly so the chunk count is exact and no timer is involved.
func TestBackfillerGivesUpWhenNothingLands(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 20)
	archive := newMockArchive(t, chainId, 5)
	archive.publish(messages)

	streamer := &backfillStreamer{refuse: true}
	backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))

	runOnce(t, ctx, backfiller, archive, false, false, 1)
	if head, _ := streamer.GetMessageCount(); head != 0 {
		t.Fatalf("head is %v, expected the refusing streamer to store nothing", head)
	}

	// Once the streamer accepts, the next run starts over from the head and fills the gap.
	streamer.accept()
	backfiller.Start(ctx)
	defer backfiller.StopAndWait()
	backfiller.Trigger()
	awaitHead(t, streamer, arbutil.MessageIndex(len(messages)), 10*time.Second)
}

// TestBackfillerEndsRunOnStreamerVerdict pins that a typed refusal from the streamer ends the run
// after a single fetch, and that a node which will not sync further treats the gap as closed.
func TestBackfillerEndsRunOnStreamerVerdict(t *testing.T) {
	t.Parallel()
	const chainId = uint64(9742)
	messages, verify := signedMessages(t, chainId, 10)

	for _, tc := range []struct {
		name       string
		err        error
		wantFilled bool
	}{
		{name: "sync target reached", err: TransactionStreamerBlockCreationStopped, wantFilled: true},
		{name: "archive diverged", err: ErrBackfillDiverged},
		{name: "feed reorg pending", err: ErrBackfillFeedReorgPending},
		{name: "other error", err: errors.New("database closed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			archive := newMockArchive(t, chainId, 5)
			archive.publish(messages)
			streamer := &backfillStreamer{failWith: tc.err}
			backfiller := newTestBackfiller(t, chainId, archive.url(), verify, streamer, lastSeqNum(messages))
			runOnce(t, ctx, backfiller, archive, tc.wantFilled, false, 1)
		})
	}
}

func TestRestBaseURL(t *testing.T) {
	t.Parallel()
	config := DefaultTestConfig
	config.URL = []string{"wss://feed.example:9642/feed?x=1"}
	got, err := restBaseURL(&config)
	Require(t, err)
	if got.String() != "https://feed.example:9642" {
		t.Errorf("derived %v", got)
	}
	config.URL = []string{"tcp://feed.example"}
	if _, err := restBaseURL(&config); err == nil {
		t.Error("expected a feed url with an unknown scheme to be rejected")
	}
}

func TestBackfillCapabilityHeader(t *testing.T) {
	t.Parallel()
	for _, advertise := range []bool{true, false} {
		t.Run(fmt.Sprintf("advertise=%v", advertise), func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			headers := make(chan http.Header, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case headers <- r.Header.Clone():
				default:
				}
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()

			config := DefaultTestConfig
			client, err := NewBroadcastClient(
				func() *Config { return &config },
				strings.Replace(server.URL, "http://", "ws://", 1),
				9742, 0, &accumulatingTransactionStreamer{}, nil, make(chan error, 10), nil,
				func(_ int32) {},
				advertise,
			)
			Require(t, err)
			client.Start(ctx)
			defer client.StopAndWait()

			select {
			case header := <-headers:
				value := header.Get(wsbroadcastserver.HTTPHeaderFeedBackfill)
				if advertise && value != feedBackfillHeaderValue {
					t.Errorf("backfill header is %q, expected %q", value, feedBackfillHeaderValue)
				}
				if !advertise && value != "" {
					t.Errorf("backfill header is %q, expected it to be absent", value)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the handshake")
			}
		})
	}
}

func awaitHead(t *testing.T, streamer *backfillStreamer, head arbutil.MessageIndex, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		current, err := streamer.GetMessageCount()
		Require(t, err)
		if current >= head {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for head %v, got %v", head, current)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
