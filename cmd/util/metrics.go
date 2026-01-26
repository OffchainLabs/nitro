// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package util

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"

	"github.com/offchainlabs/nitro/cmd/genericconf"
)

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func StartMetrics(metricsEnable bool, pprofEnable bool, metricsServerConfig *genericconf.MetricsServerConfig, pprofConfig *genericconf.PProf) error {
	mAddr := fmt.Sprintf("%v:%v", metricsServerConfig.Addr, metricsServerConfig.Port)
	pAddr := fmt.Sprintf("%v:%v", pprofConfig.Addr, pprofConfig.Port)
	if metricsEnable && pprofEnable && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if metricsEnable {
		log.Info("Enabling metrics collection")
		metrics.Enable()
		go metrics.CollectProcessMetrics(metricsServerConfig.UpdateInterval)
		exp.Setup(mAddr)
	}
	if pprofEnable {
		genericconf.StartPprof(pAddr)
	}
	return nil
}
