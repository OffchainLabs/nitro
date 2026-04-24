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
	CircuitBreaker     CircuitBreakerConfig         `koanf:"circuit-breaker"`
}

var DefaultConfig = Config{
	Workers:            1,
	PollInterval:       1 * time.Second,
	SQSWaitTimeSeconds: 5,
	ExternalEndpoint:   genericconf.HTTPClientConfigDefault,
	CircuitBreaker:     DefaultCircuitBreakerConfig,
}

func (c *Config) Validate() error {
	if c.PollInterval < 0 {
		return fmt.Errorf("poll-interval must be non-negative, got %s", c.PollInterval)
	}
	if c.SQSWaitTimeSeconds < 0 {
		return fmt.Errorf("sqs-wait-time-seconds must be non-negative, got %d", c.SQSWaitTimeSeconds)
	}
	if err := c.ExternalEndpoint.Validate(); err != nil {
		return err
	}
	return c.CircuitBreaker.Validate()
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint(prefix+".workers", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".sqs-wait-time-seconds", DefaultConfig.SQSWaitTimeSeconds, "SQS long polling wait time in seconds")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
	CircuitBreakerConfigAddOptions(prefix+".circuit-breaker", f)
}

type Forwarder struct {
	stopwaiter.StopWaiter
	config      *Config
	queueClient sqsclient.QueueClient
	httpClient  *http.Client
	breaker     *Breaker
}

func New(config *Config, queueClient sqsclient.QueueClient) (*Forwarder, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}
	if queueClient == nil {
		return nil, errors.New("queueClient must not be nil")
	}
	f := &Forwarder{
		config:      config,
		queueClient: queueClient,
		httpClient:  &http.Client{Timeout: config.ExternalEndpoint.Timeout},
	}
	if config.CircuitBreaker.Enabled {
		f.breaker = NewBreaker(&config.CircuitBreaker, nil)
	}
	return f, nil
}

func (r *Forwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	for i := uint(0); i < r.config.Workers; i++ {
		r.CallIteratively(r.pollAndForward)
	}
}

func (r *Forwarder) pollAndForward(ctx context.Context) time.Duration {
	if r.breaker != nil && !r.breaker.Allow() {
		return r.config.PollInterval
	}
	msgs, err := r.queueClient.Receive(ctx, r.config.SQSWaitTimeSeconds, 1)
	if err != nil {
		log.Error("Failed to receive SQS messages", "err", err)
		return r.config.PollInterval
	}
	if len(msgs) == 0 {
		return r.config.PollInterval
	}
	msg := msgs[0]
	outcome, forwardErr := r.forwardToEndpoint(ctx, *msg.Body)
	if r.breaker != nil {
		switch outcome {
		case outcomeSuccess:
			r.breaker.Record(true)
		case outcomeEndpointFailure:
			r.breaker.Record(false)
		}
	}
	if forwardErr != nil {
		log.Error("Failed to forward report to external endpoint", "err", forwardErr, "messageId", *msg.MessageId)
		return 0
	}
	if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	}
	return 0
}

type forwardOutcome int

const (
	outcomeSuccess forwardOutcome = iota
	// outcomeEndpointFailure signals that the external endpoint is unhealthy
	// (transport error, 5xx, or 429) and should count against the breaker.
	outcomeEndpointFailure
	// outcomeNonBreakerError covers request-construction failures and 4xx
	// responses other than 429: the endpoint is alive but rejecting our
	// payload, so retrying in-process won't help and the breaker shouldn't
	// trip on it.
	outcomeNonBreakerError
)

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) (forwardOutcome, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.ExternalEndpoint.URL, strings.NewReader(body))
	if err != nil {
		return outcomeNonBreakerError, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return outcomeNonBreakerError, err
		}
		return outcomeEndpointFailure, err
	}
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("Failed draining response body", "err", drainErr)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return outcomeSuccess, nil
	}
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if readErr != nil {
		return outcomeForStatus(resp.StatusCode), fmt.Errorf("external endpoint returned status %d (body read error: %w)", resp.StatusCode, readErr)
	}
	return outcomeForStatus(resp.StatusCode), fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(respBody))
}

func outcomeForStatus(status int) forwardOutcome {
	if status >= 500 || status == http.StatusTooManyRequests || status == http.StatusRequestTimeout {
		return outcomeEndpointFailure
	}
	return outcomeNonBreakerError
}
