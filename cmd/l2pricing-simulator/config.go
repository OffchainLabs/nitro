// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type Config interface {
	AddFlags(*pflag.FlagSet)
	Validate() error
	Print(io.Writer)
	ShouldExportCSV() bool
	ShouldPrintLine(int) bool
}

func ParseConfig(config Config, args []string) error {
	flags := pflag.NewFlagSet("l2pricing-simulator", pflag.ExitOnError)
	config.AddFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return nil
}

type CommonConfig struct {
	Verbose             bool
	ExportCSV           bool
	SequencerThroughput uint64
	SurgeDemand         uint64
	SurgeDuration       uint64
	SurgeRamp           uint64
}

func (c *CommonConfig) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&c.Verbose, "verbose", "v", false, "If set, print all data points")
	flags.BoolVar(&c.ExportCSV, "export-csv", false, "If set, print the output as csv")
	flags.Uint64Var(&c.SequencerThroughput, "sequencer-throughput", 128_000_000, "Max sequencer throughput per second")
	flags.Uint64Var(&c.SurgeDemand, "surge-gas", 200_000_000, "Amount of gas added to the backlog during peak surge")
	flags.Uint64Var(&c.SurgeDuration, "surge-duration", 30, "Surge peak duration in seconds")
	flags.Uint64Var(&c.SurgeRamp, "surge-ramp", 10, "Number of seconds surge takes to reach its peak")
}

func (c *CommonConfig) Print(w io.Writer) {
	fmt.Fprintln(w, "Verbose:\t", c.Verbose)
	fmt.Fprintln(w, "Sequencer Throughput:\t", toPrettyUint(c.SequencerThroughput))
	fmt.Fprintln(w, "Surge demand:\t", toPrettyUint(c.SurgeDemand))
	fmt.Fprintln(w, "Surge duration:\t", toPrettyUint(c.SurgeDuration))
	fmt.Fprintln(w, "Surge ramp:\t", toPrettyUint(c.SurgeRamp))
}

func (c *CommonConfig) ShouldExportCSV() bool {
	return c.ExportCSV
}

func (c *CommonConfig) Validate() error {
	return nil
}

func (c *CommonConfig) Iterations() uint64 {
	return c.SurgeDuration*2 + c.SurgeRamp*2 + 1
}

func (c *CommonConfig) ShouldPrintLine(i int) bool {
	return c.Verbose || arbmath.SaturatingUCast[uint64](i)%(c.Iterations()/10) == 0
}

type LegacyConfig struct {
	CommonConfig
	InitialBacklog   uint64
	SpeedLimit       uint64
	Inertia          uint64
	BacklogTolerance uint64
}

func (c *LegacyConfig) AddFlags(flags *pflag.FlagSet) {
	c.CommonConfig.AddFlags(flags)
	flags.Uint64Var(&c.InitialBacklog, "initial-backlog", 0, "Initial backlog")
	flags.Uint64Var(&c.SpeedLimit, "speed-limit", l2pricing.InitialSpeedLimitPerSecondV6, "Speed limit per second")
	flags.Uint64Var(&c.Inertia, "inertia", l2pricing.InitialPricingInertia, "Inertia")
	flags.Uint64Var(&c.BacklogTolerance, "backlog-tolerance", l2pricing.InitialBacklogTolerance, "Backlog tolerance")
}

func (c *LegacyConfig) Print(w io.Writer) {
	c.CommonConfig.Print(w)
	fmt.Fprintln(w, "Initial backlog:\t", toPrettyUint(c.InitialBacklog))
	fmt.Fprintln(w, "Speed limit:\t", toPrettyUint(c.SpeedLimit))
	fmt.Fprintln(w, "Inertia:\t", c.Inertia)
	fmt.Fprintln(w, "Backlog tolerance:\t", c.BacklogTolerance)
}

type ConstraintsConfig struct {
	CommonConfig
	Targets  []int64
	Windows  []int64
	Backlogs []int64
}

var DefaultConstraintConfig = ConstraintsConfig{
	Targets: []int64{60_000_000, 41_000_000, 29_000_000, 20_000_000, 14_000_000, 10_000_000},
	Windows: []int64{9, 52, 329, 2_105, 13_485, 86_400},
}

func (c *ConstraintsConfig) AddFlags(flags *pflag.FlagSet) {
	c.CommonConfig.AddFlags(flags)
	flags.Int64SliceVar(&c.Targets, "targets", DefaultConstraintConfig.Targets, "List of constraints' targets; previously speed-limit")
	flags.Int64SliceVar(&c.Windows, "windows", DefaultConstraintConfig.Windows, "List of constraints' adjustment windows; previously inertia")
	flags.Int64SliceVar(&c.Backlogs, "backlogs", DefaultConstraintConfig.Backlogs, "List of constraints' initial backlogs")
}

func (c *ConstraintsConfig) Print(w io.Writer) {
	c.CommonConfig.Print(w)
	fmt.Fprintln(w, "Number of constraints:\t", len(c.Targets))
	for i := range len(c.Targets) {
		var backlog int64
		if i < len(c.Backlogs) {
			backlog = c.Backlogs[i]
		}
		constraint := fmt.Sprintf("target=%v, window=%v, backlog=%v",
			toPrettyInt(c.Targets[i]), toPrettyInt(c.Windows[i]), toPrettyInt(backlog))
		fmt.Fprintf(w, "Constraint %v:\t%v\n", i, constraint)
	}
}

func (c *ConstraintsConfig) Validate() error {
	for _, target := range c.Targets {
		if target < 0 {
			return fmt.Errorf("invalid negative target")
		}
	}
	for _, window := range c.Windows {
		if window < 0 {
			return fmt.Errorf("invalid negative adjustment window")
		}
	}
	for _, backlog := range c.Backlogs {
		if backlog < 0 {
			return fmt.Errorf("invalid negative backlog")
		}
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("expected at least one constraint")
	}
	if len(c.Targets) != len(c.Windows) {
		return fmt.Errorf("mismatch number of targets and adjustment-windows")
	}
	if len(c.Backlogs) > len(c.Targets) {
		return fmt.Errorf("too many initial backlogs")
	}
	return nil
}
