// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import (
	"context"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func testBlock(number uint64, parent common.Hash) *types.Block {
	return types.NewBlockWithHeader(&types.Header{Number: new(big.Int).SetUint64(number), ParentHash: parent, Time: number, GasLimit: 30_000_000})
}

func readWireFrame(t *testing.T, conn net.Conn) Frame {
	t.Helper()
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		t.Fatal(err)
	}
	length := binary.BigEndian.Uint32(header[16:20])
	data := append([]byte(nil), header...)
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		t.Fatal(err)
	}
	data = append(data, payload...)
	f, err := DecodeFrame(data, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestPublisherHelloAndBlockFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feed.sock")
	c := DefaultConfig
	c.Enable, c.SocketPath, c.ChainID = true, path, 42161
	p := NewPublisher(c)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.StopAndWait()
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ {
		conn, err = net.Dial("unix", path)
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if hello := readWireFrame(t, conn); hello.Kind != FrameHello || hello.Sequence != 1 {
		t.Fatalf("unexpected HELLO: %+v", hello)
	}
	b := testBlock(1, common.Hash{})
	p.TryPublish(b, types.Receipts{})
	if begin := readWireFrame(t, conn); begin.Kind != FrameBlockBegin {
		t.Fatalf("unexpected block begin: %+v", begin)
	}
	if end := readWireFrame(t, conn); end.Kind != FrameBlockEnd {
		t.Fatalf("unexpected block end: %+v", end)
	}
}

func TestPublisherQueueOverflowSetsGap(t *testing.T) {
	c := DefaultConfig
	c.Enable = true
	c.QueueSize = 16
	p := NewPublisher(c)
	p.enabled.Store(true)
	for i := uint64(1); i <= uint64(c.QueueSize)+1; i++ {
		p.TryPublish(testBlock(i, common.Hash{}), types.Receipts{})
	}
	if !p.stickyGap.Load() {
		t.Fatal("expected sticky gap after queue overflow")
	}
}

func TestPublisherTracksReorgFromObservedHead(t *testing.T) {
	c := DefaultConfig
	c.Enable = true
	p := NewPublisher(c)
	p.enabled.Store(true)
	first := testBlock(0, common.Hash{})
	second := testBlock(2, common.HexToHash("0x1234"))
	p.TryPublish(first, types.Receipts{})
	p.TryPublish(second, types.Receipts{})
	item := <-p.ingress
	if item.reorg != nil {
		t.Fatal("first observed block must not be a reorg")
	}
	item = <-p.ingress
	if item.reorg == nil || item.reorg.oldNum != 0 || item.reorg.newNum != 2 {
		t.Fatalf("expected parent/height mismatch reorg, got %+v", item.reorg)
	}
}
