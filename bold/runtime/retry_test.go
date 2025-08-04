// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see:
// https://github.com/offchainlabs/nitro/blob/master/LICENSE.md

package retry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryUntilSucceeds(t *testing.T) {
	hello := func() (string, error) {
		return "hello", nil
	}

	ctx := context.Background()
	got, err := UntilSucceeds(ctx, hello)
	require.NoError(t, err)
	require.Equal(t, "hello", got)

	newCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = UntilSucceeds(newCtx, hello)
	require.ErrorContains(t, err, "context canceled")
}
