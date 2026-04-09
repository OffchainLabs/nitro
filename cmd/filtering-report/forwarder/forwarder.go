// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/sqsclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Config struct {
	Enable           bool                         `koanf:"enable"`
	Workers      int                          `koanf:"worker-count"`
	PollInterval     time.Duration                `koanf:"poll-interval"`
	ExternalEndpoint genericconf.HTTPClientConfig `koanf:"external-endpoint"`
}

var DefaultConfig = Config{
	Workers:      1,
	PollInterval:     5 * time.Second,
	ExternalEndpoint: genericconf.HTTPClientConfigDefault,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable SQS consumer workers")
	f.Int(prefix+".worker-count", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
}

type Forwarder struct {
	stopwaiter.StopWaiter
	config           *Config
	sqsClient        sqsclient.Client
	sqsQueueURL      string
	httpClient       *http.Client
	externalEndpoint string
}

func New(config *Config, sqsClient sqsclient.Client, sqsQueueURL string) *Forwarder {
	return &Forwarder{
		config:           config,
		sqsClient:        sqsClient,
		sqsQueueURL:      sqsQueueURL,
		httpClient:       &http.Client{Timeout: config.ExternalEndpoint.Timeout},
		externalEndpoint: config.ExternalEndpoint.URL,
	}
}

func (r *Forwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	for i := 0; i < r.config.Workers; i++ {
		r.LaunchThread(func(ctx context.Context) {
			r.CallIteratively(r.pollAndForward)
		})
	}
}

func (r *Forwarder) pollAndForward(ctx context.Context) time.Duration {
	waitTime := int32(5)
	maxMessages := int32(1)
	out, err := r.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &r.sqsQueueURL,
		WaitTimeSeconds:     waitTime,
		MaxNumberOfMessages: maxMessages,
	})
	if err != nil {
		log.Error("Failed to receive SQS messages", "err", err)
		return r.config.PollInterval
	}
	if len(out.Messages) == 0 {
		return r.config.PollInterval
	}
	msg := out.Messages[0]
	if msg.Body == nil {
		return 0
	}
	if err := r.forwardToEndpoint(ctx, *msg.Body); err != nil {
		log.Error("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
		return 0
	}
	_, err = r.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &r.sqsQueueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	}
	return 0
}

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.externalEndpoint, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("external endpoint returned status %d", resp.StatusCode)
	}
	return nil
}
