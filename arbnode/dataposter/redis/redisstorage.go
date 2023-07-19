// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package redis

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/go-redis/redis/v8"
	"github.com/offchainlabs/nitro/arbnode/dataposter/storage"
	"github.com/offchainlabs/nitro/util/signature"
)

// Storage implements redis sorted set backed storage. It does not support
// duplicate keys or values. That is, putting the same element on different
// indexes will not yield expected behavior.
// More  at: https://redis.io/commands/zadd/.
type Storage[Item any] struct {
	client redis.UniversalClient
	signer *signature.SimpleHmac
	key    string
}

func NewStorage[Item any](client redis.UniversalClient, key string, signerConf *signature.SimpleHmacConfig) (*Storage[Item], error) {
	signer, err := signature.NewSimpleHmac(signerConf)
	if err != nil {
		return nil, err
	}
	return &Storage[Item]{client, signer, key}, nil
}

func joinHmacMsg(msg []byte, sig []byte) ([]byte, error) {
	if len(sig) != 32 {
		return nil, errors.New("signature is wrong length")
	}
	return append(sig, msg...), nil
}

func (s *Storage[Item]) peelVerifySignature(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, errors.New("data is too short to contain message signature")
	}

	err := s.signer.VerifySignature(data[:32], data[32:])
	if err != nil {
		return nil, err
	}
	return data[32:], nil
}

func (s *Storage[Item]) FetchContents(ctx context.Context, startingIndex uint64, maxResults uint64) ([]*Item, error) {
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
		data, err := s.peelVerifySignature([]byte(itemString))
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

func (s *Storage[Item]) FetchLast(ctx context.Context) (*Item, error) {
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
		data, err := s.peelVerifySignature([]byte(itemStrings[0]))
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

func (s *Storage[Item]) Prune(ctx context.Context, until uint64) error {
	if until > 0 {
		return s.client.ZRemRangeByScore(ctx, s.key, "-inf", fmt.Sprintf("%v", until-1)).Err()
	}
	return nil
}

func (s *Storage[Item]) Put(ctx context.Context, index uint64, prevItem *Item, newItem *Item) error {
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
				return fmt.Errorf("%w: tried to replace item at index %v but no item exists there", storage.ErrStorageRace, index)
			}
		} else if len(haveItems) == 1 {
			if prevItem == nil {
				return fmt.Errorf("%w: tried to insert new item at index %v but an item exists there", storage.ErrStorageRace, index)
			}
			verifiedItem, err := s.peelVerifySignature([]byte(haveItems[0]))
			if err != nil {
				return fmt.Errorf("failed to validate item already in redis at index%v: %w", index, err)
			}
			prevItemEncoded, err := rlp.EncodeToBytes(prevItem)
			if err != nil {
				return err
			}
			if !bytes.Equal(verifiedItem, prevItemEncoded) {
				return fmt.Errorf("%w: replacing different item than expected at index %v", storage.ErrStorageRace, index)
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
		sig, err := s.signer.SignMessage(newItemEncoded)
		if err != nil {
			return err
		}
		signedItem, err := joinHmacMsg(newItemEncoded, sig)
		if err != nil {
			return err
		}
		err = pipe.ZAdd(ctx, s.key, &redis.Z{
			Score:  float64(index),
			Member: string(signedItem),
		}).Err()
		if err != nil {
			return err
		}
		_, err = pipe.Exec(ctx)
		if errors.Is(err, redis.TxFailedErr) {
			// Unfortunately, we can't wrap two errors.
			//nolint:errorlint
			err = fmt.Errorf("%w: %v", storage.ErrStorageRace, err.Error())
		}
		return err
	}
	// WATCH works with sorted sets: https://redis.io/docs/manual/transactions/#using-watch-to-implement-zpop
	return s.client.Watch(ctx, action, s.key)
}

func (s *Storage[Item]) Length(ctx context.Context) (int, error) {
	count, err := s.client.ZCount(ctx, s.key, "-inf", "+inf").Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Storage[Item]) IsPersistent() bool {
	return true
}
