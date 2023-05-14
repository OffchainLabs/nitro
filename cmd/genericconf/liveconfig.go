package genericconf

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ConfigConstrain[T any] interface {
	CanReload(T) error
	GetReloadInterval() time.Duration
}

type OnReloadHook[T ConfigConstrain[T]] func(old T, new T) error

func NoopOnReloadHook[T ConfigConstrain[T]](_ T, _ T) error {
	return nil
}

type ConfigParseFunction[T ConfigConstrain[T]] func(context.Context, []string) (T, error)

type LiveConfig[T ConfigConstrain[T]] struct {
	stopwaiter.StopWaiter

	mutex        sync.RWMutex
	args         []string
	config       T
	onReloadHook OnReloadHook[T]
	parse        ConfigParseFunction[T]
}

func (c *LiveConfig[T]) Get() T {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.config
}

func (c *LiveConfig[T]) Set(config T) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.config.CanReload(config); err != nil {
		return err
	}
	if err := c.onReloadHook(c.config, config); err != nil {
		// TODO(magic) panic? return err? only log the error?
		log.Error("Failed to execute onReloadHook", "err", err)
	}
	c.config = config
	return nil
}

func (c *LiveConfig[T]) Start(ctxIn context.Context) {
	c.StopWaiter.Start(ctxIn, c)

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	c.LaunchThread(func(ctx context.Context) {
		for {
			reloadInterval := c.config.GetReloadInterval()
			if reloadInterval == 0 {
				select {
				case <-ctx.Done():
					return
				case <-sigusr1:
					log.Info("Configuration reload triggered by SIGUSR1.")
				}
			} else {
				timer := time.NewTimer(reloadInterval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-sigusr1:
					timer.Stop()
					log.Info("Configuration reload triggered by SIGUSR1.")
				case <-timer.C:
				}
			}
			nodeConfig, err := c.parse(ctx, c.args)
			if err != nil {
				log.Error("error parsing live config", "error", err.Error())
				continue
			}
			err = c.Set(nodeConfig)
			if err != nil {
				log.Error("error updating live config", "error", err.Error())
				continue
			}
		}
	})
}

// SetOnReloadHook is NOT thread-safe and supports setting only one hook
func (c *LiveConfig[T]) SetOnReloadHook(hook OnReloadHook[T]) {
	c.onReloadHook = hook
}

func NewLiveConfig[T ConfigConstrain[T]](args []string, config T, parse ConfigParseFunction[T]) *LiveConfig[T] {
	return &LiveConfig[T]{
		args:         args,
		config:       config,
		onReloadHook: NoopOnReloadHook[T],
		parse:        parse,
	}
}
