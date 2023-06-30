// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/pretty"
)

// This is a redundant storage service, which replicates data across a set of StorageServices.
// The implementation assumes that there won't be a large number of replicas.

type RedundantStorageService struct {
	innerServices []StorageService
}

func NewRedundantStorageService(ctx context.Context, services []StorageService) (StorageService, error) {
	innerServices := make([]StorageService, len(services))
	copy(innerServices, services)
	return &RedundantStorageService{innerServices}, nil
}

type readResponse struct {
	data []byte
	err  error
}

func (r *RedundantStorageService) GetByHash(ctx context.Context, key common.Hash) ([]byte, error) {
	log.Trace("das.RedundantStorageService.GetByHash", "key", pretty.PrettyHash(key), "this", r)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var anyError error
	responsesExpected := len(r.innerServices)
	resultChan := make(chan readResponse, responsesExpected)
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			data, err := s.GetByHash(subCtx, key)
			resultChan <- readResponse{data, err}
		}(serv)
	}
	for responsesExpected > 0 {
		select {
		case resp := <-resultChan:
			if resp.err == nil {
				return resp.data, nil
			}
			anyError = resp.err
			responsesExpected--
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, anyError
}

func (r *RedundantStorageService) Put(ctx context.Context, data []byte, expirationTime uint64) error {
	logPut("das.RedundantStorageService.Store", data, expirationTime, r)
	var wg sync.WaitGroup
	var errorMutex sync.Mutex
	var anyError error
	wg.Add(len(r.innerServices))
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			err := s.Put(ctx, data, expirationTime)
			if err != nil {
				errorMutex.Lock()
				anyError = err
				errorMutex.Unlock()
			}
			wg.Done()
		}(serv)
	}
	wg.Wait()
	return anyError
}

func (r *RedundantStorageService) Sync(ctx context.Context) error {
	var wg sync.WaitGroup
	var errorMutex sync.Mutex
	var anyError error
	wg.Add(len(r.innerServices))
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			err := s.Sync(ctx)
			if err != nil {
				errorMutex.Lock()
				anyError = err
				errorMutex.Unlock()
			}
			wg.Done()
		}(serv)
	}
	wg.Wait()
	return anyError
}

func (r *RedundantStorageService) Close(ctx context.Context) error {
	var wg sync.WaitGroup
	var errorMutex sync.Mutex
	var anyError error
	wg.Add(len(r.innerServices))
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			err := s.Close(ctx)
			if err != nil {
				errorMutex.Lock()
				anyError = err
				errorMutex.Unlock()
			}
			wg.Done()
		}(serv)
	}
	wg.Wait()
	return anyError
}

func (r *RedundantStorageService) ExpirationPolicy(ctx context.Context) (arbstate.ExpirationPolicy, error) {
	// If at least one inner service has KeepForever,
	// then whole redundant service can serve after timeout.

	// If no inner service has KeepForever,
	// but at least one inner service has DiscardAfterArchiveTimeout,
	// then whole redundant service can serve till archive timeout.

	// If no inner service has KeepForever, DiscardAfterArchiveTimeout,
	// but at least one inner service has DiscardAfterDataTimeout,
	// then whole redundant service can serve till data timeout.
	var res arbstate.ExpirationPolicy = -1
	for _, serv := range r.innerServices {
		expirationPolicy, err := serv.ExpirationPolicy(ctx)
		if err != nil {
			return -1, err
		}
		switch expirationPolicy {
		case arbstate.KeepForever:
			return arbstate.KeepForever, nil
		case arbstate.DiscardAfterArchiveTimeout:
			res = arbstate.DiscardAfterArchiveTimeout
		case arbstate.DiscardAfterDataTimeout:
			if res != arbstate.DiscardAfterArchiveTimeout {
				res = arbstate.DiscardAfterDataTimeout
			}
		}
	}
	if res == -1 {
		return -1, errors.New("unknown expiration policy")
	}
	return res, nil
}

func (r *RedundantStorageService) String() string {
	str := "RedundantStorageService("
	for _, serv := range r.innerServices {
		str = str + serv.String() + ","
	}
	return str + ")"
}

func (r *RedundantStorageService) HealthCheck(ctx context.Context) error {
	for _, storageService := range r.innerServices {
		err := storageService.HealthCheck(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
