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

	IgnoreDuration     time.Duration
	IgnoredErrLogLevel func(string, ...interface{}) // Default IgnoredErrLogLevel is log.Debug
}

func NewEphemeralErrorHandler(duration time.Duration, errorString string, ignoreDuration time.Duration) *EphemeralErrorHandler {
	return &EphemeralErrorHandler{
		Duration:           duration,
		ErrorString:        errorString,
		FirstOccurrence:    &time.Time{},
		IgnoreDuration:     ignoreDuration,
		IgnoredErrLogLevel: log.Debug,
	}
}

// LogLevel method defaults to returning the input currentLogLevel if the given error doesnt contain the errorSubstring,
// but if it does, then returns one of the corresponding loglevels as follows
//   - IgnoredErrLogLevel - if the error has been repeating for less than the IgnoreDuration of time. Defaults to log.Debug
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

	if *h.FirstOccurrence == (time.Time{}) {
		*h.FirstOccurrence = time.Now()
	}

	if h.IgnoreDuration != 0 && time.Since(*h.FirstOccurrence) < h.IgnoreDuration {
		if h.IgnoredErrLogLevel != nil {
			return h.IgnoredErrLogLevel
		}
		return log.Debug
	}

	if time.Since(*h.FirstOccurrence) < h.Duration {
		return log.Warn
	}
	return log.Error
}

func (h *EphemeralErrorHandler) Reset() {
	*h.FirstOccurrence = time.Time{}
}
