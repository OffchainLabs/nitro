// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package stopwaiter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestStopWaiterStopAndWaitTimeout(t *testing.T) {
	logHandler := initTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	sw.Start(context.Background())
	sw.LaunchThread(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Warn("Going to sleep...")
				time.Sleep(62 * time.Second)
			}
		}
	})
	time.Sleep(100 * time.Millisecond)
	sw.StopAndWait()
	if !logHandler.WasLogged("StopWaiter taking more then 60 seconds to stop") {
		testhelpers.FailImpl(t, "Failed to log about hanging on StopAndWait for more than 60 seconds")
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
