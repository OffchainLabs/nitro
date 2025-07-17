package light_client_test

import (
	"fmt"
	"testing"

	light_client "github.com/offchainlabs/nitro/system_tests/espresso/light_client"
)

// expectPanicWithErrorLightClientUnimplementedMethod is a helper function that
// expects a panic to occur when the provided function is called.
//
// It checks that the panic is of type light_client.ErrorLightClientUnimplementedMethod
// and that the method name matches the expected method name.
func expectPanicWithErrorLightClientUnimplementedMethod(t *testing.T, fn func(), methodName string) {
	defer func() {
		err := recover()
		if have, want := err, (any)(nil); have == want {
			t.Errorf("expected recover to recover from panic:\nhave:\n\t\"%v\"\ndo not want:\n\t\"%v\"", have, want)
			return
		}

		cast, castOk := err.(light_client.ErrorLightClientUnimplementedMethod)

		if have, want := castOk, true; have != want {
			t.Errorf("expected cast to be light_client.ErrorLightClientUnimplementedMethod:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}

		if have, want := cast.Method, methodName; have != want {
			t.Errorf("expected method name to match:\nhave:\n\t\"%v\"\nwant:\n\t\"%v\"", have, want)
			return
		}
	}()

	fn()
}

// TestUnimplementedExecutionClient tests the UnimplementedLightClientReader
// to ensure that it panics when any of its methods are called.
func TestUnimplementedExecutionClient(t *testing.T) {
	var client light_client.UnimplementedLightClientReader

	t.Run("FetchMerkleRoot", func(t *testing.T) {
		expectPanicWithErrorLightClientUnimplementedMethod(t, func() {
			client.FetchMerkleRoot(0, nil)
		}, "FetchMerkleRoot")
	})

	t.Run("IsHotShotLive", func(t *testing.T) {
		expectPanicWithErrorLightClientUnimplementedMethod(t, func() {
			client.IsHotShotLive(0)
		}, "IsHotShotLive")
	})

	t.Run("IsHotShotLiveAtHeight", func(t *testing.T) {
		expectPanicWithErrorLightClientUnimplementedMethod(t, func() {
			client.IsHotShotLiveAtHeight(0, 0)
		}, "IsHotShotLiveAtHeight")
	})

	t.Run("ValidatedHeight", func(t *testing.T) {
		expectPanicWithErrorLightClientUnimplementedMethod(t, func() {
			client.ValidatedHeight()
		}, "ValidatedHeight")
	})
}

// ExampleErrorLightClientUnimplementedMethod demonstrates how to use the
// ErrorLightClientUnimplementedMethod error type.
func ExampleErrorLightClientUnimplementedMethod() {
	err := light_client.ErrorLightClientUnimplementedMethod{
		Method: "Hello",
	}

	fmt.Printf("Error: %s\n", err.Error())
	// Output: Error: unimplemented light-client method: Hello
}
