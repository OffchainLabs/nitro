// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package forwarder

import (
	"context"
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
	Workers          int                          `koanf:"workers"`
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

func (c *Config) Validate() error {
	return c.ExternalEndpoint.Validate()
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Int(prefix+".workers", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".wait-time-seconds", DefaultConfig.WaitTimeSeconds, "SQS long polling wait time in seconds")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
}

type Forwarder struct {
	stopwaiter.StopWaiter
	config      *Config
	queueClient sqsclient.QueueClient
	httpClient  *http.Client
}

func New(config *Config, queueClient sqsclient.QueueClient) *Forwarder {
	return &Forwarder{
		config:      config,
		queueClient: queueClient,
		httpClient:  &http.Client{Timeout: config.ExternalEndpoint.Timeout},
	}
}

func (r *Forwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	for i := 0; i < r.config.Workers; i++ {
		r.CallIteratively(r.pollAndForward)
	}
}

func (r *Forwarder) pollAndForward(ctx context.Context) time.Duration {
	msgs, err := r.queueClient.Receive(ctx, r.config.WaitTimeSeconds, 1)
	if err != nil {
		log.Error("Failed to receive SQS messages", "err", err)
		return r.config.PollInterval
	}
	if len(msgs) == 0 {
		return r.config.PollInterval
	}
	msg := msgs[0]
	msgID := "<unknown>"
	if msg.MessageId != nil {
		msgID = *msg.MessageId
	}
	if msg.Body == nil {
		log.Warn("Received SQS message with nil body, deleting", "messageId", msgID)
		if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
			log.Error("Failed to delete nil-body SQS message", "err", err, "messageId", msgID)
		}
		return 0
	}
	if err := r.forwardToEndpoint(ctx, *msg.Body); err != nil {
		log.Warn("Failed to forward report to external endpoint", "err", err, "messageId", msgID)
		return 0
	}
	if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", msgID)
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
		_, err = io.Copy(io.Discard, resp.Body)
		if err != nil {
			log.Error("Failed draining response body", "err", err)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
