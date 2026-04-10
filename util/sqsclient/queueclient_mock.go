// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package sqsclient

import (
	"context"
	"fmt"
	"sync"

	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type MockQueueClient struct {
	mu                    sync.Mutex
	queue                 []sqstypes.Message
	msgCounter            int
	deletedReceiptHandles []string
}

func (m *MockQueueClient) Send(_ context.Context, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgCounter++
	msgId := fmt.Sprintf("msg-%d", m.msgCounter)
	rh := fmt.Sprintf("rh-%d", m.msgCounter)
	m.queue = append(m.queue, sqstypes.Message{
		MessageId:     &msgId,
		Body:          &body,
		ReceiptHandle: &rh,
	})
	return nil
}

func (m *MockQueueClient) Receive(_ context.Context, _ int32, maxMessages int32) ([]sqstypes.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	limit := int(maxMessages)
	if limit == 0 {
		limit = 1
	}
	if limit < 1 || limit > 10 {
		return nil, fmt.Errorf("invalid parameter: MaxNumberOfMessages must be between 1 and 10, got %d", limit)
	}
	if len(m.queue) == 0 {
		return nil, nil
	}
	n := min(len(m.queue), limit)
	msgs := make([]sqstypes.Message, n)
	copy(msgs, m.queue[:n])
	m.queue = m.queue[n:]
	return msgs, nil
}

func (m *MockQueueClient) Delete(_ context.Context, receiptHandle string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedReceiptHandles = append(m.deletedReceiptHandles, receiptHandle)
	return nil
}

func (m *MockQueueClient) DeletedReceiptHandles() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.deletedReceiptHandles))
	copy(result, m.deletedReceiptHandles)
	return result
}
