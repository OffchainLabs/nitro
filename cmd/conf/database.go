// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package conf

import (
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/ethdb/pebble"
	flag "github.com/spf13/pflag"
)

type PersistentConfig struct {
	GlobalConfig string       `koanf:"global-config"`
	Chain        string       `koanf:"chain"`
	LogDir       string       `koanf:"log-dir"`
	Handles      int          `koanf:"handles"`
	Ancient      string       `koanf:"ancient"`
	DBEngine     string       `koanf:"db-engine"`
	Pebble       PebbleConfig `koanf:"pebble"`
}

var PersistentConfigDefault = PersistentConfig{
	GlobalConfig: ".arbitrum",
	Chain:        "",
	LogDir:       "",
	Handles:      512,
	Ancient:      "",
	DBEngine:     "leveldb",
	Pebble:       PebbleConfigDefault,
}

func PersistentConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.String(prefix+".global-config", PersistentConfigDefault.GlobalConfig, "directory to store global config")
	f.String(prefix+".chain", PersistentConfigDefault.Chain, "directory to store chain state")
	f.String(prefix+".log-dir", PersistentConfigDefault.LogDir, "directory to store log file")
	f.Int(prefix+".handles", PersistentConfigDefault.Handles, "number of file descriptor handles to use for the database")
	f.String(prefix+".ancient", PersistentConfigDefault.Ancient, "directory of ancient where the chain freezer can be opened")
	f.String(prefix+".db-engine", PersistentConfigDefault.DBEngine, "backing database implementation to use ('leveldb' or 'pebble')")
	PebbleConfigAddOptions(prefix+".pebble", f)
}

func (c *PersistentConfig) ResolveDirectoryNames() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to read users home directory: %w", err)
	}

	// Make persistent storage directory relative to home directory if not already absolute
	if !filepath.IsAbs(c.GlobalConfig) {
		c.GlobalConfig = path.Join(homeDir, c.GlobalConfig)
	}
	err = os.MkdirAll(c.GlobalConfig, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create global configuration directory: %w", err)
	}

	// Make chain directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.Chain) {
		c.Chain = path.Join(c.GlobalConfig, c.Chain)
	}
	err = os.MkdirAll(c.Chain, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create chain directory: %w", err)
	}
	if DatabaseInDirectory(c.Chain) {
		return fmt.Errorf("database in --persistent.chain (%s) directory, try specifying parent directory", c.Chain)
	}

	// Make Log directory relative to persistent storage directory if not already absolute
	if !filepath.IsAbs(c.LogDir) {
		c.LogDir = path.Join(c.Chain, c.LogDir)
	}
	if c.LogDir != c.Chain {
		err = os.MkdirAll(c.LogDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to create Log directory: %w", err)
		}
		if DatabaseInDirectory(c.LogDir) {
			return fmt.Errorf("database in --persistent.log-dir (%s) directory, try specifying parent directory", c.LogDir)
		}
	}
	return nil
}

func DatabaseInDirectory(path string) bool {
	// Consider database present if file `CURRENT` in directory
	_, err := os.Stat(path + "/CURRENT")

	return err == nil
}

func (c *PersistentConfig) Validate() error {
	// we are validating .db-engine here to avoid unintended behaviour as empty string value also has meaning in geth's node.Config.DBEngine
	if c.DBEngine != "leveldb" && c.DBEngine != "pebble" {
		return fmt.Errorf(`invalid .db-engine choice: %q, allowed "leveldb" or "pebble"`, c.DBEngine)
	}
	return nil
}

type PebbleConfig struct {
	BytesPerSync                int                      `koanf:"bytes-per-sync"`
	L0CompactionFileThreshold   int                      `koanf:"l0-compaction-file-threshold"`
	L0CompactionThreshold       int                      `koanf:"l0-compaction-threshold"`
	L0StopWritesThreshold       int                      `koanf:"l0-stop-writes-threshold"`
	LBaseMaxBytes               int64                    `koanf:"l-base-max-bytes"`
	MaxConcurrentCompactions    int                      `koanf:"max-concurrent-compactions"`
	DisableAutomaticCompactions bool                     `koanf:"disable-automatic-compactions"`
	WALBytesPerSync             int                      `koanf:"wal-bytes-per-sync"`
	WALDir                      string                   `koanf:"wal-dir"`
	WALMinSyncInterval          int                      `koanf:"wal-min-sync-interval"`
	TargetByteDeletionRate      int                      `koanf:"target-byte-deletion-rate"`
	Experimental                PebbleExperimentalConfig `koanf:"experimental"`

	// level specific
	BlockSize                 int   `koanf:"block-size"`
	IndexBlockSize            int   `koanf:"index-block-size"`
	TargetFileSize            int64 `koanf:"target-file-size"`
	TargetFileSizeEqualLevels bool  `koanf:"target-file-size-equal-levels"`
}

