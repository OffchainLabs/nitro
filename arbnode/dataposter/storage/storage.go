package storage

import (
	"errors"
)

var (
	ErrStorageRace = errors.New("storage race error")

	BlockValidatorPrefix string = "v" // the prefix for all block validator keys
	DataPosterPrefix     string = "d" // the prefix for all data poster keys
	// TODO(anodar): move everything else from schema.go file to here once
	// execution split is complete.
)
