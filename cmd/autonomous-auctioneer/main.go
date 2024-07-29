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

	// stackConf := DefaultValidationNodeStackConfig
	// stackConf.DataDir = "" // ephemeral
	// nodeConfig.HTTP.Apply(&stackConf)
	// nodeConfig.WS.Apply(&stackConf)
	// nodeConfig.Auth.Apply(&stackConf)
	// nodeConfig.IPC.Apply(&stackConf)
	// stackConf.P2P.ListenAddr = ""
	// stackConf.P2P.NoDial = true
	// stackConf.P2P.NoDiscovery = true
	// vcsRevision, strippedRevision, vcsTime := confighelpers.GetVersion()
	// stackConf.Version = strippedRevision

	// pathResolver := func(workdir string) func(string) string {
	// 	if workdir == "" {
	// 		workdir, err = os.Getwd()
	// 		if err != nil {
	// 			log.Warn("Failed to get workdir", "err", err)
	// 		}
	// 	}
	// 	return func(path string) string {
	// 		if filepath.IsAbs(path) {
	// 			return path
	// 		}
	// 		return filepath.Join(workdir, path)
	// 	}
	// }

	// err = genericconf.InitLog(nodeConfig.LogType, nodeConfig.LogLevel, &nodeConfig.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error initializing logging: %v\n", err)
	// 	return 1
	// }
	// if stackConf.JWTSecret == "" && stackConf.AuthAddr != "" {
	// 	filename := pathResolver(nodeConfig.Persistent.GlobalConfig)("jwtsecret")
	// 	if err := genericconf.TryCreatingJWTSecret(filename); err != nil {
	// 		log.Error("Failed to prepare jwt secret file", "err", err)
	// 		return 1
	// 	}
	// 	stackConf.JWTSecret = filename
	// }

	// log.Info("Running Arbitrum nitro validation node", "revision", vcsRevision, "vcs.time", vcsTime)

	// liveNodeConfig := genericconf.NewLiveConfig[*ValidationNodeConfig](args, nodeConfig, ParseNode)
	// liveNodeConfig.SetOnReloadHook(func(oldCfg *ValidationNodeConfig, newCfg *ValidationNodeConfig) error {

	// 	return genericconf.InitLog(newCfg.LogType, newCfg.LogLevel, &newCfg.FileLogging, pathResolver(nodeConfig.Persistent.LogDir))
	// })

	// valnode.EnsureValidationExposedViaAuthRPC(&stackConf)

	// stack, err := node.New(&stackConf)
	// if err != nil {
	// 	flag.Usage()
	// 	log.Crit("failed to initialize geth stack", "err", err)
	// }

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
