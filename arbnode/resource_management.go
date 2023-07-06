// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	flag "github.com/spf13/pflag"
)

func InitResourceManagement(conf *ResourceManagementConfig) {
	if conf.MemoryLimitPercent > 0 {
		node.WrapHTTPHandler = func(srv http.Handler) (http.Handler, error) {
			return newResourceManagementHttpServer(srv, newLimitChecker(conf)), nil
		}
	}
}

type ResourceManagementConfig struct {
	MemoryLimitPercent int `koanf:"mem-limit-percent" reload:"hot"`
}

var DefaultResourceManagementConfig = ResourceManagementConfig{
	MemoryLimitPercent: 0,
}

func ResourceManagementConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".mem-limit-percent", DefaultResourceManagementConfig.MemoryLimitPercent, "RPC calls are throttled if system memory utilization exceeds this percent value, zero (default) is disabled")
}

type resourceManagementHttpServer struct {
	inner http.Handler
	c     limitChecker
}

func newResourceManagementHttpServer(inner http.Handler, c limitChecker) *resourceManagementHttpServer {
	return &resourceManagementHttpServer{inner: inner, c: c}
}

func (s *resourceManagementHttpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	exceeded, err := s.c.isLimitExceeded()
	if err != nil {
		log.Error("Error checking memory limit", "err", err, "checker", s.c)
	} else if exceeded {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	log.Info("Limit not exceeded, serving request.")
	s.inner.ServeHTTP(w, req)
}

type limitChecker interface {
	isLimitExceeded() (bool, error)
	String() string
}

func newLimitChecker(conf *ResourceManagementConfig) limitChecker {
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
	limit, err = c.getIntFromFile(c.limitFile)
	if err != nil {
		return false, err
	}
	usage, err = c.getIntFromFile(c.usageFile)
	if err != nil {
		return false, err
	}
	inactive, err = c.getInactive()
	if err != nil {
		return false, err
	}
	return usage-inactive >= ((limit * c.memoryLimitPercent) / 100), nil
}

func (c cgroupsV1MemoryLimitChecker) getIntFromFile(fileName string) (int, error) {
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

func (c cgroupsV1MemoryLimitChecker) getInactive() (int, error) {
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
