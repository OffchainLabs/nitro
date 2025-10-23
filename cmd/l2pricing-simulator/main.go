// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"fmt"
	"math/big"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var minBaseFee = big.NewInt(params.GWei)

type LegacyModelConfig struct {
	SpeedLimit       uint64 `koanf:"speed-limit"`
	Inertia          uint64 `koanf:"inertia"`
	BacklogTolerance uint64 `koanf:"backlog-tolerance"`
	InitialBacklog   uint64 `koanf:"initial-backlog"`
	TrafficPerSec    uint64 `koanf:"traffic-per-sec"`
	Iterations       uint64 `koanf:"iterations"`
	PrintAll         bool   `koanf:"print-all"`
}

func ParseLegacyConfig(args []string) (*LegacyModelConfig, error) {
	f := pflag.NewFlagSet("l2pricing simulator", pflag.ContinueOnError)
	f.Uint64("speed-limit", l2pricing.InitialSpeedLimitPerSecondV6, "Speed limit per second")
	f.Uint64("inertia", l2pricing.InitialPricingInertia, "Inertia")
	f.Uint64("backlog-tolerance", l2pricing.InitialBacklogTolerance, "Backlog tolerance")
	f.Uint64("initial-backlog", 0, "Initial backlog")
	f.Uint64("traffic-per-sec", 0, "Generated traffic per sec")
	f.Uint64("iterations", 50, "Number of seconds that will be simulated")
	f.Bool("print-all", false, "If set, print all test iterations")

	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config LegacyModelConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *LegacyModelConfig) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Speed limit:\t", toPrettyInt(c.SpeedLimit))
	fmt.Fprintln(w, "Inertia:\t", c.Inertia)
	fmt.Fprintln(w, "Backlog tolerance:\t", c.BacklogTolerance)
	fmt.Fprintln(w, "Initial backlog:\t", toPrettyInt(c.InitialBacklog))
	fmt.Fprintln(w, "Traffic per second:\t", toPrettyInt(c.TrafficPerSec))
	fmt.Fprintln(w, "Iterations:\t", c.Iterations)
	w.Flush()
}

type IterationResult struct {
	baseFee *big.Int
	backlog uint64
}

func printResult(results []IterationResult, printAll bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintln(w, "i\tbaseFee\tbacklog\t")
	fmt.Fprintln(w, "---\t---\t---\t")
	for i, result := range results {
		if !printAll && i >= 10 && i+10 < len(results) {
			continue
		}
		fmt.Fprintf(w, "%v\t", i+1)
		fmt.Fprintf(w, "%v\t", toPrettyInt(result.backlog))
		fmt.Fprintf(w, "%v\t\n", toGwei(result.baseFee))
	}
	w.Flush()
}

func toGwei(wei *big.Int) string {
	gweiDivisor := big.NewInt(params.GWei)
	weiRat := new(big.Rat).SetInt(wei)
	gweiDivisorRat := new(big.Rat).SetInt(gweiDivisor)
	gweiRat := new(big.Rat).Quo(weiRat, gweiDivisorRat)
	return gweiRat.FloatString(3)
}

func toPrettyInt(v uint64) string {
	if v == 0 {
		return "0"
	}
	parts := []string{}
	for v > 1000 {
		parts = append(parts, fmt.Sprintf("%03d", v%1000))
		v = v / 1000
	}
	if v > 0 {
		parts = append(parts, fmt.Sprint(v))
	}
	slices.Reverse(parts)
	return strings.Join(parts, ",")
}

func newPricingState() (*l2pricing.L2PricingState, error) {
	storage := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := l2pricing.InitializeL2PricingState(storage)
	if err != nil {
		return nil, err
	}
	return l2pricing.OpenL2PricingState(storage), nil
}

func runLegacyModel(args []string) error {
	config, err := ParseLegacyConfig(args)
	if err != nil {
		return err
	}
	config.Print()

	fmt.Println()

	pricing, err := newPricingState()
	if err != nil {
		return err
	}

	_ = pricing.SetMinBaseFeeWei(minBaseFee)
	_ = pricing.SetSpeedLimitPerSecond(config.SpeedLimit)
	_ = pricing.SetPricingInertia(config.Inertia)
	_ = pricing.SetBacklogTolerance(config.BacklogTolerance)
	_ = pricing.SetGasBacklog(config.InitialBacklog)

	result := make([]IterationResult, config.Iterations)
	for i := range config.Iterations {
		_ = pricing.AddToGasPool(-arbmath.SaturatingCast[int64](config.TrafficPerSec))
		pricing.UpdatePricingModel(nil, 1, false)
		result[i].baseFee, _ = pricing.BaseFeeWei()
		result[i].backlog, _ = pricing.GasBacklog()
	}

	printResult(result, config.PrintAll)

	return nil
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: l2pricing-simulator [legacy|multi-constraints] ...")
		os.Exit(1)
	}
	var err error
	switch strings.ToLower(args[1]) {
	case "legacy":
		err = runLegacyModel(args[2:])
	default:
		err = fmt.Errorf("unknown command '%s', valid commands are: legacy and multi-constraints", args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