var PebbleConfigDefault = PebbleConfig{
	BytesPerSync:                0, // pebble default will be used
	L0CompactionFileThreshold:   0, // pebble default will be used
	L0CompactionThreshold:       0, // pebble default will be used
	L0StopWritesThreshold:       0, // pebble default will be used
	LBaseMaxBytes:               0, // pebble default will be used
	MaxConcurrentCompactions:    runtime.NumCPU(),
	DisableAutomaticCompactions: false,
	WALBytesPerSync:             0,  // pebble default will be used
	WALDir:                      "", // default will use same dir as for sstables
	WALMinSyncInterval:          0,  // pebble default will be used
	TargetByteDeletionRate:      0,  // pebble default will be used
	Experimental:                PebbleExperimentalConfigDefault,
	BlockSize:                   4096,
	IndexBlockSize:              4096,
	TargetFileSize:              2 * 1024 * 1024,
	TargetFileSizeEqualLevels:   true,
}

func PebbleConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".bytes-per-sync", PebbleConfigDefault.BytesPerSync, "number of bytes to write to a SSTable before calling Sync on it in the background (0 = pebble default)")
	f.Int(prefix+".l0-compaction-file-threshold", PebbleConfigDefault.L0CompactionFileThreshold, "count of L0 files necessary to trigger an L0 compaction (0 = pebble default)")
	f.Int(prefix+".l0-compaction-threshold", PebbleConfigDefault.L0CompactionThreshold, "amount of L0 read-amplification necessary to trigger an L0 compaction (0 = pebble default)")
	f.Int(prefix+".l0-stop-writes-threshold", PebbleConfigDefault.L0StopWritesThreshold, "hard limit on L0 read-amplification, computed as the number of L0 sublevels. Writes are stopped when this threshold is reached (0 = pebble default)")
	f.Int64(prefix+".l-base-max-bytes", PebbleConfigDefault.LBaseMaxBytes, "hard limit on L0 read-amplification, computed as the number of L0 sublevels. Writes are stopped when this threshold is reached (0 = pebble default)")
	f.Int(prefix+".max-concurrent-compactions", PebbleConfigDefault.MaxConcurrentCompactions, "maximum number of concurrent compactions (0 = pebble default)")
	f.Bool(prefix+".disable-automatic-compactions", PebbleConfigDefault.DisableAutomaticCompactions, "disables automatic compactions")
	f.Int(prefix+".wal-bytes-per-sync", PebbleConfigDefault.WALBytesPerSync, "number of bytes to write to a write-ahead log (WAL) before calling Sync on it in the backgroud (0 = pebble default)")
	f.String(prefix+".wal-dir", PebbleConfigDefault.WALDir, "directory to store write-ahead logs (WALs) in. If empty, WALs will be stored in the same directory as sstables")
	f.Int(prefix+".wal-min-sync-interval", PebbleConfigDefault.WALMinSyncInterval, "minimum duration in microseconds between syncs of the WAL. If WAL syncs are requested faster than this interval, they will be artificially delayed.")
	f.Int(prefix+".target-byte-deletion-rate", PebbleConfigDefault.TargetByteDeletionRate, "rate (in bytes per second) at which sstable file deletions are limited to (under normal circumstances).")
	f.Int(prefix+".block-size", PebbleConfigDefault.BlockSize, "target uncompressed size in bytes of each table block")
	f.Int(prefix+".index-block-size", PebbleConfigDefault.IndexBlockSize, fmt.Sprintf("target uncompressed size in bytes of each index block. When the index block size is larger than this target, two-level indexes are automatically enabled. Setting this option to a large value (such as %d) disables the automatic creation of two-level indexes.", math.MaxInt32))
	PebbleExperimentalConfigAddOptions(prefix+".experimental", f)
	f.Int64(prefix+".target-file-size", PebbleConfigDefault.TargetFileSize, "target file size for the level 0")
	f.Bool(prefix+".target-file-size-equal-levels", PebbleConfigDefault.TargetFileSizeEqualLevels, "if true same target-file-size will be uses for all levels, otherwise target size for layer n = 2 * target size for layer n - 1")
}

type PebbleExperimentalConfig struct {
	L0CompactionConcurrency   int    `koanf:"l0-compaction-concurrency"`
	CompactionDebtConcurrency uint64 `koanf:"compaction-debt-concurrency"`
	ReadCompactionRate        int64  `koanf:"read-compaction-rate"`
	ReadSamplingMultiplier    int64  `koanf:"read-sampling-multiplier"`
	MaxWriterConcurrency      int    `koanf:"max-writer-concurrency"`
	ForceWriterParallelism    bool   `koanf:"force-writer-parallelism"`
}

