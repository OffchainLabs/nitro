// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package addressfilter

import (
	"fmt"
	"strings"
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
		{"no max size", true, 0, 0, 0},
		{"negative max size", true, -1, 0, 0},
		{"one mb", true, 1, 0, 1024 * 1024 / minBytesPerHashEntry},
		{"ten mb", true, 10, 0, 10 * 1024 * 1024 / minBytesPerHashEntry},
		{"custom min bytes per entry", true, 1, 66, 1024 * 1024 / 66},
		{"default min bytes per entry", true, 1, minBytesPerHashEntry, 1024 * 1024 / minBytesPerHashEntry},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := FileConfig{Config: s3syncer.Config{PreallocateMemory: tt.prealloc, MaxFileSizeMB: tt.maxMB}, MinBytesPerHashEntry: tt.minBytes}
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
		emptyConfig := Config{FilesList: "default"}
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
		require.Empty(t, cfg.resolvedFiles)
	})

	t.Run("static list plus files", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.StaticList = "{}"
		require.NoError(t, cfg.Validate())
		require.Len(t, cfg.resolvedFiles, 1)
	})

	t.Run("valid multiple files", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files = append(cfg.Files, validTestFileConfig(t, "hashlists/other.json"))
		require.NoError(t, cfg.Validate())
	})

	t.Run("zero fields normalized to defaults", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files[0].ChunkSizeMB = 0
		cfg.Files[0].MaxRetries = 0
		cfg.Files[0].Concurrency = 0
		cfg.Files[0].PollInterval = 0
		cfg.Files[0].MinBytesPerHashEntry = 0
		require.NoError(t, cfg.Validate())
		require.Equal(t, DefaultFileConfig.ChunkSizeMB, cfg.resolvedFiles[0].ChunkSizeMB)
		require.Equal(t, DefaultFileConfig.MaxRetries, cfg.resolvedFiles[0].MaxRetries)
		require.Equal(t, DefaultFileConfig.Concurrency, cfg.resolvedFiles[0].Concurrency)
		require.Equal(t, DefaultFileConfig.PollInterval, cfg.resolvedFiles[0].PollInterval)
		require.Equal(t, DefaultFileConfig.MinBytesPerHashEntry, cfg.resolvedFiles[0].MinBytesPerHashEntry)
		// Validate derives the effective config without mutating its inputs.
		require.Zero(t, cfg.Files[0].ChunkSizeMB)
		require.Zero(t, cfg.Files[0].PollInterval)
	})

	t.Run("negative min-bytes-per-hash-entry", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.Files[0].MinBytesPerHashEntry = -1
		err := cfg.Validate()
		require.ErrorContains(t, err, "min-bytes-per-hash-entry must be positive")
		require.ErrorContains(t, err, "files[0]")
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

func TestConfigValidateFilesList(t *testing.T) {
	t.Run("parses entries and preserves defaults", func(t *testing.T) {
		downloadDir := t.TempDir()
		cfg := DefaultConfig
		cfg.FilesList = fmt.Sprintf(
			`[{"bucket":"b1","object-key":"k1","region":"us-east-1","download-dir":%q},`+
				`{"bucket":"b2","object-key":"k2","region":"us-east-1","download-dir":%q,"poll-interval":%d,"chunk-size-mb":8,"min-bytes-per-hash-entry":66}]`,
			downloadDir, downloadDir, int64(time.Minute))
		require.NoError(t, cfg.Validate())
		require.Len(t, cfg.resolvedFiles, 2)

		// Omitted fields keep their defaults.
		require.Equal(t, DefaultFileConfig.PollInterval, cfg.resolvedFiles[0].PollInterval)
		require.Equal(t, DefaultFileConfig.ChunkSizeMB, cfg.resolvedFiles[0].ChunkSizeMB)
		require.Equal(t, DefaultFileConfig.MaxRetries, cfg.resolvedFiles[0].MaxRetries)
		require.Equal(t, DefaultFileConfig.Concurrency, cfg.resolvedFiles[0].Concurrency)
		require.Equal(t, DefaultFileConfig.MinBytesPerHashEntry, cfg.resolvedFiles[0].MinBytesPerHashEntry)
		require.True(t, cfg.resolvedFiles[0].PreallocateMemory)

		// Explicit fields override defaults.
		require.Equal(t, time.Minute, cfg.resolvedFiles[1].PollInterval)
		require.Equal(t, 8, cfg.resolvedFiles[1].ChunkSizeMB)
		require.Equal(t, 66, cfg.resolvedFiles[1].MinBytesPerHashEntry)
	})

	t.Run("conflicts with files slice", func(t *testing.T) {
		cfg := validTestConfig(t)
		cfg.FilesList = `[{"bucket":"b1"}]`
		require.ErrorContains(t, cfg.Validate(), "mutually exclusive")
	})

	t.Run("malformed json", func(t *testing.T) {
		cfg := DefaultConfig
		cfg.FilesList = `{"bucket":"b"}` // object, not array
		require.ErrorContains(t, cfg.Validate(), "failed to parse address-filter files-list")
	})

	t.Run("empty array", func(t *testing.T) {
		cfg := DefaultConfig
		cfg.FilesList = `[]`
		require.ErrorContains(t, cfg.Validate(), "at least one file")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		downloadDir := t.TempDir()
		cfg := DefaultConfig
		cfg.FilesList = fmt.Sprintf(
			`[{"bucket":"b1","object-key":"k1","region":"us-east-1","download-dir":%q,"max-file-sizemb":1}]`,
			downloadDir)
		err := cfg.Validate()
		require.ErrorContains(t, err, "files-list entry 0")
		require.ErrorContains(t, err, "max-file-sizemb")
	})

	t.Run("entry validation still applies", func(t *testing.T) {
		cfg := DefaultConfig
		cfg.FilesList = `[{"bucket":"b1","object-key":"k1","region":"us-east-1"}]` // missing download-dir
		err := cfg.Validate()
		require.ErrorContains(t, err, "files[0]")
		require.True(t, strings.Contains(err.Error(), "download-dir"))
	})
}
