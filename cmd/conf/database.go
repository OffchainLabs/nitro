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
	if c.DBEngine == "pebble" {
		if err := c.Pebble.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type PebbleConfig struct {
	MaxConcurrentCompactions int                      `koanf:"max-concurrent-compactions"`
	Experimental             PebbleExperimentalConfig `koanf:"experimental"`
}

var PebbleConfigDefault = PebbleConfig{
	MaxConcurrentCompactions: runtime.NumCPU(),
	Experimental:             PebbleExperimentalConfigDefault,
}

func PebbleConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".max-concurrent-compactions", PebbleConfigDefault.MaxConcurrentCompactions, "maximum number of concurrent compactions")
	PebbleExperimentalConfigAddOptions(prefix+".experimental", f)
}

func (c *PebbleConfig) Validate() error {
	if c.MaxConcurrentCompactions < 1 {
		return fmt.Errorf("invalid .max-concurrent-compactions value: %d, has to be greater then 0", c.MaxConcurrentCompactions)
	}
	if err := c.Experimental.Validate(); err != nil {
		return err
	}
	return nil
}

type PebbleExperimentalConfig struct {
	BytesPerSync                int    `koanf:"bytes-per-sync"`
	L0CompactionFileThreshold   int    `koanf:"l0-compaction-file-threshold"`
	L0CompactionThreshold       int    `koanf:"l0-compaction-threshold"`
	L0StopWritesThreshold       int    `koanf:"l0-stop-writes-threshold"`
	LBaseMaxBytes               int64  `koanf:"l-base-max-bytes"`
	MemTableStopWritesThreshold int    `koanf:"mem-table-stop-writes-threshold"`
	DisableAutomaticCompactions bool   `koanf:"disable-automatic-compactions"`
	WALBytesPerSync             int    `koanf:"wal-bytes-per-sync"`
	WALDir                      string `koanf:"wal-dir"`
	WALMinSyncInterval          int    `koanf:"wal-min-sync-interval"`
	TargetByteDeletionRate      int    `koanf:"target-byte-deletion-rate"`

	// level specific
	BlockSize                 int   `koanf:"block-size"`
	IndexBlockSize            int   `koanf:"index-block-size"`
	TargetFileSize            int64 `koanf:"target-file-size"`
	TargetFileSizeEqualLevels bool  `koanf:"target-file-size-equal-levels"`

	// pebble experimental
	L0CompactionConcurrency   int    `koanf:"l0-compaction-concurrency"`
	CompactionDebtConcurrency uint64 `koanf:"compaction-debt-concurrency"`
	ReadCompactionRate        int64  `koanf:"read-compaction-rate"`
	ReadSamplingMultiplier    int64  `koanf:"read-sampling-multiplier"`
	MaxWriterConcurrency      int    `koanf:"max-writer-concurrency"`
	ForceWriterParallelism    bool   `koanf:"force-writer-parallelism"`
}

var PebbleExperimentalConfigDefault = PebbleExperimentalConfig{
	BytesPerSync:                512 << 10, // 512 KB
	L0CompactionFileThreshold:   500,
	L0CompactionThreshold:       4,
	L0StopWritesThreshold:       12,
	LBaseMaxBytes:               64 << 20, // 64 MB
	MemTableStopWritesThreshold: 2,
	DisableAutomaticCompactions: false,
	WALBytesPerSync:             0,  // no background syncing
	WALDir:                      "", // use same dir as for sstables
	WALMinSyncInterval:          0,  // no artificial delay
	TargetByteDeletionRate:      0,  // deletion pacing disabled

	BlockSize:                 4 << 10, // 4 KB
	IndexBlockSize:            4 << 10, // 4 KB
	TargetFileSize:            2 << 20, // 2 MB
	TargetFileSizeEqualLevels: true,

	L0CompactionConcurrency:   10,
	CompactionDebtConcurrency: 1 << 30, // 1GB
	ReadCompactionRate:        16000,   // see ReadSamplingMultiplier comment
	ReadSamplingMultiplier:    -1,      // geth default, disables read sampling and disables read triggered compaction
	MaxWriterConcurrency:      0,
	ForceWriterParallelism:    false,
}

func PebbleExperimentalConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".bytes-per-sync", PebbleExperimentalConfigDefault.BytesPerSync, "number of bytes to write to a SSTable before calling Sync on it in the background")
	f.Int(prefix+".l0-compaction-file-threshold", PebbleExperimentalConfigDefault.L0CompactionFileThreshold, "count of L0 files necessary to trigger an L0 compaction")
	f.Int(prefix+".l0-compaction-threshold", PebbleExperimentalConfigDefault.L0CompactionThreshold, "amount of L0 read-amplification necessary to trigger an L0 compaction")
	f.Int(prefix+".l0-stop-writes-threshold", PebbleExperimentalConfigDefault.L0StopWritesThreshold, "hard limit on L0 read-amplification, computed as the number of L0 sublevels. Writes are stopped when this threshold is reached")
	f.Int64(prefix+".l-base-max-bytes", PebbleExperimentalConfigDefault.LBaseMaxBytes, "The maximum number of bytes for LBase. The base level is the level which L0 is compacted into. The base level is determined dynamically based on the existing data in the LSM. The maximum number of bytes for other levels is computed dynamically based on the base level's maximum size. When the maximum number of bytes for a level is exceeded, compaction is requested.")
	f.Int(prefix+".mem-table-stop-writes-threshold", PebbleExperimentalConfigDefault.MemTableStopWritesThreshold, "hard limit on the number of queued of MemTables")
	f.Bool(prefix+".disable-automatic-compactions", PebbleExperimentalConfigDefault.DisableAutomaticCompactions, "disables automatic compactions")
	f.Int(prefix+".wal-bytes-per-sync", PebbleExperimentalConfigDefault.WALBytesPerSync, "number of bytes to write to a write-ahead log (WAL) before calling Sync on it in the background")
	f.String(prefix+".wal-dir", PebbleExperimentalConfigDefault.WALDir, "absolute path of directory to store write-ahead logs (WALs) in. If empty, WALs will be stored in the same directory as sstables")
	f.Int(prefix+".wal-min-sync-interval", PebbleExperimentalConfigDefault.WALMinSyncInterval, "minimum duration in microseconds between syncs of the WAL. If WAL syncs are requested faster than this interval, they will be artificially delayed.")
	f.Int(prefix+".target-byte-deletion-rate", PebbleExperimentalConfigDefault.TargetByteDeletionRate, "rate (in bytes per second) at which sstable file deletions are limited to (under normal circumstances).")
	f.Int(prefix+".block-size", PebbleExperimentalConfigDefault.BlockSize, "target uncompressed size in bytes of each table block")
	f.Int(prefix+".index-block-size", PebbleExperimentalConfigDefault.IndexBlockSize, fmt.Sprintf("target uncompressed size in bytes of each index block. When the index block size is larger than this target, two-level indexes are automatically enabled. Setting this option to a large value (such as %d) disables the automatic creation of two-level indexes.", math.MaxInt32))
	f.Int64(prefix+".target-file-size", PebbleExperimentalConfigDefault.TargetFileSize, "target file size for the level 0")
	f.Bool(prefix+".target-file-size-equal-levels", PebbleExperimentalConfigDefault.TargetFileSizeEqualLevels, "if true same target-file-size will be uses for all levels, otherwise target size for layer n = 2 * target size for layer n - 1")

	f.Int(prefix+".l0-compaction-concurrency", PebbleExperimentalConfigDefault.L0CompactionConcurrency, "threshold of L0 read-amplification at which compaction concurrency is enabled (if compaction-debt-concurrency was not already exceeded). Every multiple of this value enables another concurrent compaction up to max-concurrent-compactions.")
	f.Uint64(prefix+".compaction-debt-concurrency", PebbleExperimentalConfigDefault.CompactionDebtConcurrency, "controls the threshold of compaction debt at which additional compaction concurrency slots are added. For every multiple of this value in compaction debt bytes, an additional concurrent compaction is added. This works \"on top\" of l0-compaction-concurrency, so the higher of the count of compaction concurrency slots as determined by the two options is chosen.")
	f.Int64(prefix+".read-compaction-rate", PebbleExperimentalConfigDefault.ReadCompactionRate, "controls the frequency of read triggered compactions by adjusting `AllowedSeeks` in manifest.FileMetadata: AllowedSeeks = FileSize / ReadCompactionRate")
	f.Int64(prefix+".read-sampling-multiplier", PebbleExperimentalConfigDefault.ReadSamplingMultiplier, "a multiplier for the readSamplingPeriod in iterator.maybeSampleRead() to control the frequency of read sampling to trigger a read triggered compaction. A value of -1 prevents sampling and disables read triggered compactions. Geth default is -1. The pebble default is 1 << 4. which gets multiplied with a constant of 1 << 16 to yield 1 << 20 (1MB).")
	f.Int(prefix+".max-writer-concurrency", PebbleExperimentalConfigDefault.MaxWriterConcurrency, "maximum number of compression workers the compression queue is allowed to use. If max-writer-concurrency > 0, then the Writer will use parallelism, to compress and write blocks to disk. Otherwise, the writer will compress and write blocks to disk synchronously.")
	f.Bool(prefix+".force-writer-parallelism", PebbleExperimentalConfigDefault.ForceWriterParallelism, "force parallelism in the sstable Writer for the metamorphic tests. Even with the MaxWriterConcurrency option set, pebble only enables parallelism in the sstable Writer if there is enough CPU available, and this option bypasses that.")
}

