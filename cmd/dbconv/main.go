package main

import (
	"context"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	flag "github.com/spf13/pflag"
)

func parseDBConv(args []string) (*dbconv.DBConvConfig, error) {
	f := flag.NewFlagSet("dbconv", flag.ContinueOnError)
	dbconv.DBConvConfigAddOptions(f)
	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}
	var config dbconv.DBConvConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	args := os.Args[1:]
	config, err := parseDBConv(args)
	if err != nil {
		panic(err)
	}
	err = genericconf.InitLog("plaintext", log.LvlDebug, &genericconf.FileLoggingConfig{Enable: false}, nil)
	if err != nil {
		panic(err)
	}

	conv := dbconv.NewDBConverter(config)
	err = conv.Convert(context.Background())
	if err != nil {
		panic(err)
	}
}
