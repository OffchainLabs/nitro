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
	"strings"
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
	nitroMemLimit               = metrics.GetOrRegisterGauge("arb/memory/limit", nil)
	nitroMemUsage               = metrics.GetOrRegisterGauge("arb/memory/usage", nil)
	errNotSupported             = errors.New("not supported")
)

// Init adds the resource manager's httpServer to a custom hook in geth.
// Geth will add it to the stack of http.Handlers so that it is run
// prior to RPC request handling.
//
// Must be run before the go-ethereum stack is set up (ethereum/go-ethereum/node.New).
func Init(conf *Config) error {
	if conf.MemFreeLimit == "" {
		return nil
	}

	limit, err := parseMemLimit(conf.MemFreeLimit)
	if err != nil {
		return err
	}

	node.WrapHTTPHandler = func(srv http.Handler) (http.Handler, error) {
		var c limitChecker
		c, err := newCgroupsMemoryLimitCheckerIfSupported(limit)
		if errors.Is(err, errNotSupported) {
			log.Error("no method for determining memory usage and limits was discovered, disabled memory limit RPC throttling")
			c = &trivialLimitChecker{}
		}

		return newHttpServer(srv, c), nil
	}
	return nil
}

func parseMemLimit(limitStr string) (int, error) {
	var (
		limit int = 1
		s     string
	)
	if _, err := fmt.Sscanf(limitStr, "%d%s", &limit, &s); err != nil {
		return 0, err
	}

	switch strings.ToUpper(s) {
	case "K", "KB":
		limit <<= 10
	case "M", "MB":
		limit <<= 20
	case "G", "GB":
		limit <<= 30
	case "T", "TB":
		limit <<= 40
	case "B":
	default:
		return 0, fmt.Errorf("unsupported memory limit suffix string %s", s)
	}

	return limit, nil
}

// Config contains the configuration for resourcemanager functionality.
// Currently only a memory limit is supported, other limits may be added
// in the future.
type Config struct {
	MemFreeLimit string `koanf:"mem-free-limit" reload:"hot"`
}

// DefaultConfig has the defaul resourcemanager configuration,
// all limits are disabled.
var DefaultConfig = Config{
	MemFreeLimit: "",
}

// ConfigAddOptions adds the configuration options for resourcemanager.
func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".mem-free-limit", DefaultConfig.MemFreeLimit, "RPC calls are throttled if free system memory excluding the page cache is below this amount, expressed in bytes or multiples of bytes with suffix B, K, M, G. The limit should be set such that sufficient free memory is left for the page cache in order for the system to be performant")
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
		log.Error("error checking memory limit", "err", err, "checker", s.c)
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
func newCgroupsMemoryLimitCheckerIfSupported(memLimitBytes int) (*cgroupsMemoryLimitChecker, error) {
	c := newCgroupsMemoryLimitChecker(cgroupsV1MemoryFiles, memLimitBytes)
	if isSupported(c) {
		log.Info("Cgroups v1 detected, enabling memory limit RPC throttling")
		return c, nil
	}

	c = newCgroupsMemoryLimitChecker(cgroupsV2MemoryFiles, memLimitBytes)
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
	activeRe, inactiveRe            *regexp.Regexp
}

const defaultCgroupsV1MemoryDirectory = "/sys/fs/cgroup/memory/"
const defaultCgroupsV2MemoryDirectory = "/sys/fs/cgroup/"

var cgroupsV1MemoryFiles = cgroupsMemoryFiles{
	limitFile:  defaultCgroupsV1MemoryDirectory + "/memory.limit_in_bytes",
	usageFile:  defaultCgroupsV1MemoryDirectory + "/memory.usage_in_bytes",
	statsFile:  defaultCgroupsV1MemoryDirectory + "/memory.stat",
	activeRe:   regexp.MustCompile(`^total_active_file (\d+)`),
	inactiveRe: regexp.MustCompile(`^total_inactive_file (\d+)`),
}
var cgroupsV2MemoryFiles = cgroupsMemoryFiles{
	limitFile:  defaultCgroupsV2MemoryDirectory + "/memory.max",
	usageFile:  defaultCgroupsV2MemoryDirectory + "/memory.current",
	statsFile:  defaultCgroupsV2MemoryDirectory + "/memory.stat",
	activeRe:   regexp.MustCompile(`^active_file (\d+)`),
	inactiveRe: regexp.MustCompile(`^inactive_file (\d+)`),
}

type cgroupsMemoryLimitChecker struct {
	files         cgroupsMemoryFiles
	memLimitBytes int
}

func newCgroupsMemoryLimitChecker(files cgroupsMemoryFiles, memLimitBytes int) *cgroupsMemoryLimitChecker {
	return &cgroupsMemoryLimitChecker{
		files:         files,
		memLimitBytes: memLimitBytes,
	}
}

// isLimitExceeded checks if the system memory free is less than the limit.
// It returns true if the limit is exceeded.
//
// container_memory_working_set_bytes in prometheus is calculated as
// memory.usage_in_bytes - inactive page cache bytes, see
// https://mihai-albert.com/2022/02/13/out-of-memory-oom-in-kubernetes-part-3-memory-metrics-sources-and-tools-to-collect-them/
// This metric is used by kubernetes to report memory in use by the pod,
// but memory.usage_in_bytes also includes the active page cache, which
// can be evicted by the kernel when more memory is needed, see
// https://github.com/kubernetes/kubernetes/issues/43916
// The kernel cannot be guaranteed to move a page from a file from
// active to inactive even when the file is closed, or Nitro is exited.
// For larger chains, Nitro's page cache can grow quite large due to
// the large amount of state that is randomly accessed from disk as each
// block is added. So in checking the limit we also include the active
// page cache.
//
// The limit should be set such that the system has a reasonable amount of
// free memory for the page cache, to avoid cache thrashing on chain state
// access. How much "reasonable" is will depend on access patterns, state
// size, and your application's tolerance for latency.
func (c *cgroupsMemoryLimitChecker) isLimitExceeded() (bool, error) {
	var limit, usage, active, inactive int
	var err error
	if limit, err = readIntFromFile(c.files.limitFile); err != nil {
		return false, err
	}
	if usage, err = readIntFromFile(c.files.usageFile); err != nil {
		return false, err
	}
	if active, err = readFromMemStats(c.files.statsFile, c.files.activeRe); err != nil {
		return false, err
	}
	if inactive, err = readFromMemStats(c.files.statsFile, c.files.inactiveRe); err != nil {
		return false, err
	}

	memLimit := limit - c.memLimitBytes
	memUsage := usage - (active + inactive)
	nitroMemLimit.Update(int64(memLimit))
	nitroMemUsage.Update(int64(memUsage))

	return memUsage >= memLimit, nil
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

func readFromMemStats(fileName string, re *regexp.Regexp) (int, error) {
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
