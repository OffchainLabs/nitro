package genericconf

import (
	"fmt"
	"net/http"

	// Blank import pprof registers its HTTP handlers.
	_ "net/http/pprof" // #nosec G108

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
)

func StartPprof(address string) {
	exp.Exp(metrics.DefaultRegistry)
	log.Info("Starting metrics server with pprof", "addr", fmt.Sprintf("http://%s/debug/metrics", address))
	log.Info("Pprof endpoint", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
	go func() {
		// #nosec
		if err := http.ListenAndServe(address, http.DefaultServeMux); err != nil {
			log.Error("Failure in running pprof server", "err", err)
		}
	}()
}
