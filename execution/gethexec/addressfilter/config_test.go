// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"testing"
	"time"

	"github.com/knadh/koanf"
	koanfjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"

	"github.com/offchainlabs/nitro/util/s3client"
	"github.com/offchainlabs/nitro/util/s3syncer"
)

func TestConfigNumPreallocatedHashes(t *testing.T) {
	cases := []struct {
		name     string
		prealloc bool
		maxMB    int
		minBytes int
		want     int
	}{
		{"disabled", false, 10, 0, 0},
		{"no max size", true, 0, minBytesPerHashEntry, 0},
		{"negative max size", true, -1, minBytesPerHashEntry, 0},
		{"custom min bytes per entry", true, 1, 66, 1024 * 1024 / 66},
		{"default min bytes per entry", true, 1, minBytesPerHashEntry, 1024 * 1024 / minBytesPerHashEntry},
		{"ten mb", true, 10, minBytesPerHashEntry, 10 * 1024 * 1024 / minBytesPerHashEntry},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := FileConfig{Config: s3syncer.Config{DisablePreallocateMemory: !tt.prealloc, MaxFileSizeMB: tt.maxMB}, MinBytesPerHashEntry: tt.minBytes}
			if got := cfg.numPreallocatedHashes(); got != tt.want {
				t.Errorf("numPreallocatedHashes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func validTestFileConfig(t *testing.T, objectKey string) FileConfig {
	t.Helper()
	file := DefaultFileConfig
	file.Config.Config = s3client.Config{Region: "us-east-1"}
	file.Bucket = "test-bucket"
	file.ObjectKey = objectKey
	file.DownloadDir = t.TempDir()
	return file
}

func validTestConfig(t *testing.T) Config {
	t.Helper()
	cfg := DefaultConfig
	cfg.Files = []FileConfig{validTestFileConfig(t, "hashlists/current.json")}
	return cfg
}

func TestConfigValidate(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		emptyConfig := Config{}
		require.ErrorContains(t, emptyConfig.Validate(), "at least one file")
	})

	t.Run("valid single file", func(t *testing.T) {
		cfg := validTestConfig(t)
		require.NoError(t, cfg.Validate())
	})

	t.Run("static list alone satisfies file requirement", func(t *testing.T) {
		cfg := DefaultConfig
		cfg.StaticList = "{}"
		require.NoError(t, cfg.Validate())
	})

	t.Run("static list plus files", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.StaticList = "{}"
		require.NoError(t, cfg.Validate())
	})

	t.Run("valid multiple files", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files = append(cfg.Files, validTestFileConfig(t, "hashlists/other.json"))
		require.NoError(t, cfg.Validate())
	})

	t.Run("zero fields accepted via defaults", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files[0].ChunkSizeMB = 0
		cfg.Files[0].MaxRetries = 0
		cfg.Files[0].Concurrency = 0
		cfg.Files[0].PollInterval = 0
		cfg.Files[0].MinBytesPerHashEntry = 0
		require.NoError(t, cfg.Validate())
		// Validate checks defaulted copies without mutating its inputs.
		require.Zero(t, cfg.Files[0].ChunkSizeMB)
		require.Zero(t, cfg.Files[0].PollInterval)
	})

	t.Run("valid min-bytes-per-hash-entry values", func(t *testing.T) {
		for _, v := range []int{minBytesPerHashEntry, minBytesPerSha256HashEntry} {
			cfg := validTestConfig(t)
			cfg.Files[0].MinBytesPerHashEntry = v
			require.NoError(t, cfg.Validate())
		}
	})

	t.Run("invalid min-bytes-per-hash-entry", func(t *testing.T) {
		for _, v := range []int{-1, 1, 50, 67} {
			cfg := validTestConfig(t)
			cfg.Files[0].MinBytesPerHashEntry = v
			err := cfg.Validate()
			require.ErrorContains(t, err, "min-bytes-per-hash-entry must be 42 (any scheme) or 66 (sha256-only lists)")
			require.ErrorContains(t, err, "files[0]")
		}
	})

	t.Run("negative poll interval", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files[0].PollInterval = -time.Second
		err := cfg.Validate()
		require.ErrorContains(t, err, "poll-interval must be positive")
		require.ErrorContains(t, err, "files[0]")
	})

	t.Run("per-file s3 validation error carries index", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files = append(cfg.Files, validTestFileConfig(t, "hashlists/other.json"))
		cfg.Files[1].Bucket = ""
		err := cfg.Validate()
		require.ErrorContains(t, err, "files[1]")
		require.ErrorContains(t, err, "bucket is required")
	})

	t.Run("duplicate bucket and object key", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files = append(cfg.Files, validTestFileConfig(t, cfg.Files[0].ObjectKey))
		require.ErrorContains(t, cfg.Validate(), "duplicate entry")
	})

	t.Run("zero cache size", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.CacheSize = 0
		require.ErrorContains(t, cfg.Validate(), "cache-size must be positive")
	})

	t.Run("negative cache size", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.CacheSize = -1
		require.ErrorContains(t, cfg.Validate(), "cache-size must be positive")
	})
}

