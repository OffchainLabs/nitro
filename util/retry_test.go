package util

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRetryUntilSucceeds(t *testing.T) {
	hello := func() (string, error) {
		return "hello", nil
	}

	ctx := context.Background()
	got, err := RetryUntilSucceeds(ctx, hello)
	require.NoError(t, err)
	require.Equal(t, "hello", got)

	newCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = RetryUntilSucceeds(newCtx, hello)
	require.ErrorContains(t, err, "context canceled")
}
