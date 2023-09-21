package ephemeralerror

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCountEphemeralError(t *testing.T) {
	var warnCount, errorCount atomic.Int64
	e := NewCountEphemeralErrorLogger(
		func(msg string, ctx ...interface{}) { warnCount.Add(1) },
		func(msg string, ctx ...interface{}) { errorCount.Add(1) },
		10,
	)

	for run := 0; run < 1000; run++ {
		var errorCountTrigger, nEvents int64 = rand.Int63n(100), rand.Int63n(100)
		e.errorCountTrigger = errorCountTrigger
		expectedWarns := errorCountTrigger
		expectedErrors := nEvents - expectedWarns

		if expectedErrors < 0 {
			expectedWarns = nEvents
			expectedErrors = 0
		}

		wg := sync.WaitGroup{}
		for i := int64(0); i < nEvents; i++ {
			wg.Add(1)
			go func() {
				e.Error("bbq!")
				wg.Done()
			}()
		}

		wg.Wait()

		if warnCount.Load() != expectedWarns || errorCount.Load() != expectedErrors {
			t.Fatalf("unexpected warnCount, errorCount (%d, %d), expected (%d, %d), %d", warnCount.Load(), errorCount.Load(), expectedWarns, expectedErrors, nEvents)
		}

		e.Reset()
		warnCount.Store(0)
		errorCount.Store(0)
	}
}

func TestTimeEphemeralError(t *testing.T) {
	var warnCount, errorCount atomic.Int64
	e := NewTimeEphemeralErrorLogger(
		func(msg string, ctx ...interface{}) { warnCount.Add(1) },
		func(msg string, ctx ...interface{}) { errorCount.Add(1) },
		time.Second,
	)

	for run := 0; run < 10; run++ {
		totalDuration := (time.Duration(rand.Int63n(20)) + 5) * time.Millisecond * 50
		e.continuousDurationTrigger = totalDuration

		var expectedWarns, expectedErrors int64 = rand.Int63n(9) + 1, rand.Int63n(9) + 1
		totalEvents := expectedWarns + expectedErrors
		period := totalDuration / time.Duration(expectedWarns)
		for i := int64(0); i < totalEvents; i++ {
			e.Error("bbq!")
			time.Sleep(period)
		}

		if warnCount.Load() != expectedWarns || errorCount.Load() != expectedErrors {
			t.Fatalf("unexpected warnCount, errorCount (%d, %d), expected (%d, %d), %v, %v", warnCount.Load(), errorCount.Load(), expectedWarns, expectedErrors, totalDuration, period)
		}

		e.Reset()
		warnCount.Store(0)
		errorCount.Store(0)
	}

}
