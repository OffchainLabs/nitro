package blocksreexecutor

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// newTestReExecutor creates a minimal BlocksReExecutor for unit testing
// handleContextOrFatal. Only fatalErrChan is needed; other fields are zero.
func newTestReExecutor(fatalCh chan error) *BlocksReExecutor {
	s := new(BlocksReExecutor)
	s.fatalErrChan = fatalCh
	return s
}

func TestHandleContextOrFatalSuppressesContextErrors(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)

	// context.Canceled should be suppressed
	s.handleContextOrFatal(context.Background(), context.Canceled, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.Canceled should not be fatal, got: %v", err)
	default:
	}

	// Wrapped context.Canceled should also be suppressed
	s.handleContextOrFatal(context.Background(), fmt.Errorf("op failed: %w", context.Canceled), "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("wrapped context.Canceled should not be fatal, got: %v", err)
	default:
	}

	// context.DeadlineExceeded should be suppressed
	s.handleContextOrFatal(context.Background(), context.DeadlineExceeded, "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("context.DeadlineExceeded should not be fatal, got: %v", err)
	default:
	}

	// A real error should be sent to fatalErrChan
	realErr := errors.New("disk corruption")
	s.handleContextOrFatal(context.Background(), realErr, "reexecution failed")
	select {
	case err := <-fatalCh:
		if !errors.Is(err, realErr) {
			t.Fatalf("expected realErr wrapped, got: %v", err)
		}
	default:
		t.Fatal("expected real error to be sent to fatalErrChan")
	}
}

func TestHandleContextOrFatalSuppressesDuringShutdown(t *testing.T) {
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)

	// When context is cancelled (shutdown), even non-context errors should be suppressed
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.handleContextOrFatal(ctx, errors.New("some error during shutdown"), "test")
	select {
	case err := <-fatalCh:
		t.Fatalf("errors during shutdown should be suppressed, got: %v", err)
	default:
	}
}

func TestHandleContextOrFatalDropsWhenChannelFull(t *testing.T) {
	// Use a buffer of 1, fill it, then verify the second error is dropped (not panicking)
	fatalCh := make(chan error, 1)
	s := newTestReExecutor(fatalCh)

	// First error fills the channel
	s.handleContextOrFatal(context.Background(), errors.New("first error"), "test")
	// Second error should be dropped (logged, not panic)
	s.handleContextOrFatal(context.Background(), errors.New("second error"), "test")

	// Only the first error should be in the channel
	err := <-fatalCh
	if err.Error() != "test: first error" {
		t.Fatalf("expected first error, got: %v", err)
	}
	select {
	case err := <-fatalCh:
		t.Fatalf("second error should have been dropped, got: %v", err)
	default:
	}
}
