// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sqsclient

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/spf13/pflag"
)

type QueueConfig struct {
	QueueURL  string          `koanf:"queue-url"`
	SQSClient SQSClientConfig `koanf:"sqs-client"`
}

var DefaultQueueConfig = QueueConfig{}

func (c *QueueConfig) Validate() error {
	if c.QueueURL == "" {
		return errors.New("queue-url is required")
	}
	return nil
}

func QueueConfigAddOptions(prefix string, f *pflag.FlagSet) {
	f.String(prefix+".queue-url", DefaultQueueConfig.QueueURL, "SQS queue URL for filtered transaction reports")
	SQSClientConfigAddOptions(prefix+".sqs-client", f)
}

type QueueClient interface {
	Send(ctx context.Context, body string) error
	Receive(ctx context.Context, waitTimeSecs, maxMessages int32) ([]sqstypes.Message, error)
	Delete(ctx context.Context, receiptHandle string) error
}

type queueClient struct {
	sqsClient *sqs.Client
	queueURL  string
}

func NewQueueClient(ctx context.Context, config *QueueConfig) (QueueClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	sqsClient, err := NewSQSClient(ctx, &config.SQSClient)
	if err != nil {
		return nil, err
	}
	return &queueClient{sqsClient: sqsClient, queueURL: config.QueueURL}, nil
}

func (q *queueClient) Send(ctx context.Context, body string) error {
	_, err := q.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &q.queueURL,
		MessageBody: &body,
	})
	return err
}

func (q *queueClient) Receive(ctx context.Context, waitTimeSecs, maxMessages int32) ([]sqstypes.Message, error) {
	out, err := q.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &q.queueURL,
		WaitTimeSeconds:     waitTimeSecs,
		MaxNumberOfMessages: maxMessages,
	})
	if err != nil {
		return nil, err
	}
	return out.Messages, nil
}

func (q *queueClient) Delete(ctx context.Context, receiptHandle string) error {
	_, err := q.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &q.queueURL,
		ReceiptHandle: &receiptHandle,
	})
	return err
}
