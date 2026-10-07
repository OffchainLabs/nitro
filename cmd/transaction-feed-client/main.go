// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Command transaction-feed-client is a demo client for the Nitro transaction feed.
//
// It connects to the feed WebSocket endpoint, decodes each JSON message, and
// prints a summary line per message. It accepts feed versions 1 and 2 and
// skips messages of an unknown kind. The file must depend only on the
// standard library and github.com/coder/websocket so that it can be copied
// into any Go module.
//
// Usage:
//
//	transaction-feed-client -addr ws://localhost:9646
//	transaction-feed-client -addr wss://feed.example.com -token <bearer-token>
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coder/websocket"
)

const (
	feedVersionV1 = 1
	feedVersionV2 = 2
	dialTimeout   = 10 * time.Second

	kindTransaction = "Transaction"
	kindRoundEnd    = "RoundEnd"
)

type feedEnvelope struct {
	Version uint32 `json:"version"`
	Kind    string `json:"kind"`
}

func (e *feedEnvelope) kind() string {
	if e.Version == feedVersionV1 {
		return kindTransaction
	}
	return e.Kind
}

type feedMessage struct {
	TimestampMs uint64              `json:"timestamp_ms"`
	PGARound    uint64              `json:"pga_round"`
	Transaction includedTransaction `json:"transaction"`
}

type roundEndMessage struct {
	TimestampMs uint64 `json:"timestamp_ms"`
	PGARound    uint64 `json:"pga_round"`
	BlockNumber uint64 `json:"block_number"`
}

type includedTransaction struct {
	BlockNumber uint64  `json:"block_number"`
	TxIndex     uint32  `json:"tx_index"`
	RawTx       string  `json:"raw_tx"`
	TxHash      string  `json:"tx_hash"`
	Receipt     receipt `json:"receipt"`
}

type receipt struct {
	Status            uint8   `json:"status"`
	GasUsed           uint64  `json:"gas_used"`
	GasUsedForL1      uint64  `json:"gas_used_for_l1"`
	CumulativeGasUsed uint64  `json:"cumulative_gas_used"`
	EffectiveGasPrice string  `json:"effective_gas_price"`
	BaseFee           string  `json:"base_fee"`
	ContractAddress   string  `json:"contract_address"`
	Logs              []txLog `json:"logs"`
}

type txLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

func main() {
	log.SetFlags(0)

	addr := flag.String("addr", "", "feed address starting with ws:// or wss://, e.g. ws://localhost:9646 (required)")
	token := flag.String("token", "", "optional Bearer token sent in the Authorization header to an authenticating proxy in front of the feed")
	reconnectDelay := flag.Duration("reconnect-delay", 2*time.Second, "delay before reconnecting after a disconnect; 0 or less exits on the first disconnect")
	flag.Parse()

	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "error: unexpected arguments: %s\n", strings.Join(flag.Args(), " "))
		flag.Usage()
		os.Exit(2)
	}
	if *addr == "" {
		fmt.Fprintln(os.Stderr, "error: -addr is required")
		flag.Usage()
		os.Exit(2)
	}
	url := *addr
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		fmt.Fprintln(os.Stderr, "error: -addr requires ws:// or wss:// prefix")
		flag.Usage()
		os.Exit(2)
	}

	header := http.Header{}
	if *token != "" {
		header.Set("Authorization", "Bearer "+*token)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		err := run(ctx, url, header)
		if ctx.Err() != nil {
			return
		}
		log.Printf("disconnected: %v", err)
		if *reconnectDelay <= 0 {
			os.Exit(1)
		}
		log.Printf("reconnecting in %s", *reconnectDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(*reconnectDelay):
		}
	}
}

// run connects to the feed and prints messages until the connection drops or
// ctx is canceled. It always returns a non-nil error.
func run(ctx context.Context, url string, header http.Header) error {
	log.Printf("connecting to %s", url)
	conn, err := dial(ctx, url, header)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.CloseNow() }()
	// Feed messages can be larger than the library's default 32 KiB read limit.
	conn.SetReadLimit(-1)
	log.Printf("connected")

	skippedKinds := make(map[string]bool)
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				return fmt.Errorf("server closed connection: status %d %q", closeErr.Code, closeErr.Reason)
			}
			return fmt.Errorf("read: %w", err)
		}

		var envelope feedEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			log.Printf("skipping undecodable message: %v", err)
			continue
		}
		if envelope.Version != feedVersionV1 && envelope.Version != feedVersionV2 {
			log.Printf("skipping message with version %d, expected %d or %d", envelope.Version, feedVersionV1, feedVersionV2)
			continue
		}
		switch kind := envelope.kind(); kind {
		case kindTransaction:
			var msg feedMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("skipping undecodable transaction message: %v", err)
				continue
			}
			printMessage(&msg)
		case kindRoundEnd:
			var msg roundEndMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("skipping undecodable round end message: %v", err)
				continue
			}
			printRoundEnd(&msg)
		default:
			if !skippedKinds[kind] {
				skippedKinds[kind] = true
				log.Printf("skipping messages of unknown kind %q", kind)
			}
		}
	}
}

func dial(ctx context.Context, url string, header http.Header) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("%w (HTTP %s)", err, resp.Status)
		}
		return nil, err
	}
	return conn, nil
}

func formatTimestamp(timestampMs uint64) string {
	// #nosec G115 -- a wrapped value only misprints the timestamp.
	return time.UnixMilli(int64(timestampMs)).UTC().Format(time.RFC3339Nano)
}

func printMessage(msg *feedMessage) {
	tx := &msg.Transaction
	line := fmt.Sprintf("%s block=%d idx=%d hash=%s status=%d gas_used=%d effective_gas_price=%s logs=%d",
		formatTimestamp(msg.TimestampMs), tx.BlockNumber, tx.TxIndex, tx.TxHash, tx.Receipt.Status, tx.Receipt.GasUsed, tx.Receipt.EffectiveGasPrice, len(tx.Receipt.Logs))
	if msg.PGARound != 0 {
		line += fmt.Sprintf(" pga_round=%d", msg.PGARound)
	}
	if tx.Receipt.ContractAddress != "" {
		line += fmt.Sprintf(" contract=%s", tx.Receipt.ContractAddress)
	}
	fmt.Println(line)
}

func printRoundEnd(msg *roundEndMessage) {
	line := fmt.Sprintf("%s round_end block=%d", formatTimestamp(msg.TimestampMs), msg.BlockNumber)
	if msg.PGARound != 0 {
		line += fmt.Sprintf(" pga_round=%d", msg.PGARound)
	}
	fmt.Println(line)
}
