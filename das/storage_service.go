// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"errors"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
)

var ErrNotFound = errors.New("Not found")

type StorageService interface {
	GetByHash(ctx context.Context, key []byte) ([]byte, error)
	Put(ctx context.Context, data []byte, expirationTime uint64) error
	Sync(ctx context.Context) error
	Close(ctx context.Context) error
	String() string
}

type LocalDiskStorageService struct {
	dataDir string
	mutex   sync.RWMutex
}

func NewLocalDiskStorageService(dataDir string) StorageService {
	return &LocalDiskStorageService{dataDir: dataDir}
}

func (s *LocalDiskStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(key)
	return os.ReadFile(pathname)
}

func (s *LocalDiskStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(crypto.Keccak256(data))
	return os.WriteFile(pathname, data, 0600)
}

func (s *LocalDiskStorageService) Sync(ctx context.Context) error {
	return nil
}

func (s *LocalDiskStorageService) Close(ctx context.Context) error {
	return nil
}

func (s *LocalDiskStorageService) String() string {
	return "LocalDiskStorageService(" + s.dataDir + ")"
}
