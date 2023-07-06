// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package resourcemanager

import (
	"fmt"
	"os"
	"testing"
)

func updateFakeCgroupv1Files(c *cgroupsV1MemoryLimitChecker, limit, usage, inactive int) error {
	limitFile, err := os.Create(c.limitFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(limitFile, "%d\n", limit)
	if err != nil {
		return err
	}

	usageFile, err := os.Create(c.usageFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(usageFile, "%d\n", usage)
	if err != nil {
		return err
	}

	statsFile, err := os.Create(c.statsFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(statsFile, `total_cache 1029980160
total_rss 1016209408
total_inactive_file %d
total_active_file 321544192
`, inactive)
	if err != nil {
		return err
	}
	return nil
}

func TestCgroupsv1MemoryLimit(t *testing.T) {
	cgroupDir := t.TempDir()
	c := newCgroupsV1MemoryLimitChecker(cgroupDir, 95)
	_, err := c.isLimitExceeded()
	if err == nil {
		t.Error("Should fail open if can't read files")
	}

	err = updateFakeCgroupv1Files(c, 1000, 1000, 51)
	if err != nil {
		t.Error(err)
	}
	exceeded, err := c.isLimitExceeded()
	if err != nil {
		t.Error(err)
	}
	if exceeded {
		t.Error("Expected under limit")
	}

	err = updateFakeCgroupv1Files(c, 1000, 1000, 50)
	if err != nil {
		t.Error(err)
	}
	exceeded, err = c.isLimitExceeded()
	if err != nil {
		t.Error(err)
	}
	if !exceeded {
		t.Error("Expected over limit")
	}

}
