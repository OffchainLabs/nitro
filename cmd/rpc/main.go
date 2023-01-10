package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

func (t *TempResponder) GetAnswerLaterSubs(ctx context.Context, secs int, number int) (*rpc.Subscription, error) {

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

func (t *TempResponder) ReplyLater(ctx context.Context, secs int, number int) int {

	select {
	case <-time.After(time.Second * time.Duration(secs)):
		return number
	case <-ctx.Done():
		return 0
	}
}

/////////////////////////////////// Client side //////////////////////////////////////

func AskAnswerLaterSubs(ctx context.Context, client *rpc.Client, wg *sync.WaitGroup, secs int, number int) {
	wg.Add(1)
	resChan := make(chan int)
	subs, err := client.Subscribe(ctx, "test", resChan, "getAnswerLaterSubs", secs, number)
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

func AskAnswerLater(ctx context.Context, client *rpc.Client, wg *sync.WaitGroup, secs int, number int) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		var res int
		client.CallContext(ctx, &res, "test_replyLater", secs, number)
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

	_, thisFile, _, _ := runtime.Caller(0)
	rpcTestDir := filepath.Dir(thisFile)
	jwtPath := filepath.Join(rpcTestDir, "jwt.hex")
	var jwtSecret []byte
	data, err := os.ReadFile(jwtPath)
	if err == nil {
		jwtSecret = common.FromHex(strings.TrimSpace(string(data)))
	}
	if err != nil || len(jwtSecret) != 32 {
		log.Crit("failed to read jwt", "err", err, "path", jwtPath, "secret", jwtSecret)
	}

	////////////////////  server

	stackConf := node.DefaultConfig

	stackConf.AuthAddr = "127.0.0.1"
	stackConf.AuthPort = 1505
	stackConf.JWTSecret = jwtPath
	stackConf.WSModules = []string{"test"}
	stackConf.WSPathPrefix = ""
	stackConf.WSOrigins = []string{"*"}
	stackConf.WSExposeAll = false

	node.DefaultAuthModules = []string{"test"}

	stack, err := node.New(&stackConf)
	if err != nil {
		log.Crit("failed to initialize geth stack", "err", err)
	}

	tempAPIs := []rpc.API{{
		Namespace:     "test",
		Version:       "1.0",
		Service:       &TempResponder{},
		Public:        false,
		Authenticated: true,
	}}

	stack.RegisterAPIs(tempAPIs)
	stack.Start()

	////////////////////  client

	ctx := context.Background()

	client, err := rpc.DialWebsocketJWT(ctx, "ws://127.0.0.1:1505", "", jwtSecret)
	if err != nil {
		log.Crit("failed to initialize client", "err", err)
	}

	var result bool
	client.CallContext(ctx, &result, "test_ken")
	log.Info("ken result", "result", result)
	client.CallContext(ctx, &result, "test_lo")
	log.Info("lo result", "result", result)

	var wg sync.WaitGroup
	AskAnswerLater(ctx, client, &wg, 0, 36)
	AskAnswerLater(ctx, client, &wg, 3, 70)
	AskAnswerLater(ctx, client, &wg, 2, 170)
	AskAnswerLater(ctx, client, &wg, 1, 5000)
	wg.Wait()

	client.Close()

	stack.Close()
}
