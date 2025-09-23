// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
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

func startup() error {
	ctx := context.Background()

	relayConfig, err := relay.ParseRelay(ctx, os.Args[1:])
	if err != nil || len(relayConfig.Node.Feed.Input.URL) == 0 || relayConfig.Node.Feed.Input.URL[0] == "" || relayConfig.Chain.ID == 0 {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	handler, err := genericconf.HandlerFromLogType(relayConfig.LogType, io.Writer(os.Stderr))
	if err != nil {
		pflag.Usage()
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

	err = util.StartMetricsAndPProf(&util.MetricsPProfOpts{
		Metrics:       relayConfig.Metrics,
		MetricsServer: relayConfig.MetricsServer,
		PProf:         relayConfig.PProf,
		PprofCfg:      relayConfig.PprofCfg,
	})
	if err != nil {
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
