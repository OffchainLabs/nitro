// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package statetransfer

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errNoMore = errors.New("no more elements")

type InitDataReader interface {
	Close() error
	GetAddressTableReader() (AddressReader, error)
	GetNextBlockNumber() (uint64, error)
	GetRetryableDataReader() (RetryableDataReader, error)
	GetAccountDataReader() (AccountDataReader, error)
}

type ListReader interface {
	More() bool
	Close() error
}

type AddressReader interface {
	ListReader
	GetNext() (*common.Address, error)
}

type RetryableDataReader interface {
	ListReader
	GetNext() (*InitializationDataForRetryable, error)
}

type AccountDataReader interface {
	ListReader
	GetNext() (*AccountInitializationInfo, error)
}
