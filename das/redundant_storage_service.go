// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"sync"
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

func (r *RedundantStorageService) Read(ctx context.Context, key []byte) ([]byte, error) {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var anyError error
	responsesExpected := len(r.innerServices)
	resultChan := make(chan readResponse, responsesExpected)
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			data, err := s.Read(subCtx, key)
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

func (r *RedundantStorageService) Write(ctx context.Context, key []byte, value []byte, expirationTime uint64) error {
	var wg sync.WaitGroup
	var errorMutex sync.Mutex
	var anyError error
	wg.Add(len(r.innerServices))
	for _, serv := range r.innerServices {
		go func(s StorageService) {
			err := s.Write(ctx, key, value, expirationTime)
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

func (r *RedundantStorageService) String() string {
	str := "RedundantStorageService("
	for _, serv := range r.innerServices {
		str = str + serv.String() + ","
	}
	return str + ")"
}
