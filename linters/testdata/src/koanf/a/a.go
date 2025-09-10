// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package a

import (
	"flag"
)

type Config struct {
	L2       int `koanf:"chain"` // want `field name: "L2" doesn't match tag name: "chain"`
	LogLevel int `koanf:"log-level"`
	LogType  int `koanf:"log-type"`
	Metrics  int `koanf:"metrics"`
	PProf    int `koanf:"pprof"`
	Node     int `koanf:"node"`
	Queue    int `koanf:"queue"`
}

// Cover using of all fields in a various way:

// Instantiating a type.
var defaultConfig = Config{
	L2:       1,
	LogLevel: 2,
}

// Instantiating a type an taking reference.
var defaultConfigPtr = &Config{
	LogType: 3,
	Metrics: 4,
}

func init() {
	defaultConfig.PProf = 5
	defaultConfig.Node, _ = 6, 0
	defaultConfigPtr.Queue = 7
}

type BatchPosterConfig struct {
	Enable  bool `koanf:"enable"`
	MaxSize int  `koanf:"max-size" reload:"hot"`
}

var DefaultBatchPosterConfig BatchPosterConfig

func BatchPosterConfigAddOptions(prefix string, f *flag.FlagSet) { // want `koanf tag name: "enabled" doesn't match the field: "Enable"` `koanf tag name: "max-sz" doesn't match the field: "MaxSize"`
	f.Bool(prefix+".enabled", DefaultBatchPosterConfig.Enable, "")
	f.Int("max-sz", DefaultBatchPosterConfig.MaxSize, "")
}

func ConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultBatchPosterConfig.Enable, "enable posting batches to l1")
	f.Int("max-size", DefaultBatchPosterConfig.MaxSize, "maximum batch size")
}

func init() {
	// Fields must be used outside flag definitions at least once.
	DefaultBatchPosterConfig.Enable = true
	DefaultBatchPosterConfig.MaxSize = 3
}
