package util

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func TestSimple(t *testing.T) {
	allErrHandler := NewEphemeralErrorHandler(2500*time.Millisecond, "", time.Second)
	err := errors.New("sample error")
	logLevel := allErrHandler.LogLevel(err, log.Error)
	if !CompareLogLevels(log.Debug, logLevel) {
		t.Fatalf("incorrect loglevel output. Want: Debug")
	}

	time.Sleep(1 * time.Second)
	logLevel = allErrHandler.LogLevel(err, log.Error)
	if !CompareLogLevels(log.Warn, logLevel) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}

	time.Sleep(2 * time.Second)
	logLevel = allErrHandler.LogLevel(err, log.Error)
	if !CompareLogLevels(log.Error, logLevel) {
		t.Fatalf("incorrect loglevel output. Want: Error")
	}
}

func TestComplex(t *testing.T) {
	// Simulation: errorA happens continuously for 2 seconds and then errorB happens
	errorAHandler := NewEphemeralErrorHandler(time.Second, "errorA", 0)
	errorBHandler := NewEphemeralErrorHandler(1500*time.Millisecond, "errorB", 0)

	// Computes result of chaining two ephemeral error handlers for a given recurring error
	chainingErrHandlers := func(err error) func(string, ...interface{}) {
		logLevel := log.Error
		logLevel = errorAHandler.LogLevel(err, logLevel)
		logLevel = errorBHandler.LogLevel(err, logLevel)
		return logLevel
	}

	errA := errors.New("this is a sample errorA")
	if !CompareLogLevels(log.Warn, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}
	time.Sleep(2 * time.Second)
	if !CompareLogLevels(log.Error, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Error")
	}

	errB := errors.New("this is a sample errorB")
	if !CompareLogLevels(log.Warn, chainingErrHandlers(errB)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}
	if !CompareLogLevels(log.Warn, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}

	errC := errors.New("random error")
	if !CompareLogLevels(log.Error, chainingErrHandlers(errC)) {
		t.Fatalf("incorrect loglevel output. Want: Error")
	}
}
