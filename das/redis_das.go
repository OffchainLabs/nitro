// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbstate"
)

const MESSAGE_KEY_PREFIX string = "redisDas.msg."
const CERT_KEY_PREFIX string = "redisDas.cert."

type RedisConfig struct {
	RedisUrl   string        `koanf:"redis-url"`
	Expiration time.Duration `koanf:"redis-expiration"`
}

var DefaultRedisConfig = RedisConfig{
	RedisUrl:   "",
	Expiration: time.Hour,
}

type RedisDAS struct {
	das         DataAvailabilityService
	redisConfig RedisConfig
	client      redis.UniversalClient
}

func NewRedisDataAvailabilityService(redisConfig RedisConfig, das DataAvailabilityService) (*RedisDAS, error) {
	redisOptions, err := redis.ParseURL(redisConfig.RedisUrl)
	if err != nil {
		return nil, err
	}
	return &RedisDAS{
		das:         das,
		redisConfig: redisConfig,
		client:      redis.NewClient(redisOptions),
	}, nil
}
func (r *RedisDAS) setMessageAndCert(ctx context.Context, message []byte, c *arbstate.DataAvailabilityCertificate, path string) error {
	r.client.Set(ctx, MESSAGE_KEY_PREFIX+path, message, r.redisConfig.Expiration)
	cBytes, err := json.Marshal(c)
	if err != nil {
		return err
	}
	r.client.Set(ctx, CERT_KEY_PREFIX+path, cBytes, r.redisConfig.Expiration)
	return nil
}
func (r *RedisDAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	path := base32.StdEncoding.EncodeToString(crypto.Keccak256(message))
	cBytes, err := r.client.Get(ctx, CERT_KEY_PREFIX+path).Bytes()
	if err != nil {
		c, err := r.das.Store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}
		err = r.setMessageAndCert(ctx, message, c, path)
		if err != nil {
			return nil, err
		}
		return c, err
	}

	err = json.Unmarshal(cBytes, c)
	if err != nil {
		return nil, err
	}
	return c, err
}

func (r *RedisDAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])

	result, err := r.client.Get(ctx, MESSAGE_KEY_PREFIX+path).Bytes()
	if err != nil {
		result, err = r.das.Retrieve(ctx, certBytes)
		if err != nil {
			return nil, err
		}

		err = r.setMessageAndCert(ctx, result, cert, path)
		if err != nil {
			return nil, err
		}
		return result, err
	}

	return result, err
}

func (r *RedisDAS) String() string {
	return fmt.Sprintf("RedisDAS{redisConfig:%v}", r.redisConfig)
}
