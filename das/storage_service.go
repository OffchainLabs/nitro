// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/offchainlabs/nitro/arbstate"
)

var ErrNotFound = errors.New("not found")

type StorageService interface {
	arbstate.DataAvailabilityReader
	Put(ctx context.Context, data []byte, expirationTime uint64) error
	Sync(ctx context.Context) error
	Closer
	fmt.Stringer
	HealthCheck(ctx context.Context) error
}

func EncodeStorageServiceKey(key common.Hash) string {
	return key.Hex()[2:]
}

func DecodeStorageServiceKey(input string) (common.Hash, error) {
	if !strings.HasPrefix(input, "0x") {
		input = "0x" + input
	}
	key, err := hexutil.Decode(input)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(key), nil
}
