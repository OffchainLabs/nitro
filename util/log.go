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
func LogLevelEphemeralError(
	err error,
	errorSubstring string,
	ephemeralDuration time.Duration,
	firstOccurrenceTime *time.Time,
	currentLogLevel func(msg string, ctx ...interface{})) func(string, ...interface{}) {
	if strings.Contains(err.Error(), errorSubstring) || errorSubstring == "" {
		logLevel := log.Error
		if *firstOccurrenceTime == (time.Time{}) {
			*firstOccurrenceTime = time.Now()
			logLevel = log.Warn
		} else if time.Since(*firstOccurrenceTime) < ephemeralDuration {
			logLevel = log.Warn
		}
		return logLevel
	} else {
		*firstOccurrenceTime = time.Time{}
		return currentLogLevel
	}
}
