package nodehelpers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func TryCreatingJWTSecret(filename string) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return fmt.Errorf("couldn't create directory for jwt secret: %w", err)
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fs.FileMode(0600))
	if errors.Is(err, fs.ErrExist) {
		log.Info("using existing jwt file", "filename", filename)
		return nil
	} else if err != nil {
		return fmt.Errorf("couldn't create file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("failed to close file", "err", err)
		}
	}()
	secret := common.Hash{}
	_, err = rand.Read(secret[:])
	if err != nil {
		return fmt.Errorf("couldn't generate secret: %w", err)
	}
	_, err = f.Write([]byte(secret.Hex()))
	if err != nil {
		return fmt.Errorf("couldn't writeto file: %w", err)
	}
	log.Info("created jwt file", "filename", filename)
	return nil
}
