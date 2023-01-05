package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

/////////////////////////////////// Server side //////////////////////////////////////

type TempResponder struct{}

func (t *TempResponder) Ken() bool {
	return true
}

func (t *TempResponder) Lo() bool {
	return false
}

func (t *TempResponder) GetAnswerLater(ctx context.Context, secs int, number int) (*rpc.Subscription, error) {

	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		select {
		case <-time.After(time.Second * time.Duration(secs)):
			notifier.Notify(rpcSub.ID, number)
		case <-rpcSub.Err():
		case <-notifier.Closed():
		}
	}()

	return rpcSub, nil
}

/////////////////////////////////// Client side //////////////////////////////////////

func AskAnswerLater(ctx context.Context, client *rpc.Client, wg *sync.WaitGroup, secs int, number int) {
	wg.Add(1)
	resChan := make(chan int)
	subs, err := client.Subscribe(ctx, "test", resChan, "getAnswerLater", secs, number)
	if err != nil {
		log.Crit("failed to subscribe to result", "err", err)
	}

	go func() {
		defer subs.Unsubscribe()
		defer wg.Done()

		var res int
		select {
		case <-ctx.Done():
			return
		case res = <-resChan:
		}
		if res != number {
			log.Crit("got bad result", "expected", number, "got", res)
		} else {
			log.Info("got expected result", "res", res)
		}
	}()
}

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stdout, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	////////////////////  server

	stackConf := node.DefaultConfig

	stackConf.WSHost = "127.0.0.1"
	stackConf.WSPort = 1505
	stackConf.WSModules = []string{"test"}
	stackConf.WSPathPrefix = ""
	stackConf.WSOrigins = []string{"*"}
	stackConf.WSExposeAll = false

	stack, err := node.New(&stackConf)
	if err != nil {
		log.Crit("failed to initialize geth stack", "err", err)
	}

	tempAPIs := []rpc.API{{
		Namespace: "test",
		Version:   "1.0",
		Service:   &TempResponder{},
		Public:    true,
	}}

	stack.RegisterAPIs(tempAPIs)
	stack.Start()

	////////////////////  client

	ctx := context.Background()

	client, err := rpc.DialContext(ctx, "ws://127.0.0.1:1505")
	if err != nil {
		log.Crit("failed to initialize client", "err", err)
	}

	var result bool
	client.CallContext(ctx, &result, "test_ken")
	log.Info("ken result", "result", result)
	client.CallContext(ctx, &result, "test_lo")
	log.Info("lo result", "result", result)

	var wg sync.WaitGroup
	AskAnswerLater(ctx, client, &wg, 1, 70)
	AskAnswerLater(ctx, client, &wg, 2, 170)
	AskAnswerLater(ctx, client, &wg, 3, 5000)
	wg.Wait()

	stack.Close()
}
