// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/arbutil"
	"strings"
	"testing"
)

func newMockTransactionStreamer(bc *mockBlockChain, db *mockDB, queuePos arbutil.MessageIndex, queue []arbstate.MessageWithMetadata) *TransactionStreamer {
	return &TransactionStreamer{
		bc:                           bc,
		db:                           db,
		broadcasterQueuedMessagesPos: uint64(queuePos),
		broadcasterQueuedMessages:    queue,
	}
}

func TestSkipDupEmpty(t *testing.T) {
	t.Parallel()

	db := newMockDB()
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)

	// Test empty
	var newMessages []arbstate.MessageWithMetadata
	reorg, prevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(10, 15, newMessages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have been a reorg")
	}
	if prevDelayedRead != 10 {
		t.Error("incorrect prevDelayedRead")
	}
	if pos != 15 {
		t.Error("incorrect pos")
	}
	if len(dedupMessages) != 0 {
		t.Error("incorrect message size")
	}
}

func TestSkipDupEmptyWithSingle(t *testing.T) {
	t.Parallel()

	db := newMockDB()
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)

	// Test empty with message
	prevDelayedRead := uint64(30)
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedRead,
		},
	}
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(prevDelayedRead, 15, newMessages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have been a reorg")
	}
	if newPrevDelayedRead != prevDelayedRead {
		t.Error("incorrect prevDelayedRead")
	}
	if pos != 15 {
		t.Error("incorrect pos")
	}
	if len(dedupMessages) != 1 {
		t.Error("incorrect message size")
	}
}

func addDBMessages(db *mockDB, pos arbutil.MessageIndex, messages []arbstate.MessageWithMetadata) error {
	currentPos := pos
	for _, msg := range messages {
		msgBytes, err := rlp.EncodeToBytes(msg)
		if err != nil {
			return err
		}
		err = db.Put(dbKey(messagePrefix, uint64(currentPos)), msgBytes)
		if err != nil {
			return err
		}
		currentPos++
	}

	return nil
}

func TestSkipDupNoMatch(t *testing.T) {
	t.Parallel()

	// Test dup message
	currentPos := arbutil.MessageIndex(10)
	prevDelayedRead := uint64(30)
	messages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedRead,
		},
	}
	db := newMockDB()
	err := addDBMessages(db, currentPos, messages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(prevDelayedRead, currentPos+1, messages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have been a reorg")
	}
	if newPrevDelayedRead != prevDelayedRead {
		t.Error("incorrect prevDelayedRead")
	}
	if pos != currentPos+1 {
		t.Error("incorrect pos")
	}
	if len(dedupMessages) != 1 {
		t.Error("incorrect message size")
	}
}

func TestSkipDupMatch(t *testing.T) {
	t.Parallel()

	// Test dup message
	currentPos := arbutil.MessageIndex(10)
	prevDelayedRead := uint64(30)
	messages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedRead,
		},
	}
	db := newMockDB()
	err := addDBMessages(db, currentPos, messages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(prevDelayedRead, currentPos, messages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have reorg")
	}
	if newPrevDelayedRead != prevDelayedRead {
		t.Error("unexpected prevDelayedRead")
	}
	if pos != currentPos+1 {
		t.Error("unexpected pos")
	}
	if len(dedupMessages) != 0 {
		t.Error("incorrect message size")
	}
}

func TestSkipDupMatchTwo(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(10)
	prevDelayedRead := uint64(30)
	messages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedRead,
		},
		{
			DelayedMessagesRead: prevDelayedRead + 1,
		},
	}
	dbPrevDelayedRead := messages[len(messages)-1].DelayedMessagesRead
	// Just put first item into db
	db := newMockDB()
	err := addDBMessages(db, firstPos, messages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(prevDelayedRead, firstPos, messages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have reorg")
	}
	if newPrevDelayedRead != dbPrevDelayedRead {
		t.Error("unexpected prevDelayedRead")
	}
	if pos != firstPos+2 {
		t.Error("unexpected pos")
	}
	if len(dedupMessages) != 0 {
		t.Error("incorrect message size")
	}
	dbCheckPos(t, ts.db, firstPos)
}

func dbCheckPos(t *testing.T, db ethdb.Database, pos arbutil.MessageIndex) {
	t.Helper()

	ok, err := db.Has(dbKey(messagePrefix, uint64(pos)))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Error("missing expected message")
	}
}

