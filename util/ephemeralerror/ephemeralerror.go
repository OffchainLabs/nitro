package ephemeralerror

import (
	"math"
	"time"

	"go.uber.org/atomic"
)

type LogFn func(msg string, ctx ...interface{})

func NoLog(msg string, ctx ...interface{}) {
}

const notTriggered = math.MinInt64

type Logger interface {
	Error(msg string, ctx ...interface{})
	Reset()
}

type TimeEphemeralErrorLogger struct {
	logFnBeforeTriggered      LogFn
	logFnAfterTriggered       LogFn
	continuousDurationTrigger time.Duration

	firstTriggerTime atomic.Int64
}

func NewTimeEphemeralErrorLogger(
	logFnBeforeTriggered LogFn,
	logFnAfterTriggered LogFn,
	continuousDurationTrigger time.Duration,
) *TimeEphemeralErrorLogger {
	e := TimeEphemeralErrorLogger{
		logFnBeforeTriggered:      logFnBeforeTriggered,
		logFnAfterTriggered:       logFnAfterTriggered,
		continuousDurationTrigger: continuousDurationTrigger,
	}
	e.Reset()
	return &e
}

func (e *TimeEphemeralErrorLogger) Error(msg string, ctx ...interface{}) {
	now := time.Now()
	first := e.firstTriggerTime.CompareAndSwap(notTriggered, now.UnixMicro())
	if !first && e.firstTriggerTime.Load() < now.Add(-e.continuousDurationTrigger).UnixMicro() {
		e.logFnAfterTriggered(msg, ctx)
	} else {
		e.logFnBeforeTriggered(msg, ctx)
	}
}

func (e *TimeEphemeralErrorLogger) Reset() {
	e.firstTriggerTime.Store(notTriggered)
}

type CountEphemeralErrorLogger struct {
	logFnBeforeTriggered LogFn
	logFnAfterTriggered  LogFn
	errorCountTrigger    int64

	errorCount atomic.Int64
}

func NewCountEphemeralErrorLogger(
	logFnBeforeTriggered LogFn,
	logFnAfterTriggered LogFn,
	errorCountTrigger int64,
) *CountEphemeralErrorLogger {
	e := CountEphemeralErrorLogger{
		logFnBeforeTriggered: logFnBeforeTriggered,
		logFnAfterTriggered:  logFnAfterTriggered,
		errorCountTrigger:    errorCountTrigger,
	}
	e.Reset()
	return &e
}

func (e *CountEphemeralErrorLogger) Error(msg string, ctx ...interface{}) {
	if e.errorCount.Add(1) > e.errorCountTrigger {
		e.logFnAfterTriggered(msg, ctx)
	} else {
		e.logFnBeforeTriggered(msg, ctx)
	}

}

func (e *CountEphemeralErrorLogger) Reset() {
	e.errorCount.Store(0)
}
