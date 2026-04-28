// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sqsclient

import (
	"context"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/pflag"
)

type SQSClientConfig struct {
	Region    string `koanf:"region"`
	Endpoint  string `koanf:"endpoint"`
	AccessKey string `koanf:"access-key"`
	SecretKey string `koanf:"secret-key"`
}

func SQSClientConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".region", "", "SQS region")
	f.String(prefix+".endpoint", "", "custom SQS endpoint URL (for localstack or other SQS-compatible services)")
	f.String(prefix+".access-key", "", "SQS access key")
	f.String(prefix+".secret-key", "", "SQS secret key")
}

func NewSQSClient(ctx context.Context, cc *SQSClientConfig) (*sqs.Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(cc.Region), func(options *awsConfig.LoadOptions) error {
		if cc.AccessKey != "" && cc.SecretKey != "" {
			options.Credentials = credentials.NewStaticCredentialsProvider(cc.AccessKey, cc.SecretKey, "")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if cc.Endpoint != "" {
		return sqs.NewFromConfig(cfg, func(o *sqs.Options) {
			o.BaseEndpoint = &cc.Endpoint
		}), nil
	}
	return sqs.NewFromConfig(cfg), nil
}
