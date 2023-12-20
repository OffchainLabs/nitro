package util

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// LogLevelEphemeralError is a convenient intermediary level between log levels Warn and Error
//
// For a given error, errorSubstring, duration, firstOccurrenceTime and logLevel
// the function defaults to returning the given logLevel if the error doesnt contain the errorSubstring,
// but if it does, then returns one of the corresponding loglevels as follows
//   - Warn: For firstOccurrenceTime of error being less than the duration amount of time from Now
//   - Error: Otherwise
//
// # Usage Examples
//
//	log.LogLevelEphemeralError(err, "not supported yet", 5*time.Minute, &firstEphemeralError, log.Error)("msg")
//	log.LogLevelEphemeralError(err, "not supported yet", 5*time.Minute, &firstEphemeralError, log.Error)("msg", "key1", val1)
//	log.LogLevelEphemeralError(err, "not supported yet", 5*time.Minute, &firstEphemeralError, log.Error)("msg", "key1", val1, "key2", val2)

type EphemeralError struct {
	Duration        time.Duration
	FirstOccurrence *time.Time
}

func NewEphemeralError(duration time.Duration) *EphemeralError {
	return &EphemeralError{
		Duration:        duration,
		FirstOccurrence: &time.Time{},
	}
}

func (e *EphemeralError) LogLevelEphemeralError(
	err error,
	errorSubstring string,
	currentLogLevel func(msg string, ctx ...interface{})) func(string, ...interface{}) {
	if !strings.Contains(err.Error(), errorSubstring) && errorSubstring != "" {
		e.Reset()
		return currentLogLevel
	}

	logLevel := log.Error
	if *e.FirstOccurrence == (time.Time{}) {
		*e.FirstOccurrence = time.Now()
		logLevel = log.Warn
	} else if time.Since(*e.FirstOccurrence) < e.Duration {
		logLevel = log.Warn
	}
	return logLevel
}

func (e *EphemeralError) Reset() {
	*e.FirstOccurrence = time.Time{}
}
