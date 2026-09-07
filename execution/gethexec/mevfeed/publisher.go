// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package mevfeed

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"hash/crc32"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	enqueuedCounter          = metrics.NewRegisteredCounter("arb/mevfeed/enqueued", nil)
	droppedCounter           = metrics.NewRegisteredCounter("arb/mevfeed/dropped", nil)
	gapsCounter              = metrics.NewRegisteredCounter("arb/mevfeed/gaps", nil)
	encodedFramesCounter     = metrics.NewRegisteredCounter("arb/mevfeed/encoded_frames", nil)
	encodedBytesCounter      = metrics.NewRegisteredCounter("arb/mevfeed/encoded_bytes", nil)
	encodeErrorsCounter      = metrics.NewRegisteredCounter("arb/mevfeed/encode_errors", nil)
	writeErrorsCounter       = metrics.NewRegisteredCounter("arb/mevfeed/write_errors", nil)
	writeTimeoutsCounter     = metrics.NewRegisteredCounter("arb/mevfeed/write_timeouts", nil)
	clientConnectsCounter    = metrics.NewRegisteredCounter("arb/mevfeed/client_connects", nil)
	clientDisconnectsCounter = metrics.NewRegisteredCounter("arb/mevfeed/client_disconnects", nil)
	queueDepthGauge          = metrics.NewRegisteredGauge("arb/mevfeed/queue_depth", nil)
)

type blockItem struct {
	block    *types.Block
	receipts types.Receipts
	reorg    *reorgItem
}
type reorgItem struct {
	oldNum, newNum           uint64
	oldHash, newHash, parent common.Hash
}

// Publisher owns the bounded ingress queue, encoder worker and one-consumer Unix
// server. TryPublish performs only a channel send and small cursor bookkeeping.
type Publisher struct {
	config           Config
	ingress          chan blockItem
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	listener         net.Listener
	connMu           sync.Mutex
	writeMu          sync.Mutex
	stateMu          sync.RWMutex
	conn             net.Conn
	sequence         uint64
	lastObservedNum  uint64
	lastObservedHash common.Hash
	lastWrittenNum   uint64
	lastWrittenHash  common.Hash
	lastHeadNum      uint64
	lastHeadHash     common.Hash
	hasObserved      bool
	stickyGap        atomic.Bool
	enabled          atomic.Bool
	started          atomic.Bool
	clientReady      atomic.Bool
}

// Stats is a small read-only snapshot useful to health checks and integration
// tests without exposing the publisher's internal queues or socket.
type Stats struct {
	QueueDepth         int
	LastObservedNumber uint64
	LastWrittenNumber  uint64
	GapPending         bool
	Connected          bool
}

func (p *Publisher) Stats() Stats {
	p.stateMu.RLock()
	lastObserved, lastWritten := p.lastObservedNum, p.lastWrittenNum
	p.stateMu.RUnlock()
	p.connMu.Lock()
	connected := p.conn != nil && p.clientReady.Load()
	p.connMu.Unlock()
	return Stats{QueueDepth: len(p.ingress), LastObservedNumber: lastObserved, LastWrittenNumber: lastWritten, GapPending: p.stickyGap.Load(), Connected: connected}
}

func NewPublisher(config Config) *Publisher {
	return &Publisher{config: config, ingress: make(chan blockItem, config.QueueSize)}
}

func (p *Publisher) Start(parent context.Context) error {
	if !p.config.Enable {
		return nil
	}
	if err := p.config.Validate(); err != nil {
		return err
	}
	if p.started.Swap(true) {
		return errors.New("mev feed already started")
	}
	if err := prepareSocket(p.config.SocketPath, p.config.SocketMode); err != nil {
		p.started.Store(false)
		return err
	}
	l, err := net.Listen("unix", p.config.SocketPath)
	if err != nil {
		p.started.Store(false)
		return err
	}
	if err := os.Chmod(p.config.SocketPath, os.FileMode(p.config.SocketMode)); err != nil {
		_ = l.Close()
		_ = os.Remove(p.config.SocketPath)
		p.started.Store(false)
		return err
	}
	p.listener = l
	p.enabled.Store(true)
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel
	p.wg.Add(2)
	go p.acceptLoop(ctx)
	go p.encodeLoop(ctx)
	return nil
}

func (p *Publisher) StopAndWait() {
	if !p.started.Load() {
		return
	}
	p.enabled.Store(false)
	if p.cancel != nil {
		p.cancel()
	}
	if p.listener != nil {
		_ = p.listener.Close()
	}
	p.closeConn()
	p.wg.Wait()
	_ = os.Remove(p.config.SocketPath)
	p.started.Store(false)
}

func (p *Publisher) TryPublish(block *types.Block, receipts types.Receipts) {
	if !p.enabled.Load() || block == nil {
		return
	}
	p.stateMu.Lock()
	p.lastHeadNum, p.lastHeadHash = block.NumberU64(), block.Hash()
	var reorg *reorgItem
	if p.hasObserved && (block.NumberU64() != p.lastObservedNum+1 || block.ParentHash() != p.lastObservedHash) {
		reorg = &reorgItem{oldNum: p.lastObservedNum, oldHash: p.lastObservedHash, newNum: block.NumberU64(), newHash: block.Hash(), parent: block.ParentHash()}
	}
	p.lastObservedNum, p.lastObservedHash = block.NumberU64(), block.Hash()
	p.hasObserved = true
	p.stateMu.Unlock()
	select {
	case p.ingress <- blockItem{block: block, receipts: receipts, reorg: reorg}:
		enqueuedCounter.Inc(1)
		queueDepthGauge.Update(int64(len(p.ingress)))
	default:
		droppedCounter.Inc(1)
		p.stickyGap.Store(true)
		gapsCounter.Inc(1)
	}
}

