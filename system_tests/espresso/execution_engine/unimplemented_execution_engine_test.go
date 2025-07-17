package execution_engine_test

import (
	"context"
	"fmt"
	"testing"

	execution_engine "github.com/offchainlabs/nitro/system_tests/espresso/execution_engine"
)

// expectPanicWithErrorExecutionClientUnimplementedMethod is a helper function
// that expects a panic to occur when the provided function is called.
//
// It checks that the panic is of type execution_engine.ErrorExecutionClientUnimplementedMethod
// and that the method name matches the expected method name.
func expectPanicWithErrorExecutionClientUnimplementedMethod(t *testing.T, fn func(), methodName string) {
	defer func() {
		err := recover()
		if have, want := err, (any)(nil); have == want {
			t.Errorf("expected recover to recover from panic:\nhave:\n\t\"%v\"\ndo not want:\n\t\"%v\"", have, want)
			return
		}

		cast, castOk := err.(execution_engine.ErrorExecutionClientUnimplementedMethod)

		if have, want := castOk, true; have != want {
			t.Errorf("expected cast to be execution_engine.ErrorExecutionClientUnimplementedMethod:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}

		if have, want := cast.Method, methodName; have != want {
			t.Errorf("expected method name to match:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}
	}()

	fn()
}

// TestUnimplementedExecutionClient tests the UnimplementedExecutionClient
// to ensure that it panics when any of its methods are called.
func TestUnimplementedExecutionClient(t *testing.T) {
	var client execution_engine.UnimplementedExecutionClient

	t.Run("BlockNumberToMessageIndex", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.BlockNumberToMessageIndex(0)
		}, "BlockNumberToMessageIndex")
	})

	t.Run("HeadMessageIndex", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.HeadMessageIndex()
		}, "HeadMessageIndex")
	})

	t.Run("Maintenance", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.Maintenance()
		}, "Maintenance")
	})

	t.Run("MarkFeedStart", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.MarkFeedStart(0)
		}, "MarkFeedStart")
	})

	t.Run("MessageIndexToBlockNumber", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.MessageIndexToBlockNumber(0)
		}, "MessageIndexToBlockNumber")
	})

	t.Run("Reorg", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.Reorg(0, nil, nil)
		}, "Reorg")
	})

	t.Run("ResultAtMessageIndex", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			_ = client.ResultAtMessageIndex(0)
		}, "ResultAtMessageIndex")
	})

	t.Run("SetFinalityData", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			_ = client.SetFinalityData(context.Background(), nil, nil, nil)
		}, "SetFinalityData")
	})

	t.Run("Start", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			_ = client.Start(context.Background())
		}, "Start")
	})

	t.Run("StopAndWait", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			client.StopAndWait()
		}, "StopAndWait")
	})

	t.Run("DigestMessage", func(t *testing.T) {
		expectPanicWithErrorExecutionClientUnimplementedMethod(t, func() {
			_ = client.DigestMessage(0, nil, nil)
		}, "DigestMessage")
	})
}

// ExampleErrorExecutionClientUnimplementedMethod demonstrates how to use the
// ErrorExecutionClientUnimplementedMethod error type.
func ExampleErrorExecutionClientUnimplementedMethod() {
	err := execution_engine.ErrorExecutionClientUnimplementedMethod{
		Method: "Hello",
	}

	fmt.Printf("Error: %s\n", err.Error())
	// Output: Error: unimplemented method for execution client: Hello
}
