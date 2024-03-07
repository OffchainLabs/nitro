package util

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func compareFunctions(f1, f2 func(msg string, ctx ...interface{})) bool {
	return reflect.ValueOf(f1).Pointer() == reflect.ValueOf(f2).Pointer()
}
func TestSimple(t *testing.T) {
	allErrHandler := NewEphemeralErrorHandler(2500*time.Millisecond, "", time.Second)
	err := errors.New("sample error")
	logLevel := allErrHandler.LogLevel(err, log.Error)
	if !compareFunctions(log.Debug, logLevel) {
		t.Fatalf("incorrect loglevel output. Want: Debug")
	}

	time.Sleep(1 * time.Second)
	logLevel = allErrHandler.LogLevel(err, log.Error)
	if !compareFunctions(log.Warn, logLevel) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}

	time.Sleep(2 * time.Second)
	logLevel = allErrHandler.LogLevel(err, log.Error)
	if !compareFunctions(log.Error, logLevel) {
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
	if !compareFunctions(log.Warn, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}
	time.Sleep(2 * time.Second)
	if !compareFunctions(log.Error, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Error")
	}

	errB := errors.New("this is a sample errorB")
	if !compareFunctions(log.Warn, chainingErrHandlers(errB)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}
	if !compareFunctions(log.Warn, chainingErrHandlers(errA)) {
		t.Fatalf("incorrect loglevel output. Want: Warn")
	}

	errC := errors.New("random error")
	if !compareFunctions(log.Error, chainingErrHandlers(errC)) {
		t.Fatalf("incorrect loglevel output. Want: Error")
	}
}
