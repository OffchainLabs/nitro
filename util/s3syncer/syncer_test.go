// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				ObjectKey: "path/to/file.json",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			config: Config{
				Region:    "us-east-1",
				ObjectKey: "path/to/file.json",
			},
			wantErr: true,
		},
		{
			name: "missing region",
			config: Config{
				Bucket:    "test-bucket",
				ObjectKey: "path/to/file.json",
			},
			wantErr: true,
		},
		{
			name: "missing object key",
			config: Config{
				Bucket: "test-bucket",
				Region: "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "valid config with credentials",
			config: Config{
				Bucket:    "test-bucket",
				Region:    "us-east-1",
				ObjectKey: "path/to/file.json",
				AccessKey: "AKIAIOSFODNN7EXAMPLE",
				SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
