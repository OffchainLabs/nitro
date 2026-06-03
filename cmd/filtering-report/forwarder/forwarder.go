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

	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/spf13/pflag"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/cmd/filtering-report/signer"
	"github.com/offchainlabs/nitro/cmd/genericconf"
	"github.com/offchainlabs/nitro/util/httperror"
	"github.com/offchainlabs/nitro/util/sqsclient"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

var (
	externalEndpointRetryableFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/external_endpoint_retryable_failures_total", nil,
	)
	externalEndpointNonRetryableFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/external_endpoint_non_retryable_failures_total", nil,
	)
	externalEndpointSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/external_endpoint_successes_total", nil,
	)
	sqsReceiveFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/sqs_receive_failures_total", nil,
	)
	sqsReceiveSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/sqs_receive_successes_total", nil,
	)
	sqsDeleteFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/sqs_delete_failures_total", nil,
	)
	sqsDeleteSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/sqs_delete_successes_total", nil,
	)
	poisonQueueSendFailuresCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/poison_queue_send_failures_total", nil,
	)
	poisonQueueSendSuccessesCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/poison_queue_send_successes_total", nil,
	)
	externalEndpointSlowdownTriggeredCounter = metrics.NewRegisteredCounter(
		"arb/filtering-report/forwarder/external_endpoint_slowdown_triggered_total", nil,
	)
)

type ExternalEndpointRetryableErrorSlowdownConfig struct {
	Duration                   time.Duration `koanf:"duration"`
	ConsecutiveRetryableErrors int           `koanf:"consecutive-retryable-errors"`
}

var DefaultExternalEndpointRetryableErrorSlowdownConfig = ExternalEndpointRetryableErrorSlowdownConfig{
	Duration:                   2 * time.Minute,
	ConsecutiveRetryableErrors: 4,
}

func (c *ExternalEndpointRetryableErrorSlowdownConfig) Validate() error {
	if c.Duration < 0 {
		return fmt.Errorf("duration must be non-negative, got %s", c.Duration)
	}
	if c.ConsecutiveRetryableErrors <= 0 {
		return fmt.Errorf("consecutive-retryable-errors must be positive, got %d", c.ConsecutiveRetryableErrors)
	}
	return nil
}

func ExternalEndpointRetryableErrorSlowdownConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Duration(prefix+".duration", DefaultExternalEndpointRetryableErrorSlowdownConfig.Duration, "how long a worker sleeps when consecutive retryable errors threshold is reached")
	f.Int(prefix+".consecutive-retryable-errors", DefaultExternalEndpointRetryableErrorSlowdownConfig.ConsecutiveRetryableErrors, "number of consecutive retryable errors a worker encounters before slowing down")
}

type Config struct {
	Workers                                uint                                         `koanf:"workers"`
	PollInterval                           time.Duration                                `koanf:"poll-interval"`
	SQSWaitTimeSeconds                     int32                                        `koanf:"sqs-wait-time-seconds"`
	ExternalEndpoint                       genericconf.HTTPClientConfig                 `koanf:"external-endpoint"`
	ExternalEndpointRetryableErrorSlowdown ExternalEndpointRetryableErrorSlowdownConfig `koanf:"external-endpoint-retryable-error-slowdown"`
	PoisonQueue                            sqsclient.QueueConfig                        `koanf:"poison-queue"`
	Signer                                 signer.Config                                `koanf:"signer"`
}

var DefaultConfig = Config{
	Workers:                                1,
	PollInterval:                           1 * time.Second,
	SQSWaitTimeSeconds:                     5,
	ExternalEndpoint:                       genericconf.HTTPClientConfigDefault,
	ExternalEndpointRetryableErrorSlowdown: DefaultExternalEndpointRetryableErrorSlowdownConfig,
	PoisonQueue:                            sqsclient.DefaultQueueConfig,
	Signer:                                 signer.DefaultConfig,
}

