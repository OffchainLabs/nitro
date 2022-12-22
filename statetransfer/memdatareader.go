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

func (r *MemoryInitDataReader) GetNextBlockNumber() (uint64, error) {
	return r.d.NextBlockNumber, nil
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

type MemoryRetryableDataReader struct {
	FieldReader
}

func (r *MemoryRetryableDataReader) GetNext() (*InitializationDataForRetryable, error) {
	if !r.More() {
		return nil, errNoMore
	}
	r.count++
	return &r.m.d.RetryableData[r.count-1], nil
}

func (r *MemoryInitDataReader) GetRetryableDataReader() (RetryableDataReader, error) {
	return &MemoryRetryableDataReader{
		FieldReader: FieldReader{
			m:      r,
			length: len(r.d.RetryableData),
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

func (r *MemoryInitDataReader) GetAddressTableReader() (AddressReader, error) {
	return &MemoryAddressReader{
		FieldReader: FieldReader{
			m:      r,
			length: len(r.d.AddressTableContents),
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

func (r *MemoryInitDataReader) GetAccountDataReader() (AccountDataReader, error) {
	return &MemoryAccountDataReaderr{
		FieldReader: FieldReader{
			m:      r,
			length: len(r.d.Accounts),
		},
	}, nil
}

func (r *MemoryInitDataReader) Close() error {
	return nil
}
