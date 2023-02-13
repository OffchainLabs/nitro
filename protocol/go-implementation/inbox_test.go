package goimpl

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInbox(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inbox := NewInbox(ctx)
	tx := &ActiveTx{TxStatus: ReadWriteTxStatus}
	messages := [][]byte{
		[]byte("a message"),
		[]byte("second message"),
		[]byte("oh no that's a lot of messages"),
	}

	// early-starting subscriber must see correct messages
	c := make(chan error)
	go func() {
		defer close(c)
		msgChan := make(chan []byte)
		inbox.Subscribe(ctx, msgChan)
		for _, msg := range messages {
			seen := <-msgChan
			if !bytes.Equal(msg, seen) {
				c <- ErrWrongState
				return
			}
		}
		select {
		case <-msgChan:
			c <- ErrInvalidOp
		default:
			c <- nil
		}
	}()

	for _, msg := range messages {
		inbox.Append(tx, msg)
	}

	// polling must see correct messages
	require.Equal(t, uint64(len(messages)), inbox.NumMessages(tx))
	for i, msg := range messages {
		seen, err := inbox.GetMessage(tx, uint64(i))
		require.NoError(t, err)
		require.True(t, bytes.Equal(msg, seen))
	}

	// late-starting subscriber must see correct messages
	c2 := make(chan error)
	go func() {
		defer close(c2)
		msgChan := make(chan []byte)
		inbox.Subscribe(ctx, msgChan)
		for _, msg := range messages {
			seen := <-msgChan
			if !bytes.Equal(msg, seen) {
				c2 <- ErrWrongState
				return
			}
		}
		select {
		case <-msgChan:
			c2 <- ErrInvalidOp
		default:
			c2 <- nil
		}
	}()

	require.NoError(t, <-c)
	require.NoError(t, <-c2)
}
