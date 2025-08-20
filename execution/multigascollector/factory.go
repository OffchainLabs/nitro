// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package multigascollector

// NewCollectorFactory returns a CollectorFactory that instantiates either
// a real collector (writing to disk) or a no-op collector depending on config.
func NewCollectorFactory() CollectorFactory {
	return func(cfg CollectorConfig) (Collector, error) {
		// If OutputDir is not set, return nil collector
		if cfg.OutputDir == "" {
			return nil, nil
		}

		// Otherwise use the full implementation
		c, err := NewFileCollector(cfg) // this is your current "NewCollector" renamed
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}
