//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/relay"
	"github.com/offchainlabs/nitro/wsbroadcastserver"
)

func init() {
	http.DefaultServeMux = http.NewServeMux()
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running relay", "err", err)
	}
}

func startup() error {
	ctx, cancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelFunc()

	loglevel := flag.Int("loglevel", int(log.LvlInfo), "log level")

	broadcasterAddr := flag.String("feed.output.addr", "0.0.0.0", "address to bind the relay feed output to")
	broadcasterIOTimeout := flag.Duration("feed.output.io-timeout", 5*time.Second, "duration to wait before timing out HTTP to WS upgrade")
	broadcasterPort := flag.Int("feed.output.port", 9642, "port to bind the relay feed output to")
	broadcasterPing := flag.Duration("feed.output.ping", 5*time.Second, "duration for ping interval")
	broadcasterClientTimeout := flag.Duration("feed.output.client-timeout", 15*time.Second, "duration to wait before timing out connections to client")
	broadcasterWorkers := flag.Int("feed.output.workers", 100, "Number of threads to reserve for HTTP to WS upgrade")

	feedInputUrls := flag.String("feed.input.url", "", "URLs of sequencer feed source, comma separated")
	feedInputTimeout := flag.Duration("feed.input.timeout", 20*time.Second, "duration to wait before timing out conection to server")

	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*loglevel))
	log.Root().SetHandler(glogger)

	serverConf := wsbroadcastserver.BroadcasterConfig{
		Addr:          *broadcasterAddr,
		IOTimeout:     *broadcasterIOTimeout,
		Port:          strconv.Itoa(*broadcasterPort),
		Ping:          *broadcasterPing,
		ClientTimeout: *broadcasterClientTimeout,
		Queue:         100,
		Workers:       *broadcasterWorkers,
	}

	clientConf := broadcastclient.BroadcastClientConfig{
		Timeout: *feedInputTimeout,
		URLs:    strings.Split(*feedInputUrls, ","),
	}

	defer log.Info("Cleanly shutting down relay")

	// Start up an arbitrum sequencer relay
	relay := relay.NewRelay(serverConf, clientConf)
	err := relay.Start(ctx)
	if err != nil {
		return err
	}
	relay.StopAndWait()
	return nil
}
