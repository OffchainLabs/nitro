package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
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
	return &config, config.Validate()
}

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n\n", name)
}

func printProgress(conv *dbconv.DBConverter) {
	stats := conv.Stats()
	fmt.Printf("Progress:\n")
	fmt.Printf("\tprocessed entries: %d\n", stats.Entries())
	fmt.Printf("\tprocessed data (MB): %d\n", stats.Bytes()/1024/1024)
	fmt.Printf("\telapsed:\t%v\n", stats.Elapsed())
	fmt.Printf("\tcurrent:\t%.3e entries/s\t%.3f MB/s\n", stats.EntriesPerSecond()/1000, stats.BytesPerSecond()/1024/1024)
	fmt.Printf("\taverage:\t%.3e entries/s\t%.3f MB/s\n", stats.AverageEntriesPerSecond()/1000, stats.AverageBytesPerSecond()/1024/1024)
}

func main() {
	args := os.Args[1:]
	config, err := parseDBConv(args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
	}

	err = genericconf.InitLog(config.LogType, config.LogLevel, &genericconf.FileLoggingConfig{Enable: false}, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
		os.Exit(1)
	}

	if config.Metrics {
		go metrics.CollectProcessMetrics(config.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", config.MetricsServer.Addr, config.MetricsServer.Port))
	}

	conv := dbconv.NewDBConverter(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				printProgress(conv)
			case <-ctx.Done():
				return
			}
		}
	}()

	if config.Convert {
		err = conv.Convert(ctx)
		if err != nil {
			log.Error("Conversion error", "err", err)
			os.Exit(1)
		}
		stats := conv.Stats()
		log.Info("Conversion finished.", "entries", stats.Entries(), "MB", stats.Bytes()/1024/1024, "avg entries/s", fmt.Sprintf("%.3e", stats.AverageEntriesPerSecond()/1000), "avg MB/s", stats.AverageBytesPerSecond()/1024/1024, "elapsed", stats.Elapsed())
	}

	if config.Compact {
		ticker.Stop()
		err = conv.CompactDestination()
		if err != nil {
			log.Error("Compaction error", "err", err)
			os.Exit(1)
		}
	}

	if config.Verify != "" {
		ticker.Reset(10 * time.Second)
		err = conv.Verify(ctx)
		if err != nil {
			log.Error("Verification error", "err", err)
			os.Exit(1)
		}
		stats := conv.Stats()
		log.Info("Verification completed successfully.", "elapsed", stats.Elapsed())
	}
}
