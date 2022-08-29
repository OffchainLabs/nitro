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
	fmt.Printf("Sample usage:                  %s --node.feed.input.url=<L1 RPC> --l2.chain-id=<L2 chain id> \n", progname)
}

func startup() error {
	ctx := context.Background()

	vcsRevision, vcsTime := genericconf.GetVersion()
	relayConfig, err := ParseRelay(ctx, os.Args[1:])
	if err != nil || len(relayConfig.Node.Feed.Input.URLs) == 0 || relayConfig.Node.Feed.Input.URLs[0] == "" || relayConfig.L2.ChainId == 0 {
		fmt.Printf("\nrevision: %v, vcs.time: %v\n", vcsRevision, vcsTime)
		printSampleUsage()
		if err != nil && !strings.Contains(err.Error(), "help requested") {
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
		MaxSendQueue:  relayConfig.Node.Feed.Output.MaxSendQueue,
	}

	defer log.Info("Cleanly shutting down relay")

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	// Start up an arbitrum sequencer relay
	feedErrChan := make(chan error, 10)
	newRelay := relay.NewRelay(serverConf, relayConfig.Node.Feed.Input, relayConfig.L2.ChainId, feedErrChan)
	err = newRelay.Start(ctx)
	if err != nil {
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

type RelayConfig struct {
	Conf     genericconf.ConfConfig `koanf:"conf"`
	L2       L2Config               `koanf:"l2"`
	LogLevel int                    `koanf:"log-level"`
	LogType  string                 `koanf:"log-type"`
	Node     RelayNodeConfig        `koanf:"node"`
}

var RelayConfigDefault = RelayConfig{
	Conf:     genericconf.ConfConfigDefault,
	L2:       L2ConfigDefault,
	LogLevel: int(log.LvlInfo),
	LogType:  "plaintext",
	Node:     RelayNodeConfigDefault,
}

func RelayConfigAddOptions(f *flag.FlagSet) {
	genericconf.ConfConfigAddOptions("conf", f)
	L2ConfigAddOptions("l2", f)
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

type L2Config struct {
	ChainId uint64 `koanf:"chain-id"`
}

var L2ConfigDefault = L2Config{
	ChainId: 0,
}

func L2ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Uint64(prefix+".chain-id", L2ConfigDefault.ChainId, "L2 chain ID")
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
