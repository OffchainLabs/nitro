// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/util"
	flag "github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/broadcastclient"
	"github.com/offchainlabs/nitro/cmd/genericconf"
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

func printSampleUsage() {
	progname := os.Args[0]
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func startup() error {
	ctx := context.Background()

	vcsRevision, vcsTime := genericconf.GetVersion()
	relayConfig, err := ParseRelay(ctx, os.Args[1:])
	if err != nil {
		fmt.Printf("\nrevision: %v, vcs.time: %v\n", vcsRevision, vcsTime)
		printSampleUsage()
		if !strings.Contains(err.Error(), "help requested") {
			fmt.Printf("%s\n", err.Error())
		}

		return nil
	}

	logFormat, err := genericconf.ParseLogType(relayConfig.LogType)
	if err != nil {
		flag.Usage()
		panic(fmt.Sprintf("Error parsing log type: %v", err))
	}
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, logFormat))
	glogger.Verbosity(log.Lvl(relayConfig.LogLevel))
	log.Root().SetHandler(glogger)

	log.Info("Running Arbitrum nitro relay", "revision", vcsRevision, "vcs.time", vcsTime)

	serverConf := wsbroadcastserver.BroadcasterConfig{
		Addr:          relayConfig.Node.Feed.Output.Addr,
		IOTimeout:     relayConfig.Node.Feed.Output.IOTimeout,
		Port:          relayConfig.Node.Feed.Output.Port,
		Ping:          relayConfig.Node.Feed.Output.Ping,
		ClientTimeout: relayConfig.Node.Feed.Output.ClientTimeout,
		Queue:         relayConfig.Node.Feed.Output.Queue,
		Workers:       relayConfig.Node.Feed.Output.Workers,
	}

	clientConf := broadcastclient.BroadcastClientConfig{
		Timeout: relayConfig.Node.Feed.Input.Timeout,
		URLs:    relayConfig.Node.Feed.Input.URLs,
	}

	defer log.Info("Cleanly shutting down relay")

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	// Start up an arbitrum sequencer relay
	newRelay := relay.NewRelay(serverConf, clientConf)
	err = newRelay.Start(ctx)
	if err != nil {
		return err
	}
	<-sigint
	newRelay.StopAndWait()
	return nil
}

type RelayConfig struct {
	Conf     genericconf.ConfConfig `koanf:"conf"`
	LogLevel int                    `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`
	Node     RelayNodeConfig        `koanf:"node"`
}

var RelayConfigDefault = RelayConfig{
	Conf:     genericconf.ConfConfigDefault,
	LogLevel: int(log.LvlInfo),
	LogType:  "plaintext",
	Node:     RelayNodeConfigDefault,
}

func RelayConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	f.Int("log-level", RelayConfigDefault.LogLevel, "log level")
	f.String("log-type", RelayConfigDefault.LogType, "log type")
	RelayNodeConfigAddOptions("node", f)
}

type RelayNodeConfig struct {
	Feed broadcastclient.FeedConfig `koanf:"feed"`
}

var RelayNodeConfigDefault = RelayNodeConfig{
	Feed: broadcastclient.FeedConfigDefault,
}

func RelayNodeConfigAddOptions(prefix string, f *flag.FlagSet) {
	broadcastclient.FeedConfigAddOptions(prefix+".feed", f, true, true)
}

func ParseRelay(_ context.Context, args []string) (*RelayConfig, error) {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	RelayConfigAddOptions(f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var relayConfig RelayConfig
	if err := util.EndCommonParse(k, &relayConfig); err != nil {
		return nil, err
	}

	if relayConfig.Conf.Dump {
		err = util.DumpConfig(k, map[string]interface{}{})
		if err != nil {
			return nil, err
		}
	}

	return &relayConfig, nil
}