func TestSkipDupMatchFirst(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(10)
	prevDelayedRead := uint64(30)
	messages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedRead,
		},
		{
			DelayedMessagesRead: prevDelayedRead + 1,
		},
	}
	// Just put first item into db
	db := newMockDB()
	err := addDBMessages(db, firstPos, messages[:1])
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(prevDelayedRead, firstPos, messages, false)
	if err != nil {
		t.Fatal(err)
	}
	if reorg {
		t.Error("should not have reorg")
	}
	if newPrevDelayedRead != prevDelayedRead {
		t.Error("unexpected prevDelayedRead")
	}
	if pos != firstPos+1 {
		t.Error("unexpected pos")
	}
	if len(dedupMessages) != 1 {
		t.Error("incorrect message size")
	}
	dbCheckPos(t, ts.db, firstPos)
}

func TestSkipDupMatchFirstReorgSecond(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(10)
	messages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: 0,
		},
		{
			DelayedMessagesRead: 1,
		},
	}
	messagesReorg := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: 0,
		},
		{
			DelayedMessagesRead: 2,
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, messages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	reorg, newPrevDelayedRead, pos, dedupMessages, err := ts.skipDuplicateMessages(0, firstPos, messagesReorg, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reorg {
		t.Error("should have reorg")
	}
	if newPrevDelayedRead != 0 {
		t.Error("unexpected prevDelayedRead")
	}
	if pos != firstPos+1 {
		t.Error("unexpected pos")
	}
	if len(dedupMessages) != 1 {
		t.Error("incorrect message size")
	}
	dbCheckPos(t, ts.db, firstPos)
}

func TestAddMessagesNilMessage(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos, true, newMessages, nil)
	if err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "attempted to insert nil message") {
		t.Fatalf("incorrect error returned: %s", err.Error())
	}
}

func TestAddMessagesMissingPrev(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(2)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos, true, newMessages, nil)
	if err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "failed to get previous message") {
		t.Fatalf("incorrect error returned: %s", err.Error())
	}
}

func TestAddMessagesReorgInitMessage(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos, true, newMessages, nil)
	if err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "cannot reorg out init message") {
		t.Fatalf("incorrect error returned: %s", err.Error())
	}
}

func TestAddMessagesJump(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 2,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos+1, true, newMessages, nil)
	if err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "attempted to insert jump") {
		t.Fatalf("incorrect error returned: %s", err.Error())
	}
}

func TestAddMessagesReorgNotAllowed(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos, false, newMessages, nil)
	if err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "reorg required but not allowed") {
		t.Fatalf("incorrect error returned: %s", err.Error())
	}
}

func TestAddMessagesReorg(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
				L2msg:  []byte{42},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
				L2msg:  []byte{43},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos, true, newMessages, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddMessagesBatchMatch(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, 0, nil)
	err = ts.addMessagesAndEndBatchImpl(firstPos+1, true, newMessages, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(db.messages) != 2 {
		t.Error("unexpected message length")
	}
	dbCheckPos(t, ts.db, firstPos)
	dbCheckPos(t, ts.db, firstPos+1)
}

func TestAddMessagesBatchMatchPlusCache(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	queueMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 2,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 3,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, firstPos+1, queueMessages)
	err = ts.addMessagesAndEndBatchImpl(firstPos+1, false, newMessages, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 4 messages plus messageCount
	if len(db.messages) != 5 {
		t.Error("unexpected message length")
	}
	dbCheckPos(t, ts.db, firstPos)
	dbCheckPos(t, ts.db, firstPos+1)
	dbCheckPos(t, ts.db, firstPos+2)
	dbCheckPos(t, ts.db, firstPos+3)
}
func TestAddBroadcastMessagesEmpty(t *testing.T) {
	t.Parallel()

	// Test dup message
	firstPos := arbutil.MessageIndex(0)
	prevDelayedMessagesRead := uint64(0)
	dbMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	queueMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 2,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 3,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	newMessages := []arbstate.MessageWithMetadata{
		{
			DelayedMessagesRead: prevDelayedMessagesRead + 1,
			Message: &arbos.L1IncomingMessage{
				Header: &arbos.L1IncomingMessageHeader{},
			},
		},
	}
	db := newMockDB()
	err := addDBMessages(db, firstPos, dbMessages)
	if err != nil {
		t.Fatal(err)
	}
	ts := newMockTransactionStreamer(newMockBlockChain(), db, firstPos+1, queueMessages)
	err = ts.addMessagesAndEndBatchImpl(firstPos+1, false, newMessages, nil)
	if err != nil {
		t.Fatal(err)
	}

	// 4 messages plus messageCount
	if len(db.messages) != 5 {
		t.Error("unexpected message length")
	}
	dbCheckPos(t, ts.db, firstPos)
	dbCheckPos(t, ts.db, firstPos+1)
	dbCheckPos(t, ts.db, firstPos+2)
	dbCheckPos(t, ts.db, firstPos+3)

}
