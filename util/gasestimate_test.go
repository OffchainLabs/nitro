// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"errors"
	"testing"
)

func TestCheckedGasEstimate(t *testing.T) {
	estimationErr := errors.New("estimation failed")

	for _, tc := range []struct {
		name        string
		gas         uint64
		err         error
		expectedGas uint64
		expectedErr error
	}{
		{name: "non-zero estimate", gas: 21000, expectedGas: 21000},
		{name: "zero estimate", gas: 0, expectedErr: ErrZeroGasEstimate},
		{name: "error is passed through", gas: 21000, err: estimationErr, expectedErr: estimationErr},
		{name: "error wins over zero estimate", gas: 0, err: estimationErr, expectedErr: estimationErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gas, err := CheckedGasEstimate(tc.gas, tc.err)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
			if gas != tc.expectedGas {
				t.Errorf("expected gas %d, got %d", tc.expectedGas, gas)
			}
		})
	}
}
