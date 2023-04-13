package stopwaiter

import (
	"context"
	"testing"
	"time"

	"github.com/offchainlabs/nitro/util/containers"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

type ClassA struct {
	StopWaiter
}

func (c *ClassA) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
}

func (c *ClassA) shortFunc() (uint64, error) {
	return 42, nil
}

func (c *ClassA) longFunc(ctx context.Context, delay time.Duration) (uint64, error) {
	select {
	case <-time.After(delay):
		return 42, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (c *ClassA) ShortFunc() containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](c.shortFunc())
}

func (c *ClassA) LongFunc(delay time.Duration) containers.PromiseInterface[uint64] {
	return LaunchPromiseThread[uint64](c, func(ctx context.Context) (uint64, error) {
		return c.longFunc(ctx, delay)
	})
}

type Caller struct {
	StopWaiter
	calee *ClassA
}

func (c *Caller) Start(ctx context.Context) {
	c.StopWaiter.Start(ctx, c)
}

func (c *Caller) ShortCaller() error {
	_, err := c.calee.ShortFunc().Await(c.GetContext())
	return err
}

func (c *Caller) LongCaller(delay time.Duration) error {
	_, err := c.calee.LongFunc(delay).Await(c.GetContext())
	return err
}

func TestStopWaiterPromise(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	classA := &ClassA{}
	caller := &Caller{
		calee: classA,
	}
	classA.Start(ctx)
	caller.Start(ctx)

	Require(t, caller.ShortCaller())
	Require(t, caller.LongCaller(time.Millisecond*200))

	go func() {
		<-time.After(time.Millisecond * 10)
		caller.StopAndWait()
	}()
	err := caller.LongCaller(time.Minute)
	if err == nil {
		t.Fatal("longcaller succeeded after caller stop")
	}

	callerB := &Caller{
		calee: classA,
	}
	callerB.Start(ctx)
	Require(t, callerB.LongCaller(time.Millisecond*200))

	go func() {
		<-time.After(time.Millisecond * 10)
		classA.StopAndWait()
	}()
	err = callerB.LongCaller(time.Minute)
	if err == nil {
		t.Fatal("longcaller succeeded after caller stop")
	}
}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}
