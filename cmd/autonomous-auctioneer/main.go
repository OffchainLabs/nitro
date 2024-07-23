package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

func main() {
	os.Exit(mainImpl())
}

// Checks metrics and PProf flag, runs them if enabled.
// Note: they are separate so one can enable/disable them as they wish, the only
// requirement is that they can't run on the same address and port.
func startMetrics() error {
	// mAddr := fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port)
	// pAddr := fmt.Sprintf("%v:%v", cfg.PprofCfg.Addr, cfg.PprofCfg.Port)
	// if cfg.Metrics && !metrics.Enabled {
	// 	return fmt.Errorf("metrics must be enabled via command line by adding --metrics, json config has no effect")
	// }
	// if cfg.Metrics && cfg.PProf && mAddr == pAddr {
	// 	return fmt.Errorf("metrics and pprof cannot be enabled on the same address:port: %s", mAddr)
	// }
	// if cfg.Metrics {
	go metrics.CollectProcessMetrics(time.Second)
	// exp.Setup(fmt.Sprintf("%v:%v", cfg.MetricsServer.Addr, cfg.MetricsServer.Port))
	// }
	// if cfg.PProf {
	// 	genericconf.StartPprof(pAddr)
	// }
	return nil
}

func mainImpl() int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	_ = ctx

	if err := startMetrics(); err != nil {
		log.Error("Error starting metrics", "error", err)
		return 1
	}

	fatalErrChan := make(chan error, 10)
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case err := <-fatalErrChan:
		log.Error("shutting down due to fatal error", "err", err)
		defer log.Error("shut down due to fatal error", "err", err)
		exitCode = 1
	case <-sigint:
		log.Info("shutting down because of sigint")
	}
	// cause future ctrl+c's to panic
	close(sigint)
	return exitCode
}
