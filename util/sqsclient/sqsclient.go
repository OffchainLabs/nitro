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

type Client interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

type Config struct {
	Enable   bool   `koanf:"enable"`
	Region   string `koanf:"region"`
	Endpoint string `koanf:"endpoint"`
	QueueURL string `koanf:"queue-url"`

	AccessKey string `koanf:"access-key"`
	SecretKey string `koanf:"secret-key"`
}

var DefaultConfig = Config{}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable SQS reporting of filtered transactions")
	f.String(prefix+".region", DefaultConfig.Region, "SQS region")
	f.String(prefix+".endpoint", DefaultConfig.Endpoint, "custom SQS endpoint URL (for localstack or other SQS-compatible services)")
	f.String(prefix+".queue-url", DefaultConfig.QueueURL, "SQS queue URL for filtered transaction reports")
	f.String(prefix+".access-key", DefaultConfig.AccessKey, "SQS access key")
	f.String(prefix+".secret-key", DefaultConfig.SecretKey, "SQS secret key")
}

func NewClient(ctx context.Context, config *Config) (Client, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(config.Region), func(options *awsConfig.LoadOptions) error {
		if config.AccessKey != "" && config.SecretKey != "" {
			options.Credentials = credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, "")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if config.Endpoint != "" {
		return sqs.NewFromConfig(cfg, func(o *sqs.Options) {
			o.BaseEndpoint = &config.Endpoint
		}), nil
	}
	return sqs.NewFromConfig(cfg), nil
}
