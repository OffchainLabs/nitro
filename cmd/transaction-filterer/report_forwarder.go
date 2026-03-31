// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/util/sqsclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type ReportForwarderConfig struct {
	Enable           bool          `koanf:"enable"`
	Workers          int           `koanf:"workers"`
	PollInterval     time.Duration `koanf:"poll-interval"`
	ExternalEndpoint string        `koanf:"external-endpoint"`
}

var DefaultReportForwarderConfig = ReportForwarderConfig{
	Workers:      1,
	PollInterval: 5 * time.Second,
}

func ReportForwarderConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Bool(prefix+".enable", DefaultReportForwarderConfig.Enable, "enable SQS consumer workers")
	f.Int(prefix+".workers", DefaultReportForwarderConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultReportForwarderConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.String(prefix+".external-endpoint", DefaultReportForwarderConfig.ExternalEndpoint, "HTTP endpoint to forward filtered transaction reports to")
}

type ReportForwarder struct {
	stopwaiter.StopWaiter
	config           *ReportForwarderConfig
	sqsClient        sqsclient.Client
	sqsQueueURL      string
	httpClient       *http.Client
	externalEndpoint string
}

func NewReportForwarder(config *ReportForwarderConfig, sqsClient sqsclient.Client, sqsQueueURL string) *ReportForwarder {
	return &ReportForwarder{
		config:           config,
		sqsClient:        sqsClient,
		sqsQueueURL:      sqsQueueURL,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		externalEndpoint: config.ExternalEndpoint,
	}
}

func (r *ReportForwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	for i := 0; i < r.config.Workers; i++ {
		r.LaunchThread(func(ctx context.Context) {
			r.CallIteratively(r.pollAndForward)
		})
	}
}

func (r *ReportForwarder) pollAndForward(ctx context.Context) time.Duration {
	waitTime := int32(5)
	maxMessages := int32(10)
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
	for _, msg := range out.Messages {
		if msg.Body == nil {
			continue
		}
		if err := r.forwardToEndpoint(ctx, *msg.Body); err != nil {
			log.Error("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
			continue
		}
		_, err := r.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      &r.sqsQueueURL,
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
		}
	}
	return 0
}

func (r *ReportForwarder) forwardToEndpoint(ctx context.Context, body string) error {
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