var PebbleExperimentalConfigDefault = PebbleExperimentalConfig{
	L0CompactionConcurrency:   0,
	CompactionDebtConcurrency: 0,
	ReadCompactionRate:        0,
	ReadSamplingMultiplier:    -1,
	MaxWriterConcurrency:      0,
	ForceWriterParallelism:    false,
}

func PebbleExperimentalConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".l0-compaction-concurrency", PebbleExperimentalConfigDefault.L0CompactionConcurrency, "threshold of L0 read-amplification at which compaction concurrency is enabled (if compaction-debt-concurrency was not already exceeded). Every multiple of this value enables another concurrent compaction up to max-concurrent-compactions. (0 = pebble default)")
	f.Uint64(prefix+".compaction-debt-concurrency", PebbleExperimentalConfigDefault.CompactionDebtConcurrency, "controls the threshold of compaction debt at which additional compaction concurrency slots are added. For every multiple of this value in compaction debt bytes, an additional concurrent compaction is added. This works \"on top\" of l0-compaction-concurrency, so the higher of the count of compaction concurrency slots as determined by the two options is chosen. (0 = pebble default)")
	f.Int64(prefix+".read-compaction-rate", PebbleExperimentalConfigDefault.ReadCompactionRate, "controls the frequency of read triggered compactions by adjusting `AllowedSeeks` in manifest.FileMetadata: AllowedSeeks = FileSize / ReadCompactionRate")
	f.Int64(prefix+".read-sampling-multiplier", PebbleExperimentalConfigDefault.ReadSamplingMultiplier, "a multiplier for the readSamplingPeriod in iterator.maybeSampleRead() to control the frequency of read sampling to trigger a read triggered compaction. A value of -1 prevents sampling and disables read triggered compactions. Geth default is -1. The pebble default is 1 << 4. which gets multiplied with a constant of 1 << 16 to yield 1 << 20 (1MB). (0 = pebble default)")
	f.Int(prefix+".max-writer-concurrency", PebbleExperimentalConfigDefault.MaxWriterConcurrency, "maximum number of compression workers the compression queue is allowed to use. If max-writer-concurrency > 0, then the Writer will use parallelism, to compress and write blocks to disk. Otherwise, the writer will compress and write blocks to disk synchronously.")
	f.Bool(prefix+".force-writer-parallelism", PebbleExperimentalConfigDefault.ForceWriterParallelism, "force parallelism in the sstable Writer for the metamorphic tests. Even with the MaxWriterConcurrency option set, pebble only enables parallelism in the sstable Writer if there is enough CPU available, and this option bypasses that.")
}

func (c *PebbleConfig) ExtraOptions() *pebble.ExtraOptions {
	var maxConcurrentCompactions func() int
	if c.MaxConcurrentCompactions > 0 {
		maxConcurrentCompactions = func() int { return c.MaxConcurrentCompactions }
	}
	var walMinSyncInterval func() time.Duration
	if c.WALMinSyncInterval > 0 {
		walMinSyncInterval = func() time.Duration {
			return time.Microsecond * time.Duration(c.WALMinSyncInterval)
		}
	}
	var levels []pebble.ExtraLevelOptions
	for i := 0; i < 7; i++ {
		targetFileSize := c.TargetFileSize
		if !c.TargetFileSizeEqualLevels {
			targetFileSize = targetFileSize << i
		}
		levels = append(levels, pebble.ExtraLevelOptions{
			BlockSize:      c.BlockSize,
			IndexBlockSize: c.IndexBlockSize,
			TargetFileSize: targetFileSize,
		})
	}
	return &pebble.ExtraOptions{
		BytesPerSync:                c.BytesPerSync,
		L0CompactionFileThreshold:   c.L0CompactionFileThreshold,
		L0CompactionThreshold:       c.L0CompactionThreshold,
		L0StopWritesThreshold:       c.L0StopWritesThreshold,
		LBaseMaxBytes:               c.LBaseMaxBytes,
		MaxConcurrentCompactions:    maxConcurrentCompactions,
		DisableAutomaticCompactions: c.DisableAutomaticCompactions,
		WALBytesPerSync:             c.WALBytesPerSync,
		WALDir:                      c.WALDir,
		WALMinSyncInterval:          walMinSyncInterval,
		TargetByteDeletionRate:      c.TargetByteDeletionRate,
		Experimental: pebble.ExtraOptionsExperimental{
			L0CompactionConcurrency:   c.Experimental.L0CompactionConcurrency,
			CompactionDebtConcurrency: c.Experimental.CompactionDebtConcurrency,
			ReadCompactionRate:        c.Experimental.ReadCompactionRate,
			ReadSamplingMultiplier:    c.Experimental.ReadSamplingMultiplier,
			MaxWriterConcurrency:      c.Experimental.MaxWriterConcurrency,
			ForceWriterParallelism:    c.Experimental.ForceWriterParallelism,
		},
		Levels: levels,
	}
}

func (c *PebbleConfig) Validate() error {
	// TODO
	return nil
}