func (p *Publisher) acceptLoop(ctx context.Context) {
	defer p.wg.Done()
	for {
		c, err := p.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Millisecond)
				continue
			}
		}
		p.connMu.Lock()
		if p.conn != nil {
			p.connMu.Unlock()
			_ = c.Close()
			continue
		}
		p.conn = c
		p.connMu.Unlock()
		clientConnectsCounter.Inc(1)
		p.clientReady.Store(false)
		if err := p.writeHello(); err != nil {
			p.closeConn()
		} else {
			p.clientReady.Store(true)
		}
	}
}

func (p *Publisher) encodeLoop(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-p.ingress:
			queueDepthGauge.Update(int64(len(p.ingress)))
			p.publishItem(item)
		}
	}
}

func (p *Publisher) publishItem(item blockItem) {
	if item.reorg != nil {
		if !p.writeFrame(FrameReorg, reorgPayload(item.reorg.oldNum, item.reorg.newNum, item.reorg.oldHash, item.reorg.newHash, item.reorg.parent)) {
			return
		}
	}
	if len(item.block.Transactions()) != len(item.receipts) {
		encodeErrorsCounter.Inc(1)
		p.stickyGap.Store(true)
		gapsCounter.Inc(1)
		return
	}
	for _, receipt := range item.receipts {
		if receipt == nil {
			encodeErrorsCounter.Inc(1)
			p.stickyGap.Store(true)
			gapsCounter.Inc(1)
			return
		}
	}
	if !p.writeFrame(FrameBlockBegin, blockBeginPayload(item.block, p.config.ChainID)) {
		return
	}
	var crc uint32
	count := uint32(0)
	for i, tx := range item.block.Transactions() {
		payload, err := transactionPayload(tx, item.receipts[i], uint32(i), p.config.ChainID)
		if err != nil {
			encodeErrorsCounter.Inc(1)
			p.stickyGap.Store(true)
			gapsCounter.Inc(1)
			return
		}
		crc = crc32.Update(crc, crcTable, payload)
		if !p.writeFrame(FrameTransaction, payload) {
			return
		}
		count++
	}
	if p.writeFrame(FrameBlockEnd, blockEndPayload(item.block, count, crc)) {
		p.stateMu.Lock()
		p.lastWrittenNum, p.lastWrittenHash = item.block.NumberU64(), item.block.Hash()
		p.stateMu.Unlock()
	}
}

func (p *Publisher) writeHello() error {
	var session [16]byte
	_, _ = rand.Read(session[:])
	p.stateMu.RLock()
	num, hash := p.lastHeadNum, p.lastHeadHash
	p.stateMu.RUnlock()
	if !p.writeFrame(FrameHello, helloPayload(session, p.config.ChainID, num, hash, p.stickyGap.Load())) {
		return errors.New("failed to write MEV feed HELLO")
	}
	return nil
}

func (p *Publisher) writeFrame(kind FrameKind, payload []byte) bool {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	if kind != FrameHello && !p.clientReady.Load() {
		p.stickyGap.Store(true)
		return false
	}
	if p.stickyGap.Load() && kind != FrameGap && kind != FrameHello {
		p.stateMu.RLock()
		lastNum, headNum, lastHash, headHash := p.lastWrittenNum, p.lastHeadNum, p.lastWrittenHash, p.lastHeadHash
		p.stateMu.RUnlock()
		if err := p.writeRaw(FrameGap, gapPayload(lastNum, headNum, lastHash, headHash)); err != nil {
			p.closeConn()
			return false
		}
		p.stickyGap.Store(false)
	}
	p.sequence++
	encoded, err := encodeFrame(frame{kind: kind, sequence: p.sequence, payload: payload}, p.config.MaxFrameBytes)
	if err != nil {
		encodeErrorsCounter.Inc(1)
		p.stickyGap.Store(true)
		return false
	}
	if err := p.writeRawEncoded(encoded); err != nil {
		writeErrorsCounter.Inc(1)
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			writeTimeoutsCounter.Inc(1)
		}
		p.stickyGap.Store(true)
		p.closeConn()
		return false
	}
	encodedFramesCounter.Inc(1)
	encodedBytesCounter.Inc(int64(len(encoded)))
	return true
}

func (p *Publisher) writeRaw(kind FrameKind, payload []byte) error {
	p.sequence++
	encoded, err := encodeFrame(frame{kind: kind, sequence: p.sequence, payload: payload}, p.config.MaxFrameBytes)
	if err != nil {
		return err
	}
	return p.writeRawEncoded(encoded)
}
func (p *Publisher) writeRawEncoded(data []byte) error {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	if p.conn == nil {
		return errors.New("no MEV feed consumer")
	}
	_ = p.conn.SetWriteDeadline(time.Now().Add(p.config.WriteTimeout))
	_, err := io.Copy(p.conn, bytes.NewReader(data))
	return err
}
func (p *Publisher) closeConn() {
	p.clientReady.Store(false)
	p.connMu.Lock()
	defer p.connMu.Unlock()
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
		clientDisconnectsCounter.Inc(1)
	}
}
func prepareSocket(path string, mode uint32) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return errors.New("existing MEV feed path is not a Unix socket")
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}
