package arbos

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func TestExtraPreTxFilter(t *testing.T) {
	testSender := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// Test case 1: Sender not blacklisted
	err := extraPreTxFilter(nil, nil, nil, nil, nil, nil, testSender, nil)
	if err != nil {
		t.Errorf("extraPreTxFilter() error = %v, want nil for non-blacklisted sender", err)
	}

	// Test case 2: Sender blacklisted
	originalBlacklist := senderBlacklist // Backup original map (if needed for parallel tests)
	senderBlacklist = map[common.Address]struct{}{ // Override for this test
		testSender: {},
	}
	err = extraPreTxFilter(nil, nil, nil, nil, nil, nil, testSender, nil)
	if err == nil {
		t.Errorf("extraPreTxFilter() error = nil, want error for blacklisted sender")
	}
	senderBlacklist = originalBlacklist // Restore original map
}

func TestExtraPostTxFilter(t *testing.T) {
	result := &core.ExecutionResult{}

	// Test case 1: Gas used below limit
	result.UsedGas = maxTxGasLimit - 1
	err := extraPostTxFilter(nil, nil, nil, nil, nil, nil, common.Address{}, nil, result)
	if err != nil {
		t.Errorf("extraPostTxFilter() error = %v, want nil when gas is below limit", err)
	}

	// Test case 2: Gas used exactly at limit
	result.UsedGas = maxTxGasLimit
	err = extraPostTxFilter(nil, nil, nil, nil, nil, nil, common.Address{}, nil, result)
	if err != nil {
		t.Errorf("extraPostTxFilter() error = %v, want nil when gas is at the limit", err)
	}

	// Test case 3: Gas used above limit
	result.UsedGas = maxTxGasLimit + 1
	err = extraPostTxFilter(nil, nil, nil, nil, nil, nil, common.Address{}, nil, result)
	if err == nil {
		t.Errorf("extraPostTxFilter() error = nil, want error when gas is above limit")
	}
} 
