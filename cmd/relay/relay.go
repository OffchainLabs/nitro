//
// Copyright 2021-2022, Offchain Labs, Inc. All rights reserved.
//

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
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

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

func printSampleUsage() {
	progname := os.Args[0]
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func startup() error {
	loglevel := flag.Int("loglevel", int(log.LvlInfo), "log level")

	relayConfig, err := ParseRelay(context.Background(), os.Args[1:])
	if err != nil {
		printSampleUsage()
		if !strings.Contains(err.Error(), "help requested") {
			fmt.Printf("%s\n", err.Error())
		}

		return nil
	}

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*loglevel))
	log.Root().SetHandler(glogger)

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
	err = newRelay.Start(context.Background())
	if err != nil {
		return err
	}
	<-sigint
	newRelay.StopAndWait()
	return nil
}

type RelayConfig struct {
	Conf     conf.ConfConfig `koanf:"conf"`
	LogLevel int             `koanf:"log-level"`
	Node     RelayNodeConfig `koanf:"node"`
}

var RelayConfigDefault = RelayConfig{
	Conf:     conf.ConfConfigDefault,
	LogLevel: int(log.LvlInfo),
	Node:     RelayNodeConfigDefault,
}

func RelayConfigAddOptions(f *flag.FlagSet) {
	conf.ConfConfigAddOptions("conf", f)
	f.Int("log-level", RelayConfigDefault.LogLevel, "log level")
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
		// Print out current configuration

		// Don't keep printing configuration file and don't print wallet passwords
		err := k.Load(confmap.Provider(map[string]interface{}{
			"conf.dump": false,
		}, "."), nil)
		if err != nil {
			return nil, errors.Wrap(err, "error removing extra parameters before dump")
		}

		c, err := k.Marshal(json.Parser())
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal config file to JSON")
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	return &relayConfig, nil
}
