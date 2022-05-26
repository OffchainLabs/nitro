// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"encoding/base32"
	"fmt"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	flag "github.com/spf13/pflag"
	"golang.org/x/sys/unix"
)

type LocalFileStorageConfig struct {
	Enable  bool   `koanf:"enable"`
	DataDir string `koanf:"data-dir"`
}

var DefaultLocalFileStorageConfig = LocalFileStorageConfig{
	DataDir: "",
}

func LocalFileStorageConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultLocalFileStorageConfig.Enable, "enable storage/retrieval of sequencer batch data from a directory of files, one per batch")
	f.String(prefix+".data-dir", DefaultLocalFileStorageConfig.DataDir, "local data directory")
}

type LocalFileStorageService struct {
	dataDir string
	mutex   sync.RWMutex
}

func NewLocalFileStorageService(dataDir string) (StorageService, error) {
	if unix.Access(dataDir, unix.W_OK|unix.R_OK) != nil {
		return nil, fmt.Errorf("Couldn't start LocalFileStorageService, directory '%s' must be readable and writeable", dataDir)
	}
	return &LocalFileStorageService{dataDir: dataDir}, nil
}

func (s *LocalFileStorageService) GetByHash(ctx context.Context, key []byte) ([]byte, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(key)
	return os.ReadFile(pathname)
}

func (s *LocalFileStorageService) Put(ctx context.Context, data []byte, timeout uint64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	pathname := s.dataDir + "/" + base32.StdEncoding.EncodeToString(crypto.Keccak256(data))
	return os.WriteFile(pathname, data, 0600)
}

func (s *LocalFileStorageService) Sync(ctx context.Context) error {
	return nil
}

func (s *LocalFileStorageService) Close(ctx context.Context) error {
	return nil
}

func (s *LocalFileStorageService) String() string {
	return "LocalFileStorageService(" + s.dataDir + ")"
}
