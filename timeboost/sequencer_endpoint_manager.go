// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package timeboost

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/redisutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

type SequencerEndpointManager interface {
	GetSequencerRPC(ctx context.Context) (*rpc.Client, bool, error)
	Close()
}

type RedisEndpointManager struct {
	stopwaiter.StopWaiterSafe
	redisCoordinator *redisutil.RedisCoordinator
	jwtPath          string
	clientMutex      sync.Mutex
	client           *rpc.Client
	clientUrl        string
}

func NewRedisEndpointManager(redisCoordinator *redisutil.RedisCoordinator, jwtPath string) SequencerEndpointManager {
	return &RedisEndpointManager{
		redisCoordinator: redisCoordinator,
		jwtPath:          jwtPath,
	}
}

func (m *RedisEndpointManager) GetSequencerRPC(ctx context.Context) (*rpc.Client, bool, error) {
	sequencerUrl, err := m.redisCoordinator.CurrentChosenSequencer(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get current sequencer: %w", err)
	}
	if sequencerUrl == "" {
		sequencerUrl, err = m.redisCoordinator.RecommendSequencerWantingLockout(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("failed to get recommended sequencer: %w", err)
		}
		if sequencerUrl == "" {
			return nil, false, errors.New("no sequencer available")
		}
	}

	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()

	if m.client != nil {
		// Check if we're still using the correct sequencer
		if m.clientUrl == sequencerUrl {
			return m.client, false, nil
		}
	}

	// Create the new client before closing the old one so that a creation
	// failure doesn't leave the manager without any working connection.
	client, err := createRPCClient(ctx, sequencerUrl, m.jwtPath)
	if err != nil {
		return nil, false, fmt.Errorf("creating RPC client for sequencer %s: %w", sequencerUrl, err)
	}

	if m.client != nil {
		log.Info("Sequencer endpoint changed, closing stale client", "oldUrl", m.clientUrl, "newUrl", sequencerUrl)
		m.client.Close()
	}
	log.Info("Created sequencer client", "url", sequencerUrl)

	m.client = client
	m.clientUrl = sequencerUrl
	return client, true, nil
}

func (m *RedisEndpointManager) Close() {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	if m.redisCoordinator != nil {
		if err := m.redisCoordinator.Client.Close(); err != nil {
			log.Warn("Error closing Redis coordinator client during endpoint manager shutdown", "error", err)
		}
	}
}

type StaticEndpointManager struct {
	endpoint string
	jwtPath  string
	mu       sync.Mutex
	client   *rpc.Client
}

func NewStaticEndpointManager(endpoint string, jwtPath string) SequencerEndpointManager {
	return &StaticEndpointManager{
		endpoint: endpoint,
		jwtPath:  jwtPath,
	}
}

func (m *StaticEndpointManager) GetSequencerRPC(ctx context.Context) (*rpc.Client, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	isNew := false
	if m.client == nil {
		client, err := createRPCClient(ctx, m.endpoint, m.jwtPath)
		if err != nil {
			return nil, false, fmt.Errorf("creating RPC client for static endpoint %s: %w", m.endpoint, err)
		}
		m.client = client
		isNew = true
	}
	return m.client, isNew, nil
}

func (m *StaticEndpointManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
}

func createRPCClient(ctx context.Context, endpoint string, jwtPath string) (*rpc.Client, error) {
	if jwtPath == "" {
		return rpc.DialContext(ctx, endpoint)
	}

	// Create RPC client with JWT auth
	sequencerJwtStr, err := os.ReadFile(jwtPath)
	if err != nil {
		return nil, fmt.Errorf("reading JWT file %s: %w", jwtPath, err)
	}
	sequencerJwt, err := hexutil.Decode(string(sequencerJwtStr))
	if err != nil {
		return nil, fmt.Errorf("decoding JWT file content: %w", err)
	}

	return rpc.DialOptions(ctx, endpoint, rpc.WithHTTPAuth(func(h http.Header) error {
		claims := jwt.MapClaims{
			"iat": time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(sequencerJwt)
		if err != nil {
			return fmt.Errorf("could not produce signed JWT token: %w", err)
		}
		h.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		return nil
	}))
}
