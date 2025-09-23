package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const Retries = 3

func getLogsAsync(ctx context.Context, client *ethclient.Client, blockNum *big.Int) {
	for i := 0; i < Retries; i++ {
		logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: blockNum,
			ToBlock:   blockNum,
		})
		if err != nil {
			log.Error("failed to get logs", "block", blockNum, "retry", i, "err", err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		log.Info("got logs", "block", blockNum, "retry", i, "lenLogs", len(logs))
		break
	}
}

func subscribeAndGetLogs(ctx context.Context, provider string) error {
	client, err := ethclient.Dial(provider)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	chainId, err := client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain id: %w", err)
	}
	log.Info("got chain id", "chainId", chainId)

	ch := make(chan *types.Header)
	defer close(ch)
	sub, err := client.SubscribeNewHead(ctx, ch)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	log.Info("subscribed to NewHead")
	for {
		select {
		case err := <-sub.Err():
			return fmt.Errorf("subscribe error: %w", err)
		case err := <-ctx.Done():
			return fmt.Errorf("context error: %w", err)
		case block := <-ch:
			go getLogsAsync(ctx, client, block.Number)
		}
	}
}

func main() {
	glog := log.NewGlogHandler(log.NewTerminalHandlerWithLevel(os.Stdout, slog.LevelInfo, true))
	glog.Verbosity(slog.LevelInfo)
	log.SetDefault(log.NewLogger(glog))

	var rpc string
	flag.StringVar(&rpc, "rpc", "", "Ethereum RPC node")
	flag.Parse()

	if rpc == "" {
		fmt.Println("usage: subscribe-getlogs -rpc=<ethereum rpc>")
		flag.PrintDefaults()
		log.Crit("missing rpc")
	}
	rpcUrl, err := url.Parse(rpc)
	if err != nil {
		log.Crit("failed to parse rpc")
	}
	if rpcUrl.Scheme != "wss" {
		log.Crit("rpc scheme must be wss")
	}

	log.Info("connecting to", "rpc", rpc)

	if err := subscribeAndGetLogs(context.Background(), rpc); err != nil {
		fmt.Println("ERROR: ", err)
	}
}
