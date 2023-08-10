// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package resourcemanager

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/spf13/pflag"
)

var (
	limitCheckDurationHistogram = metrics.NewRegisteredHistogram("arb/rpc/limitcheck/duration", nil, metrics.NewBoundedHistogramSample())
	limitCheckSuccessCounter    = metrics.NewRegisteredCounter("arb/rpc/limitcheck/success", nil)
	limitCheckFailureCounter    = metrics.NewRegisteredCounter("arb/rpc/limitcheck/failure", nil)
	errNotSupported             = errors.New("not supported")
)

// Init adds the resource manager's httpServer to a custom hook in geth.
// Geth will add it to the stack of http.Handlers so that it is run
// prior to RPC request handling.
//
// Must be run before the go-ethereum stack is set up (ethereum/go-ethereum/node.New).
func Init(conf *Config) {
	if conf.MemoryLimitPercent > 0 {
		node.WrapHTTPHandler = func(srv http.Handler) (http.Handler, error) {
			var c limitChecker
			c, err := newCgroupsMemoryLimitCheckerIfSupported(conf)
			if errors.Is(err, errNotSupported) {
				log.Error("No method for determining memory usage and limits was discovered, disabled memory limit RPC throttling")
				c = &trivialLimitChecker{}
			}

			return newHttpServer(srv, c), nil
		}
	}
}

// Config contains the configuration for resourcemanager functionality.
// Currently only a memory limit is supported, other limits may be added
// in the future.
type Config struct {
	MemoryLimitPercent int `koanf:"mem-limit-percent" reload:"hot"`
}

// DefaultConfig has the defaul resourcemanager configuration,
// all limits are disabled.
var DefaultConfig = Config{
	MemoryLimitPercent: 0,
}

// ConfigAddOptions adds the configuration options for resourcemanager.
func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Int(prefix+".mem-limit-percent", DefaultConfig.MemoryLimitPercent, "RPC calls are throttled if system memory utilization exceeds this percent value, zero (default) is disabled")
}

// httpServer implements http.Handler and wraps calls to inner with a resource
// limit check.
type httpServer struct {
	inner http.Handler
	c     limitChecker
}

func newHttpServer(inner http.Handler, c limitChecker) *httpServer {
	return &httpServer{inner: inner, c: c}
}

// ServeHTTP passes req to inner unless any configured system resource
// limit is exceeded, in which case it returns a HTTP 429 error.
func (s *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	exceeded, err := s.c.isLimitExceeded()
	limitCheckDurationHistogram.Update(time.Since(start).Nanoseconds())
	if err != nil {
		log.Error("Error checking memory limit", "err", err, "checker", s.c)
	} else if exceeded {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		limitCheckFailureCounter.Inc(1)
		return
	}

	limitCheckSuccessCounter.Inc(1)
	s.inner.ServeHTTP(w, req)
}

type limitChecker interface {
	isLimitExceeded() (bool, error)
	String() string
}

func isSupported(c limitChecker) bool {
	_, err := c.isLimitExceeded()
	return err == nil
}

// newCgroupsMemoryLimitCheckerIfSupported attempts to auto-discover whether
// Cgroups V1 or V2 is supported for checking system memory limits.
func newCgroupsMemoryLimitCheckerIfSupported(conf *Config) (*cgroupsMemoryLimitChecker, error) {
	c := newCgroupsMemoryLimitChecker(cgroupsV1MemoryFiles, conf.MemoryLimitPercent)
	if isSupported(c) {
		log.Info("Cgroups v1 detected, enabling memory limit RPC throttling")
		return c, nil
	}

	c = newCgroupsMemoryLimitChecker(cgroupsV2MemoryFiles, conf.MemoryLimitPercent)
	if isSupported(c) {
		log.Info("Cgroups v2 detected, enabling memory limit RPC throttling")
		return c, nil
	}

	return nil, errNotSupported
}

// trivialLimitChecker checks no limits, so its limits are never exceeded.
type trivialLimitChecker struct{}

func (_ trivialLimitChecker) isLimitExceeded() (bool, error) {
	return false, nil
}

func (_ trivialLimitChecker) String() string { return "trivial" }

type cgroupsMemoryFiles struct {
	limitFile, usageFile, statsFile string
	inactiveRe                      *regexp.Regexp
}

const defaultCgroupsV1MemoryDirectory = "/sys/fs/cgroup/memory/"
const defaultCgroupsV2MemoryDirectory = "/sys/fs/cgroup/"

var cgroupsV1MemoryFiles = cgroupsMemoryFiles{
	limitFile:  defaultCgroupsV1MemoryDirectory + "/memory.limit_in_bytes",
	usageFile:  defaultCgroupsV1MemoryDirectory + "/memory.usage_in_bytes",
	statsFile:  defaultCgroupsV1MemoryDirectory + "/memory.stat",
	inactiveRe: regexp.MustCompile(`total_inactive_file (\d+)`),
}
var cgroupsV2MemoryFiles = cgroupsMemoryFiles{
	limitFile:  defaultCgroupsV2MemoryDirectory + "/memory.max",
	usageFile:  defaultCgroupsV2MemoryDirectory + "/memory.current",
	statsFile:  defaultCgroupsV2MemoryDirectory + "/memory.stat",
	inactiveRe: regexp.MustCompile(`inactive_file (\d+)`),
}

type cgroupsMemoryLimitChecker struct {
	files              cgroupsMemoryFiles
	memoryLimitPercent int
}

func newCgroupsMemoryLimitChecker(files cgroupsMemoryFiles, memoryLimitPercent int) *cgroupsMemoryLimitChecker {
	return &cgroupsMemoryLimitChecker{
		files:              files,
		memoryLimitPercent: memoryLimitPercent,
	}
}

// isLimitExceeded checks if the system memory used exceeds the limit
// scaled by the configured memoryLimitPercent.
//
// See the following page for details of calculating the memory used,
// which is reported as container_memory_working_set_bytes in prometheus:
// https://mihai-albert.com/2022/02/13/out-of-memory-oom-in-kubernetes-part-3-memory-metrics-sources-and-tools-to-collect-them/
func (c *cgroupsMemoryLimitChecker) isLimitExceeded() (bool, error) {
	var limit, usage, inactive int
	var err error
	if limit, err = readIntFromFile(c.files.limitFile); err != nil {
		return false, err
	}
	if usage, err = readIntFromFile(c.files.usageFile); err != nil {
		return false, err
	}
	if inactive, err = readInactive(c.files.statsFile, c.files.inactiveRe); err != nil {
		return false, err
	}
	return usage-inactive >= ((limit * c.memoryLimitPercent) / 100), nil
}

func (c cgroupsMemoryLimitChecker) String() string {
	return "CgroupsMemoryLimitChecker"
}

func readIntFromFile(fileName string) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}

	var limit int
	if _, err = fmt.Fscanf(file, "%d", &limit); err != nil {
		return 0, err
	}
	return limit, nil
}

func readInactive(fileName string, re *regexp.Regexp) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		matches := re.FindStringSubmatch(line)

		if len(matches) >= 2 {
			inactive, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, err
			}
			return inactive, nil
		}
	}

	return 0, errors.New("total_inactive_file not found in " + fileName)
}
