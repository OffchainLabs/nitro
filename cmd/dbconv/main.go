package main

import (
	"context"
	"os"
	"time"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stats := conv.Stats()
				log.Info("Progress:\n", "entries", stats.Entries(), "entires/s", stats.EntriesPerSecond(), "avg e/s", stats.AverageEntriesPerSecond(), "MB/s", float64(stats.Bytes())/1024/1024, "MB/s", stats.BytesPerSecond()/1024/1024, "avg MB/s", stats.AverageBytesPerSecond()/1024/1024, "forks", stats.Forks(), "threads", stats.Threads(), "elapsed", stats.Elapsed())

			case <-ctx.Done():
				return
			}
		}
	}()

	err = conv.Convert(ctx)
	if err != nil {
		panic(err)
	}
	stats := conv.Stats()
	log.Info("Conversion finished.", "entries", stats.Entries(), "avg e/s", stats.AverageEntriesPerSecond(), "avg MB/s", stats.AverageBytesPerSecond()/1024/1024, "elapsed", stats.Elapsed())
}
