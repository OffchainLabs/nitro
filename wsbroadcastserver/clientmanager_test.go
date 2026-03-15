// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package wsbroadcastserver

import (
	"bytes"
	"compress/flate"
	"encoding/json"
	"io"
	"net"
	"testing"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"

	m "github.com/offchainlabs/nitro/broadcaster/message"
)

// testClientConnection creates a ClientConnection with a real net.Conn for use
// in tests that call removeClientImpl (which calls StopOnly -> conn.Close).
func testClientConnection(compressed bool) *ClientConnection {
	server, _ := net.Pipe()
	return &ClientConnection{
		conn:        server,
		compression: compressed,
	}
}

// noopPoller implements netpoll.Poller for tests.
type noopPoller struct{}

func (noopPoller) Start(*netpoll.Desc, netpoll.CallbackFn) error { return nil }
func (noopPoller) Stop(*netpoll.Desc) error                      { return nil }
func (noopPoller) Resume(*netpoll.Desc) error                    { return nil }

func testBroadcastMessage() *m.BroadcastMessage {
	return &m.BroadcastMessage{
		Version: m.V1,
		Messages: []*m.BroadcastFeedMessage{
			{SequenceNumber: 1},
		},
	}
}

func TestSerializeMessageBothFormats(t *testing.T) {
	bm := testBroadcastMessage()
	notCompressed, compressed, err := serializeMessage(bm, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if notCompressed.Len() == 0 {
		t.Fatal("expected non-empty uncompressed output")
	}
	if compressed.Len() == 0 {
		t.Fatal("expected non-empty compressed output")
	}
	if compressed.Len() >= notCompressed.Len() {
		t.Fatalf("compressed (%d) should be smaller than uncompressed (%d)", compressed.Len(), notCompressed.Len())
	}

	// Verify uncompressed output deserializes correctly
	msgs, err := wsutil.ReadServerMessage(bytes.NewReader(notCompressed.Bytes()), nil)
	if err != nil {
		t.Fatal("failed to read uncompressed ws message:", err)
	}
	var decoded m.BroadcastMessage
	if err := json.Unmarshal(msgs[0].Payload, &decoded); err != nil {
		t.Fatal("failed to unmarshal uncompressed message:", err)
	}
	if len(decoded.Messages) != 1 || decoded.Messages[0].SequenceNumber != 1 {
		t.Fatalf("unexpected decoded message: %+v", decoded)
	}
}

func TestSerializeMessageOnlyUncompressed(t *testing.T) {
	bm := testBroadcastMessage()
	notCompressed, compressed, err := serializeMessage(bm, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if notCompressed.Len() == 0 {
		t.Fatal("expected non-empty uncompressed output")
	}
	if compressed.Len() != 0 {
		t.Fatal("expected empty compressed output")
	}
}

func TestSerializeMessageOnlyCompressed(t *testing.T) {
	bm := testBroadcastMessage()
	notCompressed, compressed, err := serializeMessage(bm, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if notCompressed.Len() != 0 {
		t.Fatal("expected empty uncompressed output")
	}
	if compressed.Len() == 0 {
		t.Fatal("expected non-empty compressed output")
	}
}

func TestSerializeMessageNeitherFormat(t *testing.T) {
	bm := testBroadcastMessage()
	notCompressed, compressed, err := serializeMessage(bm, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if notCompressed.Len() != 0 {
		t.Fatal("expected empty uncompressed output")
	}
	if compressed.Len() != 0 {
		t.Fatal("expected empty compressed output")
	}
}

func TestFlateWriterPoolReuse(t *testing.T) {
	bm := testBroadcastMessage()

	// Call serializeMessage multiple times with compression enabled.
	// The pool should reuse the flate writer without errors.
	_, firstCompressed, err := serializeMessage(bm, false, true)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes := append([]byte(nil), firstCompressed.Bytes()...)

	for i := 1; i < 10; i++ {
		_, compressed, err := serializeMessage(bm, false, true)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		// Verify the pool correctly resets state between uses: same input should
		// produce identical output on each sequential call from this goroutine.
		if !bytes.Equal(compressed.Bytes(), firstBytes) {
			t.Fatalf("iteration %d: compressed output differs from first iteration", i)
		}
	}
}

func TestSerializeMessageConsistency(t *testing.T) {
	bm := testBroadcastMessage()

	// Serialize with both formats enabled
	bothNC, bothC, err := serializeMessage(bm, true, true)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize with only uncompressed
	onlyNC, _, err := serializeMessage(bm, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize with only compressed
	_, onlyC, err := serializeMessage(bm, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// The uncompressed output should be identical regardless of whether
	// compression was also enabled
	if !bytes.Equal(bothNC.Bytes(), onlyNC.Bytes()) {
		t.Fatal("uncompressed output differs when compressed is also enabled")
	}

	// The compressed output should be identical regardless of whether
	// uncompressed was also enabled
	if !bytes.Equal(bothC.Bytes(), onlyC.Bytes()) {
		t.Fatal("compressed output differs when uncompressed is also enabled")
	}
}

func TestSerializeMessageValidWsFraming(t *testing.T) {
	bm := testBroadcastMessage()
	notCompressed, _, err := serializeMessage(bm, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// Read as websocket frame and verify it's a text opcode
	reader := bytes.NewReader(notCompressed.Bytes())
	header, err := ws.ReadHeader(reader)
	if err != nil {
		t.Fatal("failed to read ws header:", err)
	}
	if header.OpCode != ws.OpText {
		t.Fatalf("expected OpText, got %v", header.OpCode)
	}
	if !header.Fin {
		t.Fatal("expected Fin bit set")
	}
}

func TestSerializeMessageCompressedRoundTrip(t *testing.T) {
	bm := testBroadcastMessage()
	_, compressed, err := serializeMessage(bm, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// Read the websocket frame header and verify RSV1 (per-message deflate) is set
	reader := bytes.NewReader(compressed.Bytes())
	header, err := ws.ReadHeader(reader)
	if err != nil {
		t.Fatal("failed to read ws header:", err)
	}
	if header.OpCode != ws.OpText {
		t.Fatalf("expected OpText, got %v", header.OpCode)
	}
	if header.Rsv == 0 {
		t.Fatal("expected RSV bits set for compressed frame")
	}

	// Read the payload and decompress using the static dictionary.
	// PMCE requires appending the sync tail before decompressing.
	payload := make([]byte, header.Length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal("failed to read payload:", err)
	}
	syncTail := []byte{0x00, 0x00, 0xff, 0xff}
	flateReader := flate.NewReaderDict(
		io.MultiReader(bytes.NewReader(payload), bytes.NewReader(syncTail)),
		GetStaticCompressorDictionary(),
	)
	decompressed, err := io.ReadAll(flateReader)
	if err != nil {
		t.Fatal("failed to decompress:", err)
	}

	var decoded m.BroadcastMessage
	if err := json.Unmarshal(decompressed, &decoded); err != nil {
		t.Fatal("failed to unmarshal decompressed message:", err)
	}
	if len(decoded.Messages) != 1 || decoded.Messages[0].SequenceNumber != 1 {
		t.Fatalf("unexpected decoded message: %+v", decoded)
	}
}

func TestRemoveClientImplUnderflowProtection(t *testing.T) {
	config := &BroadcasterConfig{LogDisconnect: false}
	cm := &ClientManager{
		clientPtrMap: make(map[*ClientConnection]bool),
		config:       func() *BroadcasterConfig { return config },
		poller:       noopPoller{},
	}

	client := testClientConnection(true)
	cm.clientPtrMap[client] = true
	cm.compressedClientCount = 0 // Simulate a bug where count is already zero

	// removeClientImpl should clamp to 0 rather than going negative
	cm.removeClientImpl(client)
	if cm.compressedClientCount != 0 {
		t.Fatalf("expected compressedClientCount=0 after underflow protection, got %d", cm.compressedClientCount)
	}
}

func TestRemoveAllCompressedCountConsistency(t *testing.T) {
	config := &BroadcasterConfig{LogDisconnect: false}
	cm := &ClientManager{
		clientPtrMap: make(map[*ClientConnection]bool),
		config:       func() *BroadcasterConfig { return config },
		poller:       noopPoller{},
	}

	compressed1 := testClientConnection(true)
	compressed2 := testClientConnection(true)
	uncompressed := testClientConnection(false)

	cm.clientPtrMap[compressed1] = true
	cm.clientPtrMap[compressed2] = true
	cm.clientPtrMap[uncompressed] = true
	cm.compressedClientCount = 2

	cm.removeAll()

	if cm.compressedClientCount != 0 {
		t.Fatalf("expected compressedClientCount=0 after removeAll, got %d", cm.compressedClientCount)
	}
}

func TestSerializeMessageEmptyMessagesList(t *testing.T) {
	bm := &m.BroadcastMessage{
		Version:  m.V1,
		Messages: []*m.BroadcastFeedMessage{},
	}
	notCompressed, compressed, err := serializeMessage(bm, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if notCompressed.Len() == 0 {
		t.Fatal("expected non-empty uncompressed output for confirmation-only message")
	}
	if compressed.Len() == 0 {
		t.Fatal("expected non-empty compressed output for confirmation-only message")
	}
}

// TestCompressedClientCount verifies the counter logic for tracking compressed
// vs uncompressed clients, including the interaction with config flags that
// doBroadcast uses to decide which serialization formats are needed.
func TestCompressedClientCount(t *testing.T) {
	cm := &ClientManager{
		clientPtrMap: make(map[*ClientConnection]bool),
	}

	// checkNeeds mirrors the doBroadcast logic including config flags.
	checkNeeds := func(enableCompression, requireCompression, wantCompressed, wantNotCompressed bool, desc string) {
		t.Helper()
		totalClients := len(cm.clientPtrMap)
		needCompressed := cm.compressedClientCount > 0 && enableCompression
		needNotCompressed := cm.compressedClientCount < totalClients && !requireCompression
		if needCompressed != wantCompressed {
			t.Fatalf("%s: needCompressed=%v, want %v", desc, needCompressed, wantCompressed)
		}
		if needNotCompressed != wantNotCompressed {
			t.Fatalf("%s: needNotCompressed=%v, want %v", desc, needNotCompressed, wantNotCompressed)
		}
	}

	compressed1 := &ClientConnection{compression: true}
	compressed2 := &ClientConnection{compression: true}
	uncompressed := &ClientConnection{compression: false}

	// Register clients
	cm.clientPtrMap[compressed1] = true
	cm.compressedClientCount++
	cm.clientPtrMap[compressed2] = true
	cm.compressedClientCount++
	cm.clientPtrMap[uncompressed] = true

	if cm.compressedClientCount != 2 {
		t.Fatalf("expected compressedClientCount=2, got %d", cm.compressedClientCount)
	}

	// Mixed clients with compression enabled: need both formats
	checkNeeds(true, false, true, true, "mixed clients, compression enabled")

	// Mixed clients with compression disabled: only need uncompressed
	// (compressed clients will be disconnected by the loop)
	checkNeeds(false, false, false, true, "mixed clients, compression disabled")

	// Mixed clients with compression required: only need compressed
	// (uncompressed clients will be disconnected by the loop)
	checkNeeds(true, true, true, false, "mixed clients, compression required")

	// Remove uncompressed client
	delete(cm.clientPtrMap, uncompressed)
	checkNeeds(true, false, true, false, "all compressed clients")

	// All compressed but compression disabled: both false, clients will be disconnected
	checkNeeds(false, false, false, false, "all compressed, compression disabled")

	// Remove compressed clients
	delete(cm.clientPtrMap, compressed1)
	cm.compressedClientCount--
	delete(cm.clientPtrMap, compressed2)
	cm.compressedClientCount--
	checkNeeds(true, false, false, false, "no clients")
	if cm.compressedClientCount != 0 {
		t.Fatalf("expected compressedClientCount=0, got %d", cm.compressedClientCount)
	}
}
