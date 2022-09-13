// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package dataposter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/util/simple_hmac"
)

// Requires that Item is RLP encodable/decodable
type RedisStorage[Item any] struct {
	client redis.UniversalClient
	signer *simple_hmac.SimpleHmac
	key    string
}

func NewRedisStorage[Item any](client redis.UniversalClient, key string, signerConf *simple_hmac.SimpleHmacConfig) (*RedisStorage[Item], error) {
	signer, err := simple_hmac.NewSimpleHmac(signerConf)
	if err != nil {
		return nil, err
	}
	return &RedisStorage[Item]{client, signer, key}, nil
}

func (s *RedisStorage[Item]) GetContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
	query := redis.ZRangeArgs{
		Key:     s.key,
		ByScore: true,
		Start:   startingIndex,
		Stop:    startingIndex + maxResults - 1,
	}
	itemStrings, err := s.client.ZRangeArgs(ctx, query).Result()
	if err != nil {
		return nil, err
	}
	var items []*Item
	for _, itemString := range itemStrings {
		var item Item
		data, err := s.signer.VerifyMessageSignature(nil, []byte(itemString))
		if err != nil {
			return nil, err
		}
		err = rlp.DecodeBytes(data, &item)
		if err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (s *RedisStorage[Item]) GetLast(ctx context.Context) (*Item, error) {
	query := redis.ZRangeArgs{
		Key:   s.key,
		Start: 0,
		Stop:  0,
		Rev:   true,
	}
	itemStrings, err := s.client.ZRangeArgs(ctx, query).Result()
	if err != nil {
		return nil, err
	}
	if len(itemStrings) > 1 {
		return nil, fmt.Errorf("expected only one return value for GetLast but got %v", len(itemStrings))
	}
	var ret *Item
	if len(itemStrings) > 0 {
		var item Item
		data, err := s.signer.VerifyMessageSignature(nil, []byte(itemStrings[0]))
		if err != nil {
			return nil, err
		}
		err = rlp.DecodeBytes(data, &item)
		if err != nil {
			return nil, err
		}
		ret = &item
	}
	return ret, nil
}

func (s *RedisStorage[Item]) Prune(ctx context.Context, keepStartingAt uint64) error {
	if keepStartingAt > 0 {
		return s.client.ZRemRangeByScore(ctx, s.key, "-inf", fmt.Sprintf("%v", keepStartingAt-1)).Err()
	}
	return nil
}

var StorageRaceErr = errors.New("storage race error")

func (s *RedisStorage[Item]) Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error {
	if newItem == nil {
		return fmt.Errorf("tried to insert nil item at index %v", index)
	}
	action := func(tx *redis.Tx) error {
		query := redis.ZRangeArgs{
			Key:     s.key,
			ByScore: true,
			Start:   index,
			Stop:    index,
		}
		haveItems, err := s.client.ZRangeArgs(ctx, query).Result()
		if err != nil {
			return err
		}
		pipe := tx.TxPipeline()
		if len(haveItems) == 0 {
			if prevItem != nil {
				return fmt.Errorf("%w: tried to replace item at index %v but no item exists there", StorageRaceErr, index)
			}
		} else if len(haveItems) == 1 {
			if prevItem == nil {
				return fmt.Errorf("%w: tried to insert new item at index %v but an item exists there", StorageRaceErr, index)
			}
			verifiedItem, err := s.signer.VerifyMessageSignature(nil, []byte(haveItems[0]))
			if err != nil {
				return fmt.Errorf("failed to validate item already in redis at index%v: %w", index, err)
			}
			prevItemEncoded, err := rlp.EncodeToBytes(prevItem)
			if err != nil {
				return err
			}
			if !bytes.Equal(verifiedItem, prevItemEncoded) {
				return fmt.Errorf("%w: replacing different item than expected at index %v", StorageRaceErr, index)
			}
			err = pipe.ZRem(ctx, s.key, haveItems[0]).Err()
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("expected only one return value for Put but got %v", len(haveItems))
		}
		newItemEncoded, err := rlp.EncodeToBytes(*newItem)
		if err != nil {
			return err
		}
		err = pipe.ZAdd(ctx, s.key, &redis.Z{
			Score:  float64(index),
			Member: string(s.signer.SignMessage(nil, newItemEncoded)),
		}).Err()
		if err != nil {
			return err
		}
		_, err = pipe.Exec(ctx)
		if errors.Is(err, redis.TxFailedErr) {
			// Unfortunately, we can't wrap two errors.
			err = fmt.Errorf("%w: %v", StorageRaceErr, err.Error())
		}
		return err
	}
	// WATCH works with sorted sets: https://redis.io/docs/manual/transactions/#using-watch-to-implement-zpop
	return s.client.Watch(ctx, action, s.key)
}
