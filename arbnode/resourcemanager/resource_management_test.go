// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package resourcemanager

import (
	"fmt"
	"os"
	"regexp"
	"testing"
)

func updateFakeCgroupFiles(c *cgroupsMemoryLimitChecker, limit, usage, inactive int) error {
	limitFile, err := os.Create(c.files.limitFile)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(limitFile, "%d\n", limit); err != nil {
		return err
	}

	usageFile, err := os.Create(c.files.usageFile)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(usageFile, "%d\n", usage); err != nil {
		return err
	}

	statsFile, err := os.Create(c.files.statsFile)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(statsFile, `total_cache 1029980160
total_rss 1016209408
total_inactive_file %d
total_active_file 321544192
`, inactive)
	return err
}

func makeCgroupsTestDir(cgroupDir string) cgroupsMemoryFiles {
	return cgroupsMemoryFiles{
		limitFile:  cgroupDir + "/memory.limit_in_bytes",
		usageFile:  cgroupDir + "/memory.usage_in_bytes",
		statsFile:  cgroupDir + "/memory.stat",
		inactiveRe: regexp.MustCompile(`total_inactive_file (\d+)`),
	}
}

func TestCgroupsFailIfCantOpen(t *testing.T) {
	testFiles := makeCgroupsTestDir(t.TempDir())
	c := newCgroupsMemoryLimitChecker(testFiles, 95)
	var err error
	if _, err = c.isLimitExceeded(); err == nil {
		t.Fatal("Should fail open if can't read files")
	}
}

func TestCgroupsMemoryLimit(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		inactive int
		want     bool
	}{
		{
			desc:     "limit should be exceeded",
			inactive: 50,
			want:     true,
		},
		{
			desc:     "limit should not be exceeded",
			inactive: 51,
			want:     false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			testFiles := makeCgroupsTestDir(t.TempDir())
			c := newCgroupsMemoryLimitChecker(testFiles, 95)
			if err := updateFakeCgroupFiles(c, 1000, 1000, tc.inactive); err != nil {
				t.Fatalf("Updating cgroup files: %v", err)
			}
			exceeded, err := c.isLimitExceeded()
			if err != nil {
				t.Fatalf("Checking if limit exceeded: %v", err)
			}
			if exceeded != tc.want {
				t.Errorf("isLimitExceeded() = %t, want %t", exceeded, tc.want)
			}
		},
		)
	}
}
