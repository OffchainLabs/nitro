package retry

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

const sleepTime = time.Second * 5

var log = logrus.WithField("prefix", "util")

// UntilSucceeds retries the given function until it succeeds or the context is cancelled.
func UntilSucceeds[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	for {
		if ctx.Err() != nil {
			return zeroVal[T](), ctx.Err()
		}
		got, err := fn()
		if err != nil {
			log.Error(err)
			time.Sleep(sleepTime)
			continue
		}
		return got, nil
	}
}

func zeroVal[T any]() T {
	var result T
	return result
}