func (c *Config) Validate() error {
	if c.PollInterval < 0 {
		return fmt.Errorf("poll-interval must be non-negative, got %s", c.PollInterval)
	}
	if c.SQSWaitTimeSeconds < 0 {
		return fmt.Errorf("sqs-wait-time-seconds must be non-negative, got %d", c.SQSWaitTimeSeconds)
	}
	if err := c.ExternalEndpointRetryableErrorSlowdown.Validate(); err != nil {
		return err
	}
	if c.PoisonQueue.QueueURL != "" {
		if err := c.PoisonQueue.Validate(); err != nil {
			return err
		}
	}
	if err := c.ExternalEndpoint.Validate(); err != nil {
		return err
	}
	return c.Signer.Validate()
}

func ConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.Uint(prefix+".workers", DefaultConfig.Workers, "number of workers")
	f.Duration(prefix+".poll-interval", DefaultConfig.PollInterval, "interval between SQS polls when queue is empty")
	f.Int32(prefix+".sqs-wait-time-seconds", DefaultConfig.SQSWaitTimeSeconds, "SQS long polling wait time in seconds")
	genericconf.HTTPClientConfigAddOptions(prefix+".external-endpoint", f)
	ExternalEndpointRetryableErrorSlowdownConfigAddOptions(prefix+".external-endpoint-retryable-error-slowdown", f)
	sqsclient.QueueConfigAddOptions(prefix+".poison-queue", f, "SQS queue URL for messages that failed with non-retryable errors")
	signer.ConfigAddOptions(prefix+".signer", f)
}

// Forwarder polls messages from an SQS queue and forwards them to an external
// HTTP endpoint. Error handling follows two paths:
//
//   - Non-retryable HTTP errors (4xx client errors, except 408/425/429) indicate
//     the message itself is permanently invalid. These are sent directly to an
//     optional poison queue and deleted from the main queue. The poison queue can
//     be used for manual inspection, reprocessing, or simply discarded.
//
//   - All other errors — retryable HTTP errors (5xx, 408, 425, 429) and transport
//     errors (DNS, connection refused, TLS, timeouts) — leave the message in the
//     queue for SQS to redeliver after its visibility timeout expires. Since all
//     reports target the same endpoint, a per-worker slowdown kicks in after a
//     configurable number of consecutive retryable errors, preventing workers from
//     hammering a degraded endpoint and avoiding unnecessary consumption of the SQS
//     max receive count quota.
//
// The SQS queue should be configured with a small default visibility timeout
// (e.g. 1-3 minutes), a max receive count (e.g. 150+), and a DLQ (dead-letter
// queue) enabled so messages that exhaust the max receive count are captured
// rather than lost. A small visibility timeout ensures messages become available
// quickly once the endpoint recovers. For example, with a visibility timeout of
// 2 minutes and a max receive count of 150, the minimum retry window per message is
// 150 × 2 min = 300 min = 5 hours. In practice the effective interval is longer
// because the worker slowdown spaces out attempts, so the actual retry window
// will be larger.
type Forwarder struct {
	stopwaiter.StopWaiter
	config            *Config
	queueClient       sqsclient.QueueClient
	poisonQueueClient sqsclient.QueueClient
	httpClient        *http.Client
	signer            *signer.Signer
}

func New(config *Config, queueClient sqsclient.QueueClient, poisonQueueClient sqsclient.QueueClient) (*Forwarder, error) {
	if config == nil {
		return nil, errors.New("config must not be nil")
	}
	if queueClient == nil {
		return nil, errors.New("queueClient must not be nil")
	}
	sgn, err := signer.NewSigner(&config.Signer)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}
	return &Forwarder{
		config:            config,
		queueClient:       queueClient,
		poisonQueueClient: poisonQueueClient,
		httpClient:        &http.Client{Timeout: config.ExternalEndpoint.Timeout},
		signer:            sgn,
	}, nil
}

func (r *Forwarder) Start(ctx context.Context) {
	r.StopWaiter.Start(ctx, r)
	r.StartAndTrackChild(r.signer)
	for i := uint(0); i < r.config.Workers; i++ {
		var consecutiveRetryableErrors int
		r.CallIteratively(func(ctx context.Context) time.Duration {
			return r.pollAndForward(ctx, &consecutiveRetryableErrors)
		})
	}
}

