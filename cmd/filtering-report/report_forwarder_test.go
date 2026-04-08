// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/cmd/filtering-report/api"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

type mockSQSClient struct {
	mu                    sync.Mutex
	queue                 []sqstypes.Message
	msgCounter            int
	deletedReceiptHandles []string
}

func (m *mockSQSClient) SendMessage(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
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

func (m *mockSQSClient) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	max := int(params.MaxNumberOfMessages)
	if max == 0 {
		max = 1
	}
	if len(m.queue) == 0 {
		return &sqs.ReceiveMessageOutput{}, nil
	}
	n := len(m.queue)
	if n > max {
		n = max
	}
	msgs := make([]sqstypes.Message, n)
	copy(msgs, m.queue[:n])
	m.queue = m.queue[n:]
	return &sqs.ReceiveMessageOutput{Messages: msgs}, nil
}

func (m *mockSQSClient) DeleteMessage(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedReceiptHandles = append(m.deletedReceiptHandles, *params.ReceiptHandle)
	return &sqs.DeleteMessageOutput{}, nil
}

const testQueueURL = "https://sqs.test/queue"

func newTestForwarder(mock *mockSQSClient, endpointURL string) *ReportForwarder {
	config := &ReportForwarderConfig{
		Workers:          1,
		PollInterval:     time.Second,
		ExternalEndpoint: endpointURL,
	}
	return NewReportForwarder(config, mock, testQueueURL)
}

func TestReportForwarder_ForwardsMessages(t *testing.T) {
	var mu sync.Mutex
	var receivedBodiesByExternalEndpoint []string
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mu.Lock()
		receivedBodiesByExternalEndpoint = append(receivedBodiesByExternalEndpoint, string(body))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	sqsClient := &mockSQSClient{}

	stackConfig := api.DefaultStackConfig
	_, reportAPI, err := api.NewStack(&stackConfig, sqsClient, testQueueURL)
	if err != nil {
		t.Fatal(err)
	}

	reports := []gethexec.FilteredTxReport{
		{TxHash: common.HexToHash("0x01")},
		{TxHash: common.HexToHash("0x02")},
	}
	if err := reportAPI.ReportFilteredTransactions(context.Background(), reports); err != nil {
		t.Fatal(err)
	}

	forwarder := newTestForwarder(sqsClient, externalEndpointServer.URL)
	forwarder.pollAndForward(context.Background())

	mu.Lock()
	defer mu.Unlock()
	if len(receivedBodiesByExternalEndpoint) != 2 {
		t.Fatalf("expected 2 forwarded messages, got %d", len(receivedBodiesByExternalEndpoint))
	}

	sqsClient.mu.Lock()
	defer sqsClient.mu.Unlock()
	if len(sqsClient.deletedReceiptHandles) != 2 {
		t.Fatalf("expected 2 deletes, got %d", len(sqsClient.deletedReceiptHandles))
	}
}

func TestReportForwarder_EndpointFailure_DoesNotDelete(t *testing.T) {
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer externalEndpointServer.Close()

	sqsClient := &mockSQSClient{}

	stackConfig := api.DefaultStackConfig
	_, reportAPI, err := api.NewStack(&stackConfig, sqsClient, testQueueURL)
	if err != nil {
		t.Fatal(err)
	}

	reports := []gethexec.FilteredTxReport{
		{TxHash: common.HexToHash("0x01")},
	}
	if err := reportAPI.ReportFilteredTransactions(context.Background(), reports); err != nil {
		t.Fatal(err)
	}

	forwarder := newTestForwarder(sqsClient, externalEndpointServer.URL)
	forwarder.pollAndForward(context.Background())

	sqsClient.mu.Lock()
	defer sqsClient.mu.Unlock()
	if len(sqsClient.deletedReceiptHandles) != 0 {
		t.Fatalf("expected 0 deletes on endpoint failure, got %d", len(sqsClient.deletedReceiptHandles))
	}
}

func TestReportForwarder_EmptyQueue(t *testing.T) {
	externalEndpointServerCalled := false
	externalEndpointServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalEndpointServerCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer externalEndpointServer.Close()

	sqsClient := &mockSQSClient{}

	forwarder := newTestForwarder(sqsClient, externalEndpointServer.URL)
	interval := forwarder.pollAndForward(context.Background())

	if externalEndpointServerCalled {
		t.Fatal("expected no HTTP calls on empty queue")
	}
	sqsClient.mu.Lock()
	defer sqsClient.mu.Unlock()
	if len(sqsClient.deletedReceiptHandles) != 0 {
		t.Fatalf("expected 0 deletes on empty queue, got %d", len(sqsClient.deletedReceiptHandles))
	}
	if interval != forwarder.config.PollInterval {
		t.Fatalf("expected poll interval %v on empty queue, got %v", forwarder.config.PollInterval, interval)
	}
}
