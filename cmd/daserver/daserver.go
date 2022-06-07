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
	"time"

	koanfjson "github.com/knadh/koanf/parsers/json"
	flag "github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/das/dasrpc"
)

type DAServerConfig struct {
	EnableRPC bool   `koanf:"enable-rpc"`
	RPCAddr   string `koanf:"rpc-addr"`
	RPCPort   uint64 `koanf:"rpc-port"`

	EnableREST bool   `koanf:"enable-rest"`
	RESTAddr   string `koanf:"rest-addr"`
	RESTPort   uint64 `koanf:"rest-port"`

	DAConf das.DataAvailabilityConfig `koanf:"data-availability"`

	ConfConfig genericconf.ConfConfig `koanf:"conf"`
	LogLevel   int                    `koanf:"log-level"`
}

var DefaultDAServerConfig = DAServerConfig{
	EnableRPC:  false,
	RPCAddr:    "localhost",
	RPCPort:    9876,
	EnableREST: false,
	RESTAddr:   "localhost",
	RESTPort:   9877,
	DAConf:     das.DefaultDataAvailabilityConfig,
	ConfConfig: genericconf.ConfConfigDefault,
	LogLevel:   3,
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running DAServer", "err", err)
	}
}

func printSampleUsage() {
	progname := os.Args[0]
	fmt.Printf("\n")
	fmt.Printf("Sample usage:                  %s --help \n", progname)
}

func parseDAServer(args []string) (*DAServerConfig, error) {
	f := flag.NewFlagSet("daserver", flag.ContinueOnError)
	f.Bool("enable-rpc", DefaultDAServerConfig.EnableRPC, "enable the HTTP-RPC server listening on rpc-addr and rpc-port")
	f.String("rpc-addr", DefaultDAServerConfig.RPCAddr, "HTTP-RPC server listening interface")
	f.Uint64("rpc-port", DefaultDAServerConfig.RPCPort, "HTTP-RPC server listening port")

	f.Bool("enable-rest", DefaultDAServerConfig.EnableREST, "enable the REST server listening on rest-addr and rest-port")
	f.String("rest-addr", DefaultDAServerConfig.RESTAddr, "REST server listening interface")
	f.Uint64("rest-port", DefaultDAServerConfig.RESTPort, "REST server listening port")

	f.Int("log-level", int(log.LvlInfo), "log level; 1: ERROR, 2: WARN, 3: INFO, 4: DEBUG, 5: TRACE")
	das.DataAvailabilityConfigAddOptions("data-availability", f)
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var serverConfig DAServerConfig
	if err := util.EndCommonParse(k, &serverConfig); err != nil {
		return nil, err
	}
	if serverConfig.ConfConfig.Dump {
		err = util.DumpConfig(k, map[string]interface{}{
			"data-availability.key.priv-key": "",
		})
		if err != nil {
			return nil, fmt.Errorf("error removing extra parameters before dump: %w", err)
		}

		c, err := k.Marshal(koanfjson.Parser())
		if err != nil {
			return nil, fmt.Errorf("unable to marshal config file to JSON: %w", err)
		}

		fmt.Println(string(c))
		os.Exit(0)
	}

	return &serverConfig, nil
}

func startup() error {
	// Some different defaults to DAS config in a node.
	das.DefaultDataAvailabilityConfig.Enable = true

	vcsRevision, vcsTime := genericconf.GetVersion()
	serverConfig, err := parseDAServer(os.Args[1:])
	if err != nil {
		fmt.Printf("\nrevision: %v, vcs.time: %v\n", vcsRevision, vcsTime)
		printSampleUsage()
		if !strings.Contains(err.Error(), "help requested") {
			fmt.Printf("%s\n", err.Error())
		}
		return nil
	}
	if !(serverConfig.EnableRPC || serverConfig.EnableREST) {
		fmt.Printf("\nrevision: %v, vcs.time: %v\n", vcsRevision, vcsTime)
		fmt.Printf("Please specify at least one of --enable-rest or --enable-rpc\n")
		printSampleUsage()
		return nil
	}

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(serverConfig.LogLevel))
	log.Root().SetHandler(glogger)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dasImpl, dasLifecycleManager, err := arbnode.SetUpDataAvailabilityWithoutNode(ctx, &serverConfig.DAConf)
	if err != nil {
		return err
	}

	var rpcServer *http.Server
	if serverConfig.EnableRPC {
		log.Info("Starting HTTP-RPC server", "addr", serverConfig.RPCAddr, "port", serverConfig.RPCPort)

		rpcServer, err = dasrpc.StartDASRPCServer(ctx, serverConfig.RPCAddr, serverConfig.RPCPort, dasImpl)
		if err != nil {
			return err
		}
	}

	var restServer *das.RestfulDasServer
	if serverConfig.EnableREST {
		log.Info("Starting REST server", "addr", serverConfig.RESTAddr, "port", serverConfig.RESTPort)

		restServer, err = das.NewRestfulDasServer(serverConfig.RESTAddr, serverConfig.RESTPort, dasImpl)
		if err != nil {
			return err
		}
	}

	<-sigint
	dasLifecycleManager.StopAndWaitUntil(2 * time.Second)

	var err1, err2 error
	if rpcServer != nil {
		err1 = rpcServer.Shutdown(ctx)
	}

	if restServer != nil {
		err2 = restServer.Shutdown()
	}

	if err1 != nil {
		return err1
	}
	return err2
}
