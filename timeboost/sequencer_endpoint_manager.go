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
		if err == nil && m.clientUrl == sequencerUrl {
			return m.client, false, nil
		}
		// Sequencer changed, close old client
		m.client.Close()
		m.client = nil
	}

	client, err := createRPCClient(ctx, sequencerUrl, m.jwtPath)
	if err != nil {
		return nil, false, err
	}
	log.Info("Created sequencer client", "url", sequencerUrl)

	m.client = client
	m.clientUrl = sequencerUrl
	return client, true, nil
}

type StaticEndpointManager struct {
	endpoint string
	jwtPath  string
	client   *rpc.Client
}

func NewStaticEndpointManager(endpoint string, jwtPath string) SequencerEndpointManager {
	return &StaticEndpointManager{
		endpoint: endpoint,
		jwtPath:  jwtPath,
	}
}

func (m *StaticEndpointManager) GetSequencerRPC(ctx context.Context) (*rpc.Client, bool, error) {
	new := false
	if m.client == nil {
		client, err := createRPCClient(ctx, m.endpoint, m.jwtPath)
		if err != nil {
			return nil, false, err
		}
		m.client = client
		new = true
	}
	return m.client, new, nil
}

func createRPCClient(ctx context.Context, endpoint string, jwtPath string) (*rpc.Client, error) {
	if jwtPath == "" {
		return rpc.DialContext(ctx, endpoint)
	}

	// Create RPC client with JWT auth
	sequencerJwtStr, err := os.ReadFile(jwtPath)
	if err != nil {
		return nil, err
	}
	sequencerJwt, err := hexutil.Decode(string(sequencerJwtStr))
	if err != nil {
		return nil, err
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
