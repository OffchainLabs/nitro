// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package stopwaiter

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

const testStopDelayWarningTimeout = 350 * time.Millisecond

type TestStruct struct{}

func TestStopWaiterStopAndWaitTimeoutShouldWarn(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sw.Start(context.Background(), &TestStruct{})
	sw.LaunchThread(func(ctx context.Context) {
		<-testCtx.Done()
	})
	go func() {
		err := sw.stopAndWaitImpl(testStopDelayWarningTimeout)
		testhelpers.RequireImpl(t, err)
	}()
	time.Sleep(testStopDelayWarningTimeout + 100*time.Millisecond)
	if !logHandler.WasLogged("taking too long to stop") {
		testhelpers.FailImpl(t, "Failed to log about waiting long on StopAndWait")
	}
}

func TestStopWaiterStopAndWaitTimeoutShouldNotWarn(t *testing.T) {
	logHandler := testhelpers.InitTestLog(t, log.LvlTrace)
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})
	sw.LaunchThread(func(ctx context.Context) {
		<-ctx.Done()
	})
	sw.StopAndWait()
	if logHandler.WasLogged("taking too long to stop") {
		testhelpers.FailImpl(t, "Incorrectly logged about waiting long on StopAndWait")
	}
}

func TestStopWaiterStopAndWaitBeforeStart(t *testing.T) {
	sw := StopWaiter{}
	sw.StopAndWait()
}

func TestStopWaiterStopAndWaitAfterStop(t *testing.T) {
	sw := StopWaiter{}
	sw.Start(context.Background(), &TestStruct{})
	ctx := sw.GetContext()
	sw.StopOnly()
	<-ctx.Done()
	sw.StopAndWait()
}

func TestStopWaiterStopAndWaitMultipleTimes(t *testing.T) {
	sw := StopWaiter{}
	sw.StopAndWait()
	sw.StopAndWait()
	sw.StopAndWait()
	sw.Start(context.Background(), &TestStruct{})
	sw.StopAndWait()
	sw.StopAndWait()
	sw.StopAndWait()
}
