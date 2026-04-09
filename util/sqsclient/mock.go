// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sqsclient

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type MockClient struct {
	mu                    sync.Mutex
	queue                 []sqstypes.Message
	msgCounter            int
	deletedReceiptHandles []string
}

func (m *MockClient) SendMessage(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgCounter++
	msgId := fmt.Sprintf("msg-%d", m.msgCounter)
	rh := fmt.Sprintf("rh-%d", m.msgCounter)
	m.queue = append(m.queue, sqstypes.Message{
		MessageId:     &msgId,
		Body:          params.MessageBody,
		ReceiptHandle: &rh,
	})
	return &sqs.SendMessageOutput{}, nil
}

func (m *MockClient) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit := int(params.MaxNumberOfMessages)
	if limit == 0 {
		limit = 1
	}
	if limit < 1 || limit > 10 {
		return nil, fmt.Errorf("invalid parameter: MaxNumberOfMessages must be between 1 and 10, got %d", limit)
	}
	if len(m.queue) == 0 {
		return &sqs.ReceiveMessageOutput{}, nil
	}
	n := min(len(m.queue), limit)
	msgs := make([]sqstypes.Message, n)
	copy(msgs, m.queue[:n])
	m.queue = m.queue[n:]
	return &sqs.ReceiveMessageOutput{Messages: msgs}, nil
}

func (m *MockClient) DeleteMessage(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedReceiptHandles = append(m.deletedReceiptHandles, *params.ReceiptHandle)
	return &sqs.DeleteMessageOutput{}, nil
}

func (m *MockClient) DeletedReceiptHandles() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.deletedReceiptHandles))
	copy(result, m.deletedReceiptHandles)
	return result
}
