package util

import (
	"fmt"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
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
	if opts.Metrics && !metrics.Enabled {
		return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	}
	if opts.Metrics && opts.PProf && mAddr == pAddr {
		return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	}
	if opts.Metrics {
		go metrics.CollectProcessMetrics(opts.MetricsServer.UpdateInterval)
		exp.Setup(fmt.Sprintf("%v:%v", opts.MetricsServer.Addr, opts.MetricsServer.Port))
	}
	if opts.PProf {
		genericconf.StartPprof(pAddr)
	}
	return nil
}