func (r *Forwarder) pollAndForward(ctx context.Context, consecutiveRetryableErrors *int) time.Duration {
	msgs, err := r.queueClient.Receive(ctx, r.config.SQSWaitTimeSeconds, 1)
	if err != nil {
		sqsReceiveFailuresCounter.Inc(1)
		log.Error("Failed to receive SQS messages", "err", err)
		return r.config.PollInterval
	}
	sqsReceiveSuccessesCounter.Inc(1)
	if len(msgs) == 0 {
		return r.config.PollInterval
	}
	msg := msgs[0]
	if err := r.forwardToEndpoint(ctx, *msg.Body); err != nil {
		log.Error("Failed to forward report to external endpoint", "err", err, "messageId", *msg.MessageId)
		var httpErr *httperror.HTTPError
		if errors.As(err, &httpErr) && !httpErr.IsRetryable() {
			externalEndpointNonRetryableFailuresCounter.Inc(1)
			*consecutiveRetryableErrors = 0
			r.sendToPoisonQueue(ctx, msg, httpErr)
			return 0
		}
		externalEndpointRetryableFailuresCounter.Inc(1)
		*consecutiveRetryableErrors++
		if *consecutiveRetryableErrors >= r.config.ExternalEndpointRetryableErrorSlowdown.ConsecutiveRetryableErrors {
			externalEndpointSlowdownTriggeredCounter.Inc(1)
			return r.config.ExternalEndpointRetryableErrorSlowdown.Duration
		}
		return 0
	}
	externalEndpointSuccessesCounter.Inc(1)
	*consecutiveRetryableErrors = 0
	if err = r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		sqsDeleteFailuresCounter.Inc(1)
		log.Error("Failed to delete SQS message after forwarding", "err", err, "messageId", *msg.MessageId)
	} else {
		sqsDeleteSuccessesCounter.Inc(1)
	}
	return 0
}

func (r *Forwarder) sendToPoisonQueue(ctx context.Context, msg sqstypes.Message, httpErr *httperror.HTTPError) {
	if r.poisonQueueClient == nil {
		return
	}
	if err := r.poisonQueueClient.Send(ctx, *msg.Body); err != nil {
		poisonQueueSendFailuresCounter.Inc(1)
		log.Error("Failed to send message to poison queue", "err", err, "messageId", *msg.MessageId,
			"triggerStatusCode", httpErr.StatusCode, "triggerBody", httpErr.Body)
		return
	}
	poisonQueueSendSuccessesCounter.Inc(1)
	log.Info("Sent message to poison queue", "messageId", *msg.MessageId,
		"triggerStatusCode", httpErr.StatusCode, "triggerBody", httpErr.Body)
	if err := r.queueClient.Delete(ctx, *msg.ReceiptHandle); err != nil {
		sqsDeleteFailuresCounter.Inc(1)
		log.Error("Failed to delete SQS message after sending to poison queue", "err", err, "messageId", *msg.MessageId)
	} else {
		sqsDeleteSuccessesCounter.Inc(1)
	}
}

func (r *Forwarder) forwardToEndpoint(ctx context.Context, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.config.ExternalEndpoint.URL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	r.signer.SignHTTPRequest(req, []byte(body), time.Now())
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if _, drainErr := io.Copy(io.Discard, resp.Body); drainErr != nil {
			log.Warn("Failed draining response body", "err", drainErr)
		}
		resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024)) // cap error body to avoid unbounded reads
		respBodyStr := string(respBody)
		if readErr != nil {
			log.Warn("Failed reading external endpoint error response body", "err", readErr, "statusCode", resp.StatusCode)
			respBodyStr = fmt.Sprintf("%s (body read error: %s)", respBodyStr, readErr)
		}
		return &httperror.HTTPError{StatusCode: resp.StatusCode, Body: respBodyStr}
	}
	return nil
}
