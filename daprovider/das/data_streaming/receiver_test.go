package data_streaming

import (
    "context"
    "testing"
    "time"
)

// Ensure that attempting to register a message with totalSize == 0 fails
func TestRegisterNewMessage_ZeroTotalSizeRejected(t *testing.T) {
    // Stub verifier that always accepts the payload
    verifier := CustomPayloadVerifier(func(ctx context.Context, _ []byte, _ []byte, _ ...uint64) error { return nil })

    dsr := NewDataStreamReceiver(verifier, 10, time.Second, nil)

    // Call StartReceiving through the public API to exercise validation path as in production
    _, err := dsr.StartReceiving(context.Background(), uint64(time.Now().Unix()), 1, 1024, 0, 10, nil)
    if err == nil {
        t.Fatalf("expected error when totalSize == 0, got nil")
    }
}
