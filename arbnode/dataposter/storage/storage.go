package storage

import (
	"errors"
)

var ErrStorageRace = errors.New("storage race error")
