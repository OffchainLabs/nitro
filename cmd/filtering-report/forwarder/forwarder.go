// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	Workers          int                          `koanf:"worker-count"`
	PollInterval     time.Duration                `koanf:"poll-interval"`
	WaitTimeSeconds  int32                        `koanf:"wait-time-seconds"`
	ExternalEndpoint genericconf.HTTPClientConfig `koanf:"external-endpoint"`
}

var DefaultConfig = Config{
	Workers:          1,
	PollInterval:     1 * time.Second,
	WaitTimeSeconds:  5,
	ExternalEndpoint: genericconf.HTTPClientConfigDefault,
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultConfig.Enable, "enable SQS consumer workers")
	f.Int(prefix+".worker-count", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".wait-time-seconds", DefaultConfig.WaitTimeSeconds, "SQS long polling wait time in seconds")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
}

type Forwarder struct {
	stopwaiter.StopWaiter
	config     *Config
	sqsClient  *sqsclient.QueueClient
	httpClient *http.Client
}

func New(config *Config, sqsClient *sqsclient.QueueClient) *Forwarder {
	return &Forwarder{
		config:     config,
		sqsClient:  sqsClient,
		httpClient: &http.Client{Timeout: config.ExternalEndpoint.Timeout},
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
	out, err := r.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &r.sqsClient.QueueURL,
		WaitTimeSeconds:     r.config.WaitTimeSeconds,
		MaxNumberOfMessages: 1,
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
		log.Warn("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
		return 0
	}
	_, err = r.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &r.sqsClient.QueueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	}
	return 0
}

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.ExternalEndpoint.URL, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