// TestConfigKoanfFilesDecoding pins down that koanf decodes the files slice
// from a config file, including the two-level koanf ",squash" nesting
// (FileConfig → s3syncer.Config → s3client.Config) inside slice elements.
func TestConfigKoanfFilesDecoding(t *testing.T) {
	configJSON := `{
		"files": [
			{
				"bucket": "b1",
				"object-key": "k1",
				"region": "us-east-1",
				"access-key": "ak",
				"secret-key": "sk",
				"download-dir": "/data/tmp",
				"poll-interval": "1m",
				"chunk-size-mb": 8
			},
			{
				"bucket": "b2",
				"object-key": "k2",
				"region": "us-east-1",
				"download-dir": "/data/tmp",
				"poll-interval": "10m"
			}
		],
		"cache-size": 100
	}`

	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider([]byte(configJSON)), koanfjson.Parser()))

	var cfg Config
	decoderConfig := mapstructure.DecoderConfig{
		ErrorUnused:      true,
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		Result:           &cfg,
		WeaklyTypedInput: true,
	}
	require.NoError(t, k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{DecoderConfig: &decoderConfig}))

	require.Len(t, cfg.Files, 2)
	require.Equal(t, "b1", cfg.Files[0].Bucket)
	require.Equal(t, "k1", cfg.Files[0].ObjectKey)
	require.Equal(t, "us-east-1", cfg.Files[0].Region)
	require.Equal(t, "ak", cfg.Files[0].AccessKey)
	require.Equal(t, "sk", cfg.Files[0].SecretKey)
	require.Equal(t, "/data/tmp", cfg.Files[0].DownloadDir)
	require.Equal(t, time.Minute, cfg.Files[0].PollInterval)
	require.Equal(t, 8, cfg.Files[0].ChunkSizeMB)
	require.Equal(t, "b2", cfg.Files[1].Bucket)
	require.Equal(t, 10*time.Minute, cfg.Files[1].PollInterval)
	require.Equal(t, 100, cfg.CacheSize)
}

func TestFileConfigWithDefaults(t *testing.T) {
	t.Run("zero fields backfilled", func(t *testing.T) {
		var file FileConfig
		got := file.withDefaults()
		require.Equal(t, DefaultFileConfig.ChunkSizeMB, got.ChunkSizeMB)
		require.Equal(t, DefaultFileConfig.MaxRetries, got.MaxRetries)
		require.Equal(t, DefaultFileConfig.Concurrency, got.Concurrency)
		require.Equal(t, DefaultFileConfig.PollInterval, got.PollInterval)
		require.Equal(t, DefaultFileConfig.MinBytesPerHashEntry, got.MinBytesPerHashEntry)
		// The receiver is left untouched.
		require.Zero(t, file.ChunkSizeMB)
		require.Zero(t, file.PollInterval)
	})

	t.Run("explicit fields preserved", func(t *testing.T) {
		file := FileConfig{
			Config:               s3syncer.Config{ChunkSizeMB: 8, MaxRetries: 7, Concurrency: 2},
			PollInterval:         time.Minute,
			MinBytesPerHashEntry: 66,
		}
		got := file.withDefaults()
		require.Equal(t, 8, got.ChunkSizeMB)
		require.Equal(t, 7, got.MaxRetries)
		require.Equal(t, 2, got.Concurrency)
		require.Equal(t, time.Minute, got.PollInterval)
		require.Equal(t, 66, got.MinBytesPerHashEntry)
	})
}
