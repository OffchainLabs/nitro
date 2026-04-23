// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package util

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/cmd/genericconf"
)

type MetricsPProfOpts struct {
	Metrics       bool                            `koanf:"metrics"`
	MetricsServer genericconf.MetricsServerConfig `koanf:"metrics-server"`
	PProf         bool                            `koanf:"pprof"`
	PprofCfg      genericconf.PProf               `koanf:"pprof-cfg"`
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func StartMetricsAndPProf(opts *MetricsPProfOpts) error {
	mAddr := fmt.Sprintf("%v:%v", opts.MetricsServer.Addr, opts.MetricsServer.Port)
	pAddr := fmt.Sprintf("%v:%v", opts.PprofCfg.Addr, opts.PprofCfg.Port)
	if opts.Metrics && opts.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if opts.Metrics {
		log.Info("Enabling metrics collection")
		metrics.Enable()
		go metrics.CollectProcessMetrics(opts.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", opts.MetricsServer.Addr, opts.MetricsServer.Port))
	}
	if opts.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}

func ReadChainConfig(gen *core.Genesis) (*params.ChainConfig, []byte, error) {
	// 1. Validate that the correct fields are used
	if gen.Config != nil { //nolint:staticcheck // we want to explicitly check that the deprecated field is not used
		return nil, nil, errors.New("`config` field is deprecated and not supported; use `serializedChainConfig` instead")
	}
	if gen.SerializedChainConfig == "" {
		return nil, nil, errors.New("serialized chain config was not set (`serializedChainConfig`)")
	}
	// 2. Deserialize the chain config
	chainConfig, err := gen.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	return chainConfig, []byte(gen.SerializedChainConfig), nil
}
