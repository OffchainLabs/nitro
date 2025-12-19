package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/timeboost"
)

func printSampleUsage(name string) {
	fmt.Printf("Sample usage: %s --help \n", name)
}

func main() {
	if err := mainImpl(); err != nil {
		log.Error("Error running bidder-client", "err", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func mainImpl() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	args := os.Args[1:]
	bidderClientConfig, err := parseBidderClientArgs(ctx, args)
	if err != nil {
		confighelpers.PrintErrorAndExit(err, printSampleUsage)
		return err
	}

	configFetcher := func() *timeboost.BidderClientConfig {
		return bidderClientConfig
	}

	bidderClient, err := timeboost.NewBidderClient(ctx, configFetcher)
	if err != nil {
		return err
	}

	if bidderClientConfig.DepositGwei > 0 && bidderClientConfig.BidGwei > 0 {
		return errors.New("--deposit-gwei and --bid-gwei can't both be set, either make a deposit or a bid")
	}

	if bidderClientConfig.DepositGwei > 0 {
		err = bidderClient.Deposit(ctx, big.NewInt(int64(bidderClientConfig.DepositGwei)*1_000_000_000))
		if err == nil {
			log.Info("Deposit successful")
		}
		return err
	}

	if bidderClientConfig.BidGwei > 0 {
		bidderClient.Start(ctx)
		bid, err := bidderClient.Bid(ctx, big.NewInt(int64(bidderClientConfig.BidGwei)*1_000_000_000), common.Address{})
		if err == nil {
			log.Info("Bid submitted successfully", "bid", bid)
		}
		return err
	}

	return errors.New("select one of --deposit-gwei or --bid-gwei")
}

func parseBidderClientArgs(ctx context.Context, args []string) (*timeboost.BidderClientConfig, error) {
	f := pflag.NewFlagSet("", pflag.ContinueOnError)

	timeboost.BidderClientConfigAddOptions(f)

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	err = confighelpers.ApplyOverrides(f, k)
	if err != nil {
		return nil, err
	}

	var cfg timeboost.BidderClientConfig
	if err := confighelpers.EndCommonParse(k, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
