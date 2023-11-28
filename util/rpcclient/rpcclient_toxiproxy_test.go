//go:build toxiproxy
// +build toxiproxy

package rpcclient

import (
	"context"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/client"
)

func TestToxiRpcClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	server1 := createTestNode(t, ctx, 0)

	toxiprox := toxiproxy.NewClient("localhost:8474")
	proxy, err := toxiprox.CreateProxy("testRpc", "", server1.WSEndpoint()[5:])
	Require(t, err)
	defer proxy.Delete()

	config := &ClientConfig{
		URL:         "ws://" + proxy.Listen,
		Timeout:     time.Second * 5,
		Retries:     3,
		RetryErrors: "websocket: close.*|.* i/o timeout|.*connection reset by peer|dial tcp .*",
		RetryDelay:  time.Millisecond * 500,
	}
	Require(t, config.Validate())
	configFetcher := func() *ClientConfig { return config }

	client := NewRpcClient(configFetcher, server1)

	err = client.Start(ctx)
	Require(t, err)

	err = client.CallContext(ctx, nil, "test_delay", 500)
	Require(t, err)

	errChan := make(chan error)
	proxyErrChan := make(chan error)
	callDealy := func() {
		callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		errChan <- client.CallContext(callCtx, nil, "test_delay", 3000)
		cancel()
	}
	proxyDisable := func() {
		<-time.After(time.Millisecond * 30)
		err = proxy.Disable()
		if err != nil {
			proxyErrChan <- err
			return
		}
		<-time.After(time.Millisecond * 500)
		err = proxy.Enable()
		proxyErrChan <- err
	}
	proxyReset := func() {
		<-time.After(time.Millisecond * 20)
		_, err = proxy.AddToxic("reset_all", "reset_peer", "downstream", 1.0, toxiproxy.Attributes{"timeout": 5})
		if err != nil {
			proxyErrChan <- err
			return
		}
		<-time.After(time.Millisecond * 3000)
		err = proxy.RemoveToxic("reset_all")
		proxyErrChan <- err
	}

	config.Retries = 0
	go callDealy()
	go proxyDisable()
	err = <-proxyErrChan
	Require(t, err)
	err = <-errChan
	if err == nil {
		Fail(t, "call during proxyDisable succeeded without retries")
	}

	config.Retries = 3
	go callDealy()
	go proxyDisable()
	err = <-proxyErrChan
	Require(t, err)
	err = <-errChan
	if err != nil {
		Fail(t, "call during proxyDisable failed with retries:", err)
	}

	config.Retries = 0
	go callDealy()
	go proxyReset()
	err = <-proxyErrChan
	Require(t, err)
	err = <-errChan
	if err == nil {
		Fail(t, "call during proxyReset succeeded without retries")
	}

	config.Retries = 3
	go callDealy()
	go proxyReset()
	err = <-proxyErrChan
	Require(t, err)
	err = <-errChan
	if err != nil {
		Fail(t, "call during proxyReset failed with retries:", err)
	}
}
