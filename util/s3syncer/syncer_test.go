// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package s3syncer

import (
	"testing"

	"github.com/offchainlabs/nitro/util/s3client"
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
				Config:    s3client.Config{Region: "us-east-1"},
				Bucket:    "test-bucket",
				ObjectKey: "path/to/file.json",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			config: Config{
				Config:    s3client.Config{Region: "us-east-1"},
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
				Config: s3client.Config{Region: "us-east-1"},
				Bucket: "test-bucket",
			},
			wantErr: true,
		},
		{
			name: "valid config with credentials",
			config: Config{
				Config: s3client.Config{
					Region:    "us-east-1",
					AccessKey: "AKIAIOSFODNN7EXAMPLE",
					SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Bucket:    "test-bucket",
				ObjectKey: "path/to/file.json",
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
