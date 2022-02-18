package statetransfer

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errNoMore = errors.New("no more elements")

type InitDataReader interface {
	Close() error
	GetAddressTableReader() (AddressReader, error)
	GetStoredBlockReader() (StoredBlockReader, error)
	GetRetriableDataReader() (RetriableDataReader, error)
	GetAccountDataReader() (AccountDataReader, error)
}

type ListReader interface {
	More() bool
	Close() error
}

type StoredBlockReader interface {
	ListReader
	GetNext() (*StoredBlock, error)
}

type AddressReader interface {
	ListReader
	GetNext() (*common.Address, error)
}

type RetriableDataReader interface {
	ListReader
	GetNext() (*InitializationDataForRetryable, error)
}

type AccountDataReader interface {
	ListReader
	GetNext() (*AccountInitializationInfo, error)
}
