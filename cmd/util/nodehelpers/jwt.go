package nodehelpers

import (
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func TryCreatingJWTSecret(filename string) {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		log.Crit("couldn't create directory for jwt secret", "err", err, "dirName", filepath.Dir(filename))
		return
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fs.FileMode(0600))
	if errors.Is(err, fs.ErrExist) {
		log.Info("using existing jwt file", "filename", filename)
		return
	} else if err != nil {
		log.Crit("couldn't create jwt secret file", "err", err, "filename", filename)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error("failed to close jwt secret file", "err", err)
		}
	}()
	secret := common.Hash{}
	_, err = rand.Read(secret[:])
	if err != nil {
		log.Crit("couldn't create jwt secret", "err", err, "filename", filename)
		return
	}
	_, err = f.Write([]byte(secret.Hex()))
	if err != nil {
		log.Crit("couldn't write jwt secret", "err", err, "filename", filename)
		return
	}
	log.Info("created jwt file", "filename", filename)
}
