// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbnode

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/andybalholm/brotli"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"

	"github.com/offchainlabs/nitro/util/arbmath"
)

// CompressionLevelStep defines compression levels to use at a given backlog threshold.
type CompressionLevelStep struct {
	Backlog            uint64 `koanf:"backlog" json:"backlog"`
	Level              int    `koanf:"level" json:"level"`
	RecompressionLevel int    `koanf:"recompression-level" json:"recompression-level"`
}

// CompressionLevelStepList is a list of compression level steps for configuring
// adaptive compression based on batch backlog.
type CompressionLevelStepList []CompressionLevelStep

func (l *CompressionLevelStepList) Set(jsonStr string) error {
	return l.UnmarshalJSON([]byte(jsonStr))
}

func (l *CompressionLevelStepList) String() string {
	b, _ := json.Marshal(l)
	return string(b)
}

func (l *CompressionLevelStepList) UnmarshalJSON(data []byte) error {
	var tmp []CompressionLevelStep
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*l = tmp
	return nil
}

func (_ *CompressionLevelStepList) Type() string {
	return "CompressionLevelStepList"
}

// Validate checks that the compression level steps are valid:
// - Must have at least one entry
// - First entry must have backlog: 0
// - Backlog thresholds must be strictly ascending
// - Level and RecompressionLevel must be weakly descending (non-increasing)
// - RecompressionLevel must be >= Level within each entry
// - All levels must be in valid range: 0-11
func (l CompressionLevelStepList) Validate() error {
	if len(l) == 0 {
		return errors.New("compression-levels must have at least one entry")
	}
	if l[0].Backlog != 0 {
		return errors.New("first compression-levels entry must have backlog: 0")
	}
	for i, step := range l {
		if step.Level < 0 || step.Level > 11 {
			return fmt.Errorf("compression-levels[%d].level must be 0-11, got %d", i, step.Level)
		}
		if step.RecompressionLevel < 0 || step.RecompressionLevel > 11 {
			return fmt.Errorf("compression-levels[%d].recompression-level must be 0-11, got %d", i, step.RecompressionLevel)
		}
		if step.RecompressionLevel < step.Level {
			return fmt.Errorf("compression-levels[%d].recompression-level (%d) must be >= level (%d)", i, step.RecompressionLevel, step.Level)
		}
		if i > 0 {
			if step.Backlog <= l[i-1].Backlog {
				return fmt.Errorf("compression-levels[%d].backlog must be > compression-levels[%d].backlog", i, i-1)
			}
			if step.Level > l[i-1].Level {
				return fmt.Errorf("compression-levels[%d].level must be <= compression-levels[%d].level (weakly descending)", i, i-1)
			}
			if step.RecompressionLevel > l[i-1].RecompressionLevel {
				return fmt.Errorf("compression-levels[%d].recompression-level must be <= compression-levels[%d].recompression-level (weakly descending)", i, i-1)
			}
		}
	}
	return nil
}

var parsedCompressionLevelsConf CompressionLevelStepList

// FixCompressionLevelsCLIParsing decode compression-levels json CLI ARG
func FixCompressionLevelsCLIParsing(path string, k *koanf.Koanf) error {
	raw := k.Get(path)
	if jsonStr, ok := raw.(string); ok {
		if err := parsedCompressionLevelsConf.Set(jsonStr); err != nil {

			return err
		}
		tempMap := map[string]interface{}{path: parsedCompressionLevelsConf}
		if err := k.Load(confmap.Provider(tempMap, "."), nil); err != nil {
			return err
		}
	}
	return fmt.Errorf("CompressionLevels config not found in %s", path)
}

// DefaultCompressionLevels replicates the previous hardcoded adaptive compression behavior:
var DefaultCompressionLevels = CompressionLevelStepList{
	{Backlog: 0, Level: brotli.BestCompression, RecompressionLevel: brotli.BestCompression},
	{Backlog: 21, Level: brotli.DefaultCompression, RecompressionLevel: brotli.BestCompression},
	{Backlog: 41, Level: brotli.DefaultCompression, RecompressionLevel: brotli.DefaultCompression},
	{Backlog: 61, Level: 4, RecompressionLevel: brotli.DefaultCompression},
}

// ResolveCompressionLevels resolves the compression configuration from deprecated and new fields.
// Returns error if both are set. Converts deprecated format to new format if needed.
func ResolveCompressionLevels(compressionLevel int, compressionLevels CompressionLevelStepList) (CompressionLevelStepList, error) {
	// Check for conflict: both deprecated and new config set
	if compressionLevel > 0 && len(compressionLevels) > 0 {
		return nil, errors.New("cannot specify both compression-level (deprecated) and compression-levels; use only compression-levels")
	}

	// Return DefaultCompressionLevels if both (compressionLevel and compressionLevels ) are not set
	if len(compressionLevels) == 0 && compressionLevel == 0 {
		return DefaultCompressionLevels, nil
	}

	// If new config is set, validate and return it
	if len(compressionLevels) > 0 {
		if err := compressionLevels.Validate(); err != nil {
			return nil, fmt.Errorf("invalid compression-levels: %w", err)
		}
		return compressionLevels, nil
	}

	// Convert deprecated `compressionLevel` config to new format of compressionLevels
	resolved := CompressionLevelStepList{
		{Backlog: 0, Level: compressionLevel, RecompressionLevel: compressionLevel},
		{Backlog: 21, Level: arbmath.MinInt(compressionLevel, brotli.DefaultCompression), RecompressionLevel: compressionLevel},
		{Backlog: 41, Level: arbmath.MinInt(compressionLevel, brotli.DefaultCompression), RecompressionLevel: arbmath.MinInt(compressionLevel, brotli.DefaultCompression)},
		{Backlog: 61, Level: arbmath.MinInt(compressionLevel, 4), RecompressionLevel: arbmath.MinInt(compressionLevel, brotli.DefaultCompression)},
	}

	if err := resolved.Validate(); err != nil {
		return nil, fmt.Errorf("invalid compression-levels derived from compression-level: %w", err)
	}
	return resolved, nil
}
