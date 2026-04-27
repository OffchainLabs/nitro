// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/sqsclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type Config struct {
	Workers            uint                         `koanf:"workers"`
	PollInterval       time.Duration                `koanf:"poll-interval"`
	SQSWaitTimeSeconds int32                        `koanf:"sqs-wait-time-seconds"`
	ExternalEndpoint   genericconf.HTTPClientConfig `koanf:"external-endpoint"`
}

var DefaultConfig = Config{
	Workers:            1,
	PollInterval:       1 * time.Second,
	SQSWaitTimeSeconds: 5,
	ExternalEndpoint:   genericconf.HTTPClientConfigDefault,
}

func (c *Config) Validate() error {
	if c.PollInterval < 0 {
		return fmt.Errorf("poll-interval must be non-negative, got %s", c.PollInterval)
	}
	if c.SQSWaitTimeSeconds < 0 {
		return fmt.Errorf("sqs-wait-time-seconds must be non-negative, got %d", c.SQSWaitTimeSeconds)
	}
	return c.ExternalEndpoint.Validate()
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint(prefix+".workers", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".sqs-wait-time-seconds", DefaultConfig.SQSWaitTimeSeconds, "SQS long polling wait time in seconds")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
}

type Forwarder struct {
	stopwaiter.StopWaiter
	config      *Config
	queueClient sqsclient.QueueClient
	httpClient  *http.Client
}

func New(config *Config, queueClient sqsclient.QueueClient) (*Forwarder, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}
	if queueClient == nil {
		return nil, errors.New("queueClient must not be nil")
	}
	return &Forwarder{
		config:      config,
		queueClient: queueClient,
		httpClient:  &http.Client{Timeout: config.ExternalEndpoint.Timeout},
	}, nil
}

func (r *Forwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	for i := uint(0); i < r.config.Workers; i++ {
		r.CallIteratively(r.pollAndForward)
	}
}

func (r *Forwarder) pollAndForward(ctx context.Context) time.Duration {
	msgs, err := r.queueClient.Receive(ctx, r.config.SQSWaitTimeSeconds, 1)
	if err != nil {
		log.Error("Failed to receive SQS messages", "err", err)
		return r.config.PollInterval
	}
	if len(msgs) == 0 {
		return r.config.PollInterval
	}
	msg := msgs[0]
	if err := r.forwardToEndpoint(ctx, *msg.Body); err != nil {
		log.Error("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
		return 0
	}
	if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	}
	return 0
}

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.ExternalEndpoint.URL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("Failed draining response body", "err", drainErr)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024)) // cap error body to avoid unbounded reads
		if readErr != nil {
			return fmt.Errorf("external endpoint returned status %d (body read error: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
