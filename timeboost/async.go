package timeboost

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

func receiveAsync[T any](ctx context.Context, channel chan T, f func(context.Context, T) error) {
	for {
		select {
		case item := <-channel:
			go func() {
				if err := f(ctx, item); err != nil {
					log.Error("Error processing item", "error", err)
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}
