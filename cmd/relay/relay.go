// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"io"
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

func main() {
	if err := startup(); err != nil {
		log.Error("Error running relay", "err", err)
	}
}

func printSampleUsage(progname string) {
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --node.feed.input.url=<L1 RPC> --chain.id=<L2 chain id> \n", progname)
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics(cfg *relay.Config) error {
	mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	if cfg.Metrics && !metrics.Enabled {
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if cfg.Metrics && cfg.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if cfg.Metrics {
		go metrics.CollectProcessMetrics(cfg.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	}
	if cfg.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}

func startup() error {
	ctx := context.Background()

	relayConfig, err := relay.ParseRelay(ctx, os.Args[1:])
	if err != nil || len(relayConfig.Node.Feed.Input.URL) == 0 || relayConfig.Node.Feed.Input.URL[0] == "" || relayConfig.Chain.ID == 0 {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	handler, err := genericconf.HandlerFromLogType(relayConfig.LogType, io.Writer(os.Stderr))
	if err != nil {
		flag.Usage()
		return fmt.Errorf("error parsing log type when creating handler: %w", err)
	}
	logLevel, err := genericconf.ToSlogLevel(relayConfig.LogLevel)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}
	glogger := log.NewGlogHandler(handler)
	glogger.Verbosity(logLevel)
	log.SetDefault(log.NewLogger(glogger))

	vcsRevision, _, vcsTime := confighelpers.GetVersion()
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

	if err := startMetrics(relayConfig); err != nil {
		return err
	}

	if err := newRelay.Start(ctx); err != nil {
		return err
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
