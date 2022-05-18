// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	koanfjson "github.com/knadh/koanf/parsers/json"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util"
	"github.com/offchainlabs/nitro/das"
	flag "github.com/spf13/pflag"
	"os"
	"os/signal"
	"syscall"
)

type RESTServerConfig struct {
	Addr       string                 `koanf:"addr"`
	Port       uint64                 `koanf:"port"`
	LogLevel   int                    `koanf:"log-level"`
	StorageDir string                 `koanf:"storage-dir"`
	ConfConfig genericconf.ConfConfig `koanf:"conf"`
}

func main() {
	if err := startup(); err != nil {
		log.Error("Error running DAServer", "err", err)
	}
}

func parseRESTServer(args []string) (*RESTServerConfig, error) {
	f := flag.NewFlagSet("dasrest", flag.ContinueOnError)
	f.String("addr", "localhost:9877", "address to listen on ")
	f.Uint64("port", 9877, "port number to listen on")
	f.Int("log-level", int(log.LvlInfo), "log level")
	f.String("storage-dir", "", "directory path for DAS storage files")
	genericconf.ConfConfigAddOptions("conf", f)

	k, err := util.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var serverConfig RESTServerConfig
	if err := util.EndCommonParse(k, &serverConfig); err != nil {
		return nil, err
	}

	if serverConfig.ConfConfig.Dump {
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
	serverConfig, err := parseRESTServer(os.Args[1:])
	if err != nil {
		return err
	}

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	storage := das.NewLocalDiskStorageService(serverConfig.StorageDir)
	restServer := das.NewRestfulDasServerHTTP(serverConfig.Addr, serverConfig.Port, storage)

	<-sigint
	return restServer.Shutdown()
}
