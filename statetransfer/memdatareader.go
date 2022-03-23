// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"github.com/ethereum/go-ethereum/common"
)

type MemoryInitDataReader struct {
	d *ArbosInitializationInfo
}

func NewMemoryInitDataReader(data *ArbosInitializationInfo) InitDataReader {
	return &MemoryInitDataReader{
		d: data,
	}
}

type FieldReader struct {
	m      *MemoryInitDataReader
	count  int
	length int
}

func (f *FieldReader) More() bool {
	return f.count < f.length
}

func (f *FieldReader) Close() error {
	f.count = f.length
	return nil
}

type MemoryStoredBlockReader struct {
	FieldReader
}

func (r *MemoryStoredBlockReader) GetNext() (*StoredBlock, error) {
	if !r.More() {
		return nil, errNoMore
	}
	r.count++
	return &r.m.d.Blocks[r.count-1], nil
}

func (m *MemoryInitDataReader) GetStoredBlockReader() (StoredBlockReader, error) {
	return &MemoryStoredBlockReader{
		FieldReader: FieldReader{
			m:      m,
			length: len(m.d.Blocks),
		},
	}, nil
}

type MemoryRetriableDataReader struct {
	FieldReader
}

func (r *MemoryRetriableDataReader) GetNext() (*InitializationDataForRetryable, error) {
	if !r.More() {
		return nil, errNoMore
	}
	r.count++
	return &r.m.d.RetryableData[r.count-1], nil
}

func (m *MemoryInitDataReader) GetRetriableDataReader() (RetriableDataReader, error) {
	return &MemoryRetriableDataReader{
		FieldReader: FieldReader{
			m:      m,
			length: len(m.d.RetryableData),
		},
	}, nil
}

type MemoryAddressReader struct {
	FieldReader
}

func (r *MemoryAddressReader) GetNext() (*common.Address, error) {
	if !r.More() {
		return nil, errNoMore
	}
	r.count++
	return &r.m.d.AddressTableContents[r.count-1], nil
}

func (m *MemoryInitDataReader) GetAddressTableReader() (AddressReader, error) {
	return &MemoryAddressReader{
		FieldReader: FieldReader{
			m:      m,
			length: len(m.d.AddressTableContents),
		},
	}, nil
}

type MemoryAccountDataReaderr struct {
	FieldReader
}

func (r *MemoryAccountDataReaderr) GetNext() (*AccountInitializationInfo, error) {
	if !r.More() {
		return nil, errNoMore
	}
	r.count++
	return &r.m.d.Accounts[r.count-1], nil
}

func (m *MemoryInitDataReader) GetAccountDataReader() (AccountDataReader, error) {
	return &MemoryAccountDataReaderr{
		FieldReader: FieldReader{
			m:      m,
			length: len(m.d.Accounts),
		},
	}, nil
}

func (m *MemoryInitDataReader) Close() error {
	return nil
}
