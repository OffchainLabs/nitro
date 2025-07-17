package chain_test

import (
	"context"
	"fmt"
	"testing"

	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
)

// expectPanicWithErrorLightClientUnimplementedMethod is a helper function that
// expects a panic to occur when the provided function is called.
//
// It checks that the panic is of type chain.ErrorEspressoClientUnimplementedMethod
// and that the method name matches the expected method name.
func expectPanicWithErrorEspressoClientUnimplementedMethod(t *testing.T, fn func(), methodName string) {
	defer func() {
		err := recover()
		if have, want := err, (any)(nil); have == want {
			t.Errorf("expected recover to recover from panic:\nhave:\n\t\"%v\"\ndo not want:\n\t\"%v\"", have, want)
			return
		}

		cast, castOk := err.(chain.ErrorEspressoClientUnimplementedMethod)

		if have, want := castOk, true; have != want {
			t.Errorf("expected cast to be chain.ErrorEspressoClientUnimplementedMethod:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}

		if have, want := cast.Method, methodName; have != want {
			t.Errorf("expected method name to match:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}
	}()

	fn()
}

// TestUnimplementedExecutionClient tests the UnimplementedEspressoClient
// to ensure that it panics when any of its methods are called.
func TestUnimplementedExecutionClient(t *testing.T) {
	var client chain.UnimplementedEspressoClient

	t.Run("FetchHeaderByHeight", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchHeaderByHeight(context.Background(), 0)
		}, "FetchHeaderByHeight")
	})

	t.Run("FetchHeadersByRange", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchHeadersByRange(context.Background(), 0, 1)
		}, "FetchHeadersByRange")
	})

	t.Run("FetchLatestBlockHeight", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchLatestBlockHeight(context.Background())
		}, "FetchLatestBlockHeight")
	})

	t.Run("FetchRawHeaderByHeight", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchRawHeaderByHeight(context.Background(), 0)
		}, "FetchRawHeaderByHeight")
	})

	t.Run("FetchTransactionByHash", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchTransactionByHash(context.Background(), nil)
		}, "FetchTransactionByHash")
	})

	t.Run("FetchTransactionsInBlock", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchTransactionsInBlock(context.Background(), 0, 0)
		}, "FetchTransactionsInBlock")
	})

	t.Run("FetchVidCommonByHeight", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchVidCommonByHeight(context.Background(), 0)
		}, "FetchVidCommonByHeight")
	})

	t.Run("SubmitTransaction", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.SubmitTransaction(context.Background(), espresso_common.Transaction{})
		}, "SubmitTransaction")
	})

	t.Run("FetchExplorerTransactionByHash", func(t *testing.T) {
		expectPanicWithErrorEspressoClientUnimplementedMethod(t, func() {
			client.FetchExplorerTransactionByHash(context.Background(), nil)
		}, "FetchExplorerTransactionByHash")
	})
}

// ExampleErrorEspressoClientUnimplementedMethod demonstrates how to use the
// ErrorEspressoClientUnimplementedMethod error type.
func ExampleErrorEspressoClientUnimplementedMethod() {
	err := chain.ErrorEspressoClientUnimplementedMethod{
		Method: "Hello",
	}

	fmt.Printf("Error: %s\n", err.Error())
	// Output: Error: unimplemented espresso client method: Hello
}
