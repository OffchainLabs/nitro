// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"errors"
	"os"
	"sync"
)

var ErrNotFound = errors.New("Not found")

type StorageService interface {
	Read(ctx context.Context, key []byte) ([]byte, error)
	Write(ctx context.Context, key []byte, value []byte, timeout uint64) error
	Sync(ctx context.Context) error
	String() string
}

type LocalDiskStorageService struct {
	dataDir string
	mutex   sync.RWMutex
}

func NewLocalDiskStorageService(dataDir string) StorageService {
	return &LocalDiskStorageService{dataDir: dataDir}
}

func (s *LocalDiskStorageService) Read(ctx context.Context, key []byte) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(key)
	return os.ReadFile(pathname)
}

func (s *LocalDiskStorageService) Write(ctx context.Context, key []byte, value []byte, timeout uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(key)
	return os.WriteFile(pathname, value, 0600)
}

func (s *LocalDiskStorageService) Sync(ctx context.Context) error {
	return nil
}

func (s *LocalDiskStorageService) String() string {
	return "LocalDiskStorageService(" + s.dataDir + ")"
}