func (c *PebbleExperimentalConfig) Validate() error {
	if !filepath.IsAbs(c.WALDir) {
		return fmt.Errorf("invalid .wal-dir directory (%s) - has to be an absolute path", c.WALDir)
	}
	// TODO
	return nil
}

func (c *PebbleConfig) ExtraOptions(namespace string) *pebble.ExtraOptions {
	var maxConcurrentCompactions func() int
	if c.MaxConcurrentCompactions > 0 {
		maxConcurrentCompactions = func() int { return c.MaxConcurrentCompactions }
	}
	var walMinSyncInterval func() time.Duration
	if c.Experimental.WALMinSyncInterval > 0 {
		walMinSyncInterval = func() time.Duration {
			return time.Microsecond * time.Duration(c.Experimental.WALMinSyncInterval)
		}
	}
	var levels []pebble.ExtraLevelOptions
	for i := 0; i < 7; i++ {
		targetFileSize := c.Experimental.TargetFileSize
		if !c.Experimental.TargetFileSizeEqualLevels {
			targetFileSize = targetFileSize << i
		}
		levels = append(levels, pebble.ExtraLevelOptions{
			BlockSize:      c.Experimental.BlockSize,
			IndexBlockSize: c.Experimental.IndexBlockSize,
			TargetFileSize: targetFileSize,
		})
	}
	walDir := c.Experimental.WALDir
	if walDir != "" {
		walDir = path.Join(walDir, namespace)
	}
	return &pebble.ExtraOptions{
		BytesPerSync:                c.Experimental.BytesPerSync,
		L0CompactionFileThreshold:   c.Experimental.L0CompactionFileThreshold,
		L0CompactionThreshold:       c.Experimental.L0CompactionThreshold,
		L0StopWritesThreshold:       c.Experimental.L0StopWritesThreshold,
		LBaseMaxBytes:               c.Experimental.LBaseMaxBytes,
		MemTableStopWritesThreshold: c.Experimental.MemTableStopWritesThreshold,
		MaxConcurrentCompactions:    maxConcurrentCompactions,
		DisableAutomaticCompactions: c.Experimental.DisableAutomaticCompactions,
		WALBytesPerSync:             c.Experimental.WALBytesPerSync,
		WALDir:                      walDir,
		WALMinSyncInterval:          walMinSyncInterval,
		TargetByteDeletionRate:      c.Experimental.TargetByteDeletionRate,
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
