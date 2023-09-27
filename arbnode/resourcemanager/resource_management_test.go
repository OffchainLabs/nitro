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
	c := newCgroupsMemoryLimitChecker(testFiles, 1024*1024*512)
	if _, err := c.isLimitExceeded(); err == nil {
		t.Fatal("Should fail open if can't read files")
	}
}

func TestCgroupsMemoryLimit(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		sysLimit int
		inactive int
		usage    int
		memLimit string
		want     bool
	}{
		{
			desc:     "limit should be exceeded",
			sysLimit: 1000,
			inactive: 50,
			usage:    1000,
			memLimit: "50B",
			want:     true,
		},
		{
			desc:     "limit should not be exceeded",
			sysLimit: 1000,
			inactive: 51,
			usage:    1000,
			memLimit: "50b",
			want:     false,
		},
		{
			desc:     "limit (MB) should be exceeded",
			sysLimit: 1000 * 1024 * 1024,
			inactive: 50 * 1024 * 1024,
			usage:    1000 * 1024 * 1024,
			memLimit: "50MB",
			want:     true,
		},
		{
			desc:     "limit (MB) should not be exceeded",
			sysLimit: 1000 * 1024 * 1024,
			inactive: 1 + 50*1024*1024,
			usage:    1000 * 1024 * 1024,
			memLimit: "50m",
			want:     false,
		},
		{
			desc:     "limit (GB) should be exceeded",
			sysLimit: 1000 * 1024 * 1024 * 1024,
			inactive: 50 * 1024 * 1024 * 1024,
			usage:    1000 * 1024 * 1024 * 1024,
			memLimit: "50G",
			want:     true,
		},
		{
			desc:     "limit (GB) should not be exceeded",
			sysLimit: 1000 * 1024 * 1024 * 1024,
			inactive: 1 + 50*1024*1024*1024,
			usage:    1000 * 1024 * 1024 * 1024,
			memLimit: "50gb",
			want:     false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			testFiles := makeCgroupsTestDir(t.TempDir())
			memLimit, err := parseMemLimit(tc.memLimit)
			if err != nil {
				t.Fatalf("Parsing memory limit failed: %v", err)
			}
			c := newCgroupsMemoryLimitChecker(testFiles, memLimit)
			if err := updateFakeCgroupFiles(c, tc.sysLimit, tc.usage, tc.inactive); err != nil {
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
