//go:build falky_eth_test
// +build falky_eth_test

package endtoend

import "testing"

func TestChallengeProtocol_AliceAndBob_AnvilLocal_WithFlakyEthClient(t *testing.T) {
	aliceAndBobInMiddleOfBlock(t, true)
}
