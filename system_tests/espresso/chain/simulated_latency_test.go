package chain_test

import (
	"context"
	"testing"
	"time"

	espresso_common "github.com/EspressoSystems/espresso-network/sdks/go/types/common"
	"github.com/offchainlabs/nitro/system_tests/espresso/chain"
)

// expectDelay is a helper function that runs the provided function and checks
// if the execution time is at least the expected delay.
func expectDelay(t *testing.T, target time.Duration, fn func()) {
	start := time.Now()
	fn()
	end := time.Now()

	if have, want := end.Sub(start), target; have < want {
		t.Errorf("call didn't take as long as expected:\nhave:\n\t\"%v\"\nwant greater than or equal to:\n\t\"%v\"", have, want)
	}
}

// TestSimulatedLatency tests the EspressoClientSimulatedLatency wrapper to
// ensure that it introduces the expected delays for various methods
// invocations.
func TestSimulatedLatency(t *testing.T) {
	mockChain := chain.NewMockEspressoChain()
	delay := 100 * time.Millisecond

	latencyClient := chain.NewEspressoClientSimulatedLatency(mockChain, chain.WithAllDelaysSetTo(delay))

	t.Run("FetchTransactionsInBlock", func(t *testing.T) {
		expectDelay(t, delay, func() {
			latencyClient.FetchTransactionsInBlock(context.Background(), 1, 1)
		})
	})

	t.Run("FetchTransactionByHash", func(t *testing.T) {
		expectDelay(t, delay, func() {
			latencyClient.FetchTransactionByHash(context.Background(), nil)
		})
	})

	t.Run("SubmitTransaction", func(t *testing.T) {
		expectDelay(t, delay, func() {
			latencyClient.SubmitTransaction(context.Background(), espresso_common.Transaction{})
		})
	})

	t.Run("FetchExplorerTransactionByHash", func(t *testing.T) {
		expectDelay(t, delay, func() {
			latencyClient.FetchExplorerTransactionByHash(context.Background(), nil)
		})
	})
}
