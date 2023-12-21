package util

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// EphemeralErrorHandler handles errors that are ephemeral in nature i.h these are errors
// that we would like to log as a warning unless they repeat for more than a certain duration of time.
type EphemeralErrorHandler struct {
	Duration        time.Duration
	ErrorString     string
	FirstOccurrence *time.Time
}

func NewEphemeralErrorHandler(duration time.Duration, errorString string) *EphemeralErrorHandler {
	return &EphemeralErrorHandler{
		Duration:        duration,
		ErrorString:     errorString,
		FirstOccurrence: &time.Time{},
	}
}

// LogLevel method defaults to returning the input currentLogLevel if the givenerror doesnt contain the errorSubstring,
// but if it does, then returns one of the corresponding loglevels as follows
//   - log.Warn - if the error has been repeating for less than the given duration of time
//   - log.Error - Otherwise
//
// # Usage Examples
//
//	ephemeralErrorHandler.Loglevel(err, log.Error)("msg")
//	ephemeralErrorHandler.Loglevel(err, log.Error)("msg", "key1", val1, "key2", val2)
//	ephemeralErrorHandler.Loglevel(err, log.Error)("msg", "key1", val1)
func (h *EphemeralErrorHandler) LogLevel(err error, currentLogLevel func(msg string, ctx ...interface{})) func(string, ...interface{}) {
	if h.ErrorString != "" && !strings.Contains(err.Error(), h.ErrorString) {
		h.Reset()
		return currentLogLevel
	}

	logLevel := log.Error
	if *h.FirstOccurrence == (time.Time{}) {
		*h.FirstOccurrence = time.Now()
		logLevel = log.Warn
	} else if time.Since(*h.FirstOccurrence) < h.Duration {
		logLevel = log.Warn
	}
	return logLevel
}

func (h *EphemeralErrorHandler) Reset() {
	*h.FirstOccurrence = time.Time{}
}
