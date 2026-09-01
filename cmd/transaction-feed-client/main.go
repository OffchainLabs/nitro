// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// Command transaction-feed-client is a demo client for the Nitro transaction feed.
//
// It connects to the feed WebSocket endpoint, decodes each JSON message, and
// prints a summary line per transaction. The file must depend only on the
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
	feedMessageVersion = 1
	dialTimeout        = 10 * time.Second
)

type feedMessage struct {
	Version     uint32              `json:"version"`
	TimestampMs uint64              `json:"timestamp_ms"`
	PGARound    uint64              `json:"pga_round"`
	Transaction includedTransaction `json:"transaction"`
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

		var msg feedMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("skipping undecodable message: %v", err)
			continue
		}
		if msg.Version != feedMessageVersion {
			log.Printf("skipping message with version %d, expected %d", msg.Version, feedMessageVersion)
			continue
		}
		printMessage(&msg)
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

func printMessage(msg *feedMessage) {
	tx := &msg.Transaction
	// #nosec G115 -- a wrapped value only misprints the timestamp.
	ts := time.UnixMilli(int64(msg.TimestampMs)).UTC().Format(time.RFC3339Nano)
	line := fmt.Sprintf("%s block=%d idx=%d hash=%s status=%d gas_used=%d effective_gas_price=%s logs=%d",
		ts, tx.BlockNumber, tx.TxIndex, tx.TxHash, tx.Receipt.Status, tx.Receipt.GasUsed, tx.Receipt.EffectiveGasPrice, len(tx.Receipt.Logs))
	if msg.PGARound != 0 {
		line += fmt.Sprintf(" pga_round=%d", msg.PGARound)
	}
	if tx.Receipt.ContractAddress != "" {
		line += fmt.Sprintf(" contract=%s", tx.Receipt.ContractAddress)
	}
	fmt.Println(line)
}
