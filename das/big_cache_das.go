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

	"github.com/allegro/bigcache/v3"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/offchainlabs/nitro/arbstate"
)

const bigCacheMessageKeyPrefix string = "bigCacheDas.msg."
const bigCacheCertKeyPrefix string = "bigCacheDas.cert."

type BigCacheConfig struct {
	// TODO add other config information like HardMaxCacheSize
	Expiration time.Duration `koanf:"big-cache-expiration"`
}

var DefaultBigCacheConfig = BigCacheConfig{
	Expiration: time.Hour,
}

type BigCacheDAS struct {
	das            DataAvailabilityService
	bigCacheConfig BigCacheConfig
	bigCache       *bigcache.BigCache
}

func NewBigCacheDAS(bigCacheConfig BigCacheConfig, das DataAvailabilityService) (*BigCacheDAS, error) {
	bigCache, err := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	if err != nil {
		return nil, err
	}
	return &BigCacheDAS{
		das:            das,
		bigCacheConfig: bigCacheConfig,
		bigCache:       bigCache,
	}, nil
}

func (b *BigCacheDAS) setMessageAndCert(message []byte, c *arbstate.DataAvailabilityCertificate, path string) error {
	err := b.bigCache.Set(bigCacheMessageKeyPrefix+path, message)
	if err != nil {
		return err
	}
	cBytes, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = b.bigCache.Set(bigCacheCertKeyPrefix+path, cBytes)
	return err
}

func (b *BigCacheDAS) Store(ctx context.Context, message []byte, timeout uint64) (c *arbstate.DataAvailabilityCertificate, err error) {
	path := base32.StdEncoding.EncodeToString(crypto.Keccak256(message))
	cBytes, err := b.bigCache.Get(bigCacheCertKeyPrefix + path)
	if err != nil {
		c, err := b.das.Store(ctx, message, timeout)
		if err != nil {
			return nil, err
		}
		err = b.setMessageAndCert(message, c, path)
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

func (b *BigCacheDAS) Retrieve(ctx context.Context, certBytes []byte) ([]byte, error) {
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(certBytes))
	if err != nil {
		return nil, err
	}

	path := base32.StdEncoding.EncodeToString(cert.DataHash[:])

	result, err := b.bigCache.Get(bigCacheMessageKeyPrefix + path)
	if err != nil {
		result, err = b.das.Retrieve(ctx, certBytes)
		if err != nil {
			return nil, err
		}

		err = b.setMessageAndCert(result, cert, path)
		if err != nil {
			return nil, err
		}
		return result, err
	}

	return result, err
}

func (b *BigCacheDAS) String() string {
	return fmt.Sprintf("BigCacheDAS{bigCacheConfig:%v}", b.bigCacheConfig)
}
