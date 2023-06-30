// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/relay"
)

func init() {
	http.DefaultServeMux = http.NewServeMux()
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running relay", "err", err)
	}
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --node.feed.input.url=<L1 RPC> --chain.id=<L2 chain id> \n", progname)
}

func startup() error {
	ctx := context.Background()

	relayConfig, err := relay.ParseRelay(ctx, os.Args[1:])
	if err != nil || len(relayConfig.Node.Feed.Input.URLs) == 0 || relayConfig.Node.Feed.Input.URLs[0] == "" || relayConfig.L2.ChainId == 0 {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	logFormat, err := genericconf.ParseLogType(relayConfig.LogType)
	if err != nil {
		flag.Usage()
		panic(fmt.Sprintf("Error parsing log type: %v", err))
	}
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	glogger.Verbosity(log.Lvl(relayConfig.LogLevel))
	log.Root().SetHandler(glogger)

	vcsRevision, vcsTime := confighelpers.GetVersion()
	log.Info("Running Arbitrum nitro relay", "revision", vcsRevision, "vcs.time", vcsTime)

	defer log.Info("Cleanly shutting down relay")

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	// Start up an arbitrum sequencer relay
	feedErrChan := make(chan error, 10)
	newRelay, err := relay.NewRelay(relayConfig, feedErrChan)
	if err != nil {
		return err
	}
	err = newRelay.Start(ctx)
	if err != nil {
		return err
	}

	if relayConfig.Metrics && relayConfig.MetricsServer.Addr != "" {
		go metrics.CollectProcessMetrics(relayConfig.MetricsServer.UpdateInterval)

		address := fmt.Sprintf("%v:%v", relayConfig.MetricsServer.Addr, relayConfig.MetricsServer.Port)
		exp.Setup(address)
	}

	select {
	case <-sigint:
		log.Info("shutting down because of sigint")
	case err := <-feedErrChan:
		log.Error("error connecting, exiting", "err", err)
	}

	// cause future ctrl+c's to panic
	close(sigint)

	newRelay.StopAndWait()
	return nil
}
