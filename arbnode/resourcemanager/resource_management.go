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
)

func Init(conf *Config) {
	if conf.MemoryLimitPercent > 0 {
		node.WrapHTTPHandler = func(srv http.Handler) (http.Handler, error) {
			return newHttpServer(srv, newLimitChecker(conf)), nil
		}
	}
}

type Config struct {
	MemoryLimitPercent int `koanf:"mem-limit-percent" reload:"hot"`
}

var DefaultConfig = Config{
	MemoryLimitPercent: 0,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Int(prefix+".mem-limit-percent", DefaultConfig.MemoryLimitPercent, "RPC calls are throttled if system memory utilization exceeds this percent value, zero (default) is disabled")
}

type httpServer struct {
	inner http.Handler
	c     limitChecker
}

func newHttpServer(inner http.Handler, c limitChecker) *httpServer {
	return &httpServer{inner: inner, c: c}
}

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

func newLimitChecker(conf *Config) limitChecker {
	{
		c := newCgroupsV1MemoryLimitChecker(DefaultCgroupsV1MemoryDirectory, conf.MemoryLimitPercent)
		if isSupported(c) {
			log.Info("Cgroups v1 detected, enabling memory limit RPC throttling")
			return c
		}
	}

	log.Error("No method for determining memory usage and limits was discovered, disabled memory limit RPC throttling")
	return &trivialLimitChecker{}
}

type trivialLimitChecker struct{}

func (_ trivialLimitChecker) isLimitExceeded() (bool, error) {
	return false, nil
}

func (_ trivialLimitChecker) String() string { return "trivial" }

const DefaultCgroupsV1MemoryDirectory = "/sys/fs/cgroup/memory/"

type cgroupsV1MemoryLimitChecker struct {
	cgroupDir          string
	memoryLimitPercent int

	limitFile, usageFile, statsFile string
}

func newCgroupsV1MemoryLimitChecker(cgroupDir string, memoryLimitPercent int) *cgroupsV1MemoryLimitChecker {
	return &cgroupsV1MemoryLimitChecker{
		cgroupDir:          cgroupDir,
		memoryLimitPercent: memoryLimitPercent,
		limitFile:          cgroupDir + "/memory.limit_in_bytes",
		usageFile:          cgroupDir + "/memory.usage_in_bytes",
		statsFile:          cgroupDir + "/memory.stat",
	}
}

func isSupported(c limitChecker) bool {
	_, err := c.isLimitExceeded()
	return err == nil
}

func (c *cgroupsV1MemoryLimitChecker) isLimitExceeded() (bool, error) {
	var limit, usage, inactive int
	var err error
	limit, err = c.readIntFromFile(c.limitFile)
	if err != nil {
		return false, err
	}
	usage, err = c.readIntFromFile(c.usageFile)
	if err != nil {
		return false, err
	}
	inactive, err = c.readInactive()
	if err != nil {
		return false, err
	}
	return usage-inactive >= ((limit * c.memoryLimitPercent) / 100), nil
}

func (c cgroupsV1MemoryLimitChecker) readIntFromFile(fileName string) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}

	var limit int
	_, err = fmt.Fscanf(file, "%d", &limit)
	if err != nil {
		return 0, err
	}
	return limit, nil
}

func (c cgroupsV1MemoryLimitChecker) readInactive() (int, error) {
	file, err := os.Open(c.statsFile)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`total_inactive_file (\d+)`)
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

	return 0, errors.New("total_inactive_file not found in " + c.statsFile)
}

func (c cgroupsV1MemoryLimitChecker) String() string {
	return "CgroupsV1MemoryLimitChecker"
}
