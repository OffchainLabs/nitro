// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package resourcemanager

import (
	"fmt"
	"os"
	"testing"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func updateFakeCgroupv1Files(t *testing.T, c *cgroupsV1MemoryLimitChecker, limit, usage, inactive int) {
	limitFile, err := os.Create(c.limitFile)
	Require(t, err)
	_, err = fmt.Fprintf(limitFile, "%d\n", limit)
	Require(t, err)

	usageFile, err := os.Create(c.usageFile)
	Require(t, err)
	_, err = fmt.Fprintf(usageFile, "%d\n", usage)
	Require(t, err)

	statsFile, err := os.Create(c.statsFile)
	Require(t, err)
	_, err = fmt.Fprintf(statsFile, `total_cache 1029980160
total_rss 1016209408
total_inactive_file %d
total_active_file 321544192
`, inactive)
	Require(t, err)
}

func TestCgroupsv1MemoryLimit(t *testing.T) {
	cgroupDir := t.TempDir()
	c := newCgroupsV1MemoryLimitChecker(cgroupDir, 95)
	_, err := c.isLimitExceeded()
	if err == nil {
		Fail(t, "Should fail open if can't read files")
	}

	updateFakeCgroupv1Files(t, c, 1000, 1000, 51)
	exceeded, err := c.isLimitExceeded()
	Require(t, err)
	if exceeded {
		Fail(t, "Expected under limit")
	}

	updateFakeCgroupv1Files(t, c, 1000, 1000, 50)
	exceeded, err = c.isLimitExceeded()
	Require(t, err)
	if !exceeded {
		Fail(t, "Expected over limit")
	}

}

func Require(t *testing.T, err error, printables ...interface{}) {
	t.Helper()
	testhelpers.RequireImpl(t, err, printables...)
}

func Fail(t *testing.T, printables ...interface{}) {
	t.Helper()
	testhelpers.FailImpl(t, printables...)
}
