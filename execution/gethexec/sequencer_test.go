// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package gethexec

import (
	"testing"
	"time"
)

func TestSequencerConfigValidatePGA(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*SequencerConfig)
		wantErr bool
	}{
		{"default config", func(c *SequencerConfig) {}, false},
		{"pga enabled", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
		}, false},
		{"timeboost enabled", func(c *SequencerConfig) {
			c.Timeboost.Enable = true
		}, false},
		{"pga and timeboost enabled", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.Timeboost.Enable = true
		}, true},
		{"zero value pga config", func(c *SequencerConfig) {
			c.ExperimentalPGA = PGAConfig{}
		}, true},
		{"pga enabled with zero rounds per block", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.ExperimentalPGA.RoundsPerBlock = 0
		}, true},
		{"pga enabled with one round per block", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.ExperimentalPGA.RoundsPerBlock = 1
		}, false},
		{"round length below minimum", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.MaxBlockSpeed = 250 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 6
		}, true},
		{"round length at minimum", func(c *SequencerConfig) {
			c.ExperimentalPGA.Enable = true
			c.MaxBlockSpeed = 250 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 5
		}, false},
		{"fast blocks with pga disabled", func(c *SequencerConfig) {
			c.MaxBlockSpeed = 10 * time.Millisecond
			c.ExperimentalPGA.RoundsPerBlock = 1
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultSequencerConfig
			tt.modify(&c)
			err := c.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestPGARoundLength(t *testing.T) {
	c := DefaultSequencerConfig
	c.MaxBlockSpeed = 250 * time.Millisecond
	c.ExperimentalPGA.RoundsPerBlock = 2
	if got := c.PGARoundLength(); got != 125*time.Millisecond {
		t.Errorf("expected round length 125ms, got %v", got)
	}
}
