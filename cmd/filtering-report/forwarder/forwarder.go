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
	Workers              uint                         `koanf:"workers"`
	PollInterval         time.Duration                `koanf:"poll-interval"`
	SQSWaitTimeSeconds   int32                        `koanf:"sqs-wait-time-seconds"`
	SQSVisibilityTimeout time.Duration                `koanf:"sqs-visibility-timeout"`
	ExternalEndpoint     genericconf.HTTPClientConfig `koanf:"external-endpoint"`
	MaxRetries           uint                         `koanf:"max-retries"`
	InitialBackoff       time.Duration                `koanf:"initial-backoff"`
	MaxBackoff           time.Duration                `koanf:"max-backoff"`
	BackoffMultiplier    float64                      `koanf:"backoff-multiplier"`
}

var DefaultConfig = Config{
	Workers:              1,
	PollInterval:         1 * time.Second,
	SQSWaitTimeSeconds:   5,
	SQSVisibilityTimeout: 30 * time.Second,
	ExternalEndpoint:     genericconf.HTTPClientConfigDefault,
	MaxRetries:           3,
	InitialBackoff:       200 * time.Millisecond,
	MaxBackoff:           5 * time.Second,
	BackoffMultiplier:    2.0,
}

func (c *Config) Validate() error {
	if c.PollInterval < 0 {
		return fmt.Errorf("poll-interval must be non-negative, got %s", c.PollInterval)
	}
	if c.SQSWaitTimeSeconds < 0 {
		return fmt.Errorf("sqs-wait-time-seconds must be non-negative, got %d", c.SQSWaitTimeSeconds)
	}
	if c.InitialBackoff < 0 {
		return fmt.Errorf("initial-backoff must be non-negative, got %s", c.InitialBackoff)
	}
	if c.MaxBackoff < c.InitialBackoff {
		return fmt.Errorf("max-backoff (%s) must be >= initial-backoff (%s)", c.MaxBackoff, c.InitialBackoff)
	}
	if c.BackoffMultiplier < 1.0 {
		return fmt.Errorf("backoff-multiplier must be >= 1.0, got %f", c.BackoffMultiplier)
	}
	if c.SQSVisibilityTimeout <= 0 {
		return fmt.Errorf("sqs-visibility-timeout must be positive, got %s", c.SQSVisibilityTimeout)
	}
	if err := c.ExternalEndpoint.Validate(); err != nil {
		return err
	}
	worstCase := c.worstCaseRetryBudget()
	if worstCase > c.SQSVisibilityTimeout {
		return fmt.Errorf(
			"worst-case retry budget %s exceeds sqs-visibility-timeout %s; reduce max-retries / max-backoff / external-endpoint.timeout or raise sqs-visibility-timeout",
			worstCase, c.SQSVisibilityTimeout,
		)
	}
	return nil
}

// worstCaseRetryBudget returns an upper bound on the wall-clock time a single
// SQS message can spend in the forwarder's retry loop: every attempt uses its
// full HTTP timeout, plus the exponentially-growing (capped) sleep between
// attempts. Used by Validate to guarantee the retry window fits inside the
// SQS visibility timeout, so a message can't be retried in-process and
// redelivered to another worker concurrently.
func (c *Config) worstCaseRetryBudget() time.Duration {
	var totalBackoff time.Duration
	delay := c.InitialBackoff
	for i := uint(0); i < c.MaxRetries; i++ {
		totalBackoff += min(delay, c.MaxBackoff)
		delay = time.Duration(float64(delay) * c.BackoffMultiplier)
	}
	// #nosec G115 -- MaxRetries is a small config value, attempt count fits in time.Duration math.
	totalRequestTime := time.Duration(c.MaxRetries+1) * c.ExternalEndpoint.Timeout
	return totalBackoff + totalRequestTime
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint(prefix+".workers", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".sqs-wait-time-seconds", DefaultConfig.SQSWaitTimeSeconds, "SQS long polling wait time in seconds")
	f.Duration(prefix+".sqs-visibility-timeout", DefaultConfig.SQSVisibilityTimeout, "SQS message visibility timeout; must match the queue's server-side setting (used to validate that the worst-case retry budget fits in the visibility window)")
	f.Uint(prefix+".max-retries", DefaultConfig.MaxRetries, "maximum number of retries per external-endpoint call (0 means a single attempt)")
	f.Duration(prefix+".initial-backoff", DefaultConfig.InitialBackoff, "initial backoff delay between external-endpoint retries")
	f.Duration(prefix+".max-backoff", DefaultConfig.MaxBackoff, "maximum backoff delay between external-endpoint retries")
	f.Float64(prefix+".backoff-multiplier", DefaultConfig.BackoffMultiplier, "multiplier applied to the backoff delay after each retry")
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
		log.Warn("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
		return 0
	}
	if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	}
	return 0
}

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) error {
	var lastErr error
	delay := r.config.InitialBackoff
	for attempt := uint(0); attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(min(delay, r.config.MaxBackoff)):
			}
			delay = time.Duration(float64(delay) * r.config.BackoffMultiplier)
		}
		retry, err := r.doForwardAttempt(ctx, body)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			return lastErr
		}
		log.Debug("External endpoint call failed, will retry", "err", err, "attempt", attempt, "maxRetries", r.config.MaxRetries)
	}
	return lastErr
}

// doForwardAttempt performs a single HTTP POST to the external endpoint. The
// returned retry flag tells forwardToEndpoint whether the error class is worth
// retrying: transport errors and retryable status codes (5xx, 429) are; other
// 4xx responses and request-construction errors are not.
func (r *Forwarder) doForwardAttempt(ctx context.Context, body string) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.ExternalEndpoint.URL, strings.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return true, err
	}
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("Failed draining response body", "err", drainErr)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if readErr != nil {
		log.Warn("Failed reading response body", "err", readErr)
	}
	return isRetryableStatus(resp.StatusCode), fmt.Errorf("external endpoint returned status %d: %s", resp.StatusCode, string(respBody))
}

func isRetryableStatus(status int) bool {
	return status >= 500 || status == http.StatusTooManyRequests
}
