// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package testhelpers

import (
	"context"
	crypto "crypto/rand"
	"io"
	"math/big"
	"math/rand"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/colors"
	"golang.org/x/exp/slog"
)

// Fail a test should an error occur
func RequireImpl(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatal(colors.Red, printables, err, colors.Clear)
	}
}

func FailImpl(t *testing.T, printables ...interface{}) {
	t.Helper()
	t.Fatal(colors.Red, printables, colors.Clear)
}

func RandomizeSlice(slice []byte) []byte {
	_, err := crypto.Read(slice)
	if err != nil {
		panic(err)
	}
	return slice
}

func RandomSlice(size uint64) []byte {
	return RandomizeSlice(make([]byte, size))
}

func RandomHash() common.Hash {
	var hash common.Hash
	RandomizeSlice(hash[:])
	return hash
}

func RandomAddress() common.Address {
	var address common.Address
	RandomizeSlice(address[:])
	return address
}

func RandomCallValue(limit int64) *big.Int {
	return big.NewInt(rand.Int63n(limit))
}

// Computes a psuedo-random uint64 on the interval [min, max]
func RandomUint32(min, max uint32) uint32 {
	return uint32(RandomUint64(uint64(min), uint64(max)))
}

// Computes a psuedo-random uint64 on the interval [min, max]
func RandomUint64(min, max uint64) uint64 {
	return uint64(rand.Uint64()%(max-min+1) + min)
}

func RandomBool() bool {
	return rand.Int31n(2) == 0
}

type LogHandler struct {
	mutex           sync.Mutex
	t               *testing.T
	records         []slog.Record
	terminalHandler *log.TerminalHandler
}

func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.terminalHandler.Enabled(context.Background(), level)
}
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return h.terminalHandler.WithGroup(name)
}
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.terminalHandler.WithAttrs(attrs)
}

func (h *LogHandler) Handle(_ context.Context, record slog.Record) error {
	if err := h.terminalHandler.Handle(context.Background(), record); err != nil {
		return err
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.records = append(h.records, record)
	return nil
}

func (h *LogHandler) WasLogged(pattern string) bool {
	re, err := regexp.Compile(pattern)
	RequireImpl(h.t, err)
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for _, record := range h.records {
		if re.MatchString(record.Message) {
			return true
		}
	}
	return false
}

func newLogHandler(t *testing.T) *LogHandler {
	return &LogHandler{
		t:               t,
		records:         make([]slog.Record, 0),
		terminalHandler: log.NewTerminalHandler(io.Writer(os.Stderr), false),
	}
}

func InitTestLog(t *testing.T, level slog.Level) *LogHandler {
	handler := newLogHandler(t)
	glogger := log.NewGlogHandler(handler)
	glogger.Verbosity(level)
	log.SetDefault(log.NewLogger(glogger))
	return handler
}
