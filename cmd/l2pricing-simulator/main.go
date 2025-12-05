// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// l2pricing-simulator is a command-line tool that simulates the behavior of Nitro's l2 pricing algorithm.
package main

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var minBaseFee = big.NewInt(params.GWei)

func newPricingState(arbosVersion uint64) (*l2pricing.L2PricingState, error) {
	storage := storage.NewMemoryBacked(burn.NewSystemBurner(nil, false))
	err := l2pricing.InitializeL2PricingState(storage)
	if err != nil {
		return nil, err
	}
	pricing := l2pricing.OpenL2PricingState(storage, arbosVersion)
	_ = pricing.SetMinBaseFeeWei(minBaseFee)
	_ = pricing.SetBaseFeeWei(minBaseFee)
	return pricing, nil
}

func runLegacyModel(args []string) error {
	var config LegacyConfig
	if err := ParseConfig(&config, args); err != nil {
		return err
	}

	pricing, err := newPricingState(params.ArbosVersion_40)
	if err != nil {
		return err
	}

	_ = pricing.SetGasBacklog(config.InitialBacklog)
	_ = pricing.SetSpeedLimitPerSecond(config.SpeedLimit)
	_ = pricing.SetPricingInertia(config.Inertia)
	_ = pricing.SetBacklogTolerance(config.BacklogTolerance)

	gasSimulator := NewGasSimulator(config.CommonConfig, config.SpeedLimit)
	results := []Result{}
	for i := range config.Iterations() {
		baseFee, _ := pricing.BaseFeeWei()
		gas := gasSimulator.compute(i, baseFee)
		_ = pricing.GrowBacklog(gas, multigas.ComputationGas(gas))
		pricing.UpdatePricingModel(1)
		baseFee, _ = pricing.BaseFeeWei()
		results = append(results, Result{
			baseFee:  baseFee,
			gasRatio: float64(gas) / float64(config.SpeedLimit),
		})
	}

	printOutput(&config, results)
	return nil
}

func runConstraintsModel(args []string) error {
	var config ConstraintsConfig
	if err := ParseConfig(&config, args); err != nil {
		return err
	}

	pricing, err := newPricingState(params.ArbosVersion_MultiConstraintFix)
	if err != nil {
		return err
	}

	longTermWindow := uint64(0)
	longTermTarget := uint64(0)
	numConstraints := len(config.Targets)
	for i := range numConstraints {
		target := arbmath.SaturatingUCast[uint64](config.Targets[i])
		window := arbmath.SaturatingUCast[uint64](config.Windows[i])
		var backlog uint64
		if i < len(config.Backlogs) {
			backlog = arbmath.SaturatingUCast[uint64](config.Backlogs[i])
		}
		if err := pricing.AddGasConstraint(target, window, backlog); err != nil {
			return fmt.Errorf("failed to add constraint: %w", err)
		}
		if window > longTermWindow {
			longTermWindow = window
			longTermTarget = target
		}
	}

	gasSimulator := NewGasSimulator(config.CommonConfig, longTermTarget)

	results := []Result{}
	for i := range config.Iterations() {
		baseFee, _ := pricing.BaseFeeWei()
		gas := gasSimulator.compute(i, baseFee)
		_ = pricing.GrowBacklog(gas, multigas.ComputationGas(gas))
		pricing.UpdatePricingModel(1)
		baseFee, _ = pricing.BaseFeeWei()
		results = append(results, Result{
			baseFee:  baseFee,
			gasRatio: float64(gas) / float64(longTermTarget),
		})
	}

	printOutput(&config, results)
	return nil
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: l2pricing-simulator [legacy|constraints] ...")
		os.Exit(1)
	}
	var err error
	switch strings.ToLower(args[1]) {
	case "legacy":
		err = runLegacyModel(args[2:])
	case "constraints":
		err = runConstraintsModel(args[2:])
	default:
		err = fmt.Errorf("unknown command '%s', valid commands are: legacy and constraints", args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
