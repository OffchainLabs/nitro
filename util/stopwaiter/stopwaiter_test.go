// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package stopwaiter

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const testStopDelayWarningTimeout = 350 * time.Millisecond

type TestStruct struct{}

func TestStopWaiterStopAndWaitTimeout(t *testing.T) {
	logHandler := initTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	sw.Start(context.Background(), TestStruct{})
	sw.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(testStopDelayWarningTimeout + 150*time.Millisecond)
			}
		}
	})
	time.Sleep(50 * time.Millisecond)
	sw.stopAndWaitImpl(testStopDelayWarningTimeout)
	if !logHandler.WasLogged(fmt.Sprintf("stopwaiter.TestStruct taking more than %s to stop", testStopDelayWarningTimeout.String())) {
		testhelpers.FailImpl(t, "Failed to log about hanging on StopAndWait")
	}
}

type LogHandler struct {
	t       *testing.T
	records []log.Record
}

func (h *LogHandler) Log(record *log.Record) error {
	h.t.Log(record.Msg)
	h.records = append(h.records, *record)
	return nil
}

func (h *LogHandler) WasLogged(pattern string) bool {
	for _, r := range h.records {
		if strings.Contains(r.Msg, pattern) {
			return true
		}
	}
	return false
}

func initTestLog(t *testing.T, level log.Lvl) *LogHandler {
	handler := LogHandler{t, make([]log.Record, 0)}
	glogger := log.NewGlogHandler(&handler)
	glogger.Verbosity(level)
	log.Root().SetHandler(glogger)
	return &handler
}
