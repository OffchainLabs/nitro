// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/offchainlabs/nitro/cmd/daprovider/das/dasutil"
)

var ErrNotFound = errors.New("not found")

type StorageService interface {
	dasutil.DASReader
	Put(ctx context.Context, data []byte, expirationTime uint64) error
	Sync(ctx context.Context) error
	Closer
	fmt.Stringer
	HealthCheck(ctx context.Context) error
}

const defaultStorageRetention = time.Hour * 24 * 21 // 6 days longer than the batch poster default

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
