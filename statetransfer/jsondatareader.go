// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package statetransfer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
)

type ArbosInitFileContents struct {
	BlocksPath               string
	AddressTableContentsPath string
	RetryableDataPath        string
	AccountsPath             string
}

type JsonInitDataReader struct {
	basePath string
	data     ArbosInitFileContents
}

type JsonListReader struct {
	input *json.Decoder
	file  *os.File
}

func (i *JsonListReader) eatInputDelim(expected json.Delim) error {
	token, err := i.input.Token()
	if err != nil {
		return err
	}
	foundString := "<Not Delim>"
	delim, match := token.(json.Delim)
	if match {
		foundString = delim.String()
	}
	if foundString != expected.String() {
		return fmt.Errorf("expected %s, found: %s", expected, foundString)
	}
	return nil
}

func (l *JsonListReader) open() error {
	if l.input == nil {
		return nil
	}
	return l.eatInputDelim('[')
}

func (l *JsonListReader) More() bool {
	if l.input == nil {
		return false
	}
	return l.input.More()
}

func (l *JsonListReader) Close() error {
	l.input = nil
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return err
		}
		l.file = nil
	}
	return nil
}

func (m *JsonInitDataReader) getListReader(fileName string) (JsonListReader, error) {
	if fileName == "" {
		return JsonListReader{}, nil
	}
	filePath := path.Join(m.basePath, fileName)
	inboundFile, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return JsonListReader{}, err
	}
	res := JsonListReader{
		file:  inboundFile,
		input: json.NewDecoder(inboundFile),
	}
	if err := res.open(); err != nil {
		return JsonListReader{}, err
	}
	return res, nil
}

func NewJsonInitDataReader(filepath string) (InitDataReader, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	reader := JsonInitDataReader{
		basePath: path.Dir(filepath),
	}
	if err := json.Unmarshal(data, &reader.data); err != nil {
		return nil, err
	}
	return &reader, nil
}

func (m *JsonInitDataReader) Close() error {
	return nil
}

type JsonStoredBlockReader struct {
	JsonListReader
}

func (r *JsonStoredBlockReader) GetNext() (*StoredBlock, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem StoredBlock
	if err := r.input.Decode(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (m *JsonInitDataReader) GetStoredBlockReader() (StoredBlockReader, error) {
	listreader, err := m.getListReader(m.data.BlocksPath)
	if err != nil {
		return nil, err
	}
	return &JsonStoredBlockReader{
		JsonListReader: listreader,
	}, nil
}

type JsonRetriableDataReader struct {
	JsonListReader
}

func (r *JsonRetriableDataReader) GetNext() (*InitializationDataForRetryable, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem InitializationDataForRetryable
	if err := r.input.Decode(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (m *JsonInitDataReader) GetRetriableDataReader() (RetriableDataReader, error) {
	listreader, err := m.getListReader(m.data.RetryableDataPath)
	if err != nil {
		return nil, err
	}
	return &JsonRetriableDataReader{
		JsonListReader: listreader,
	}, nil
}

type JsonAddressReader struct {
	JsonListReader
}

func (r *JsonAddressReader) GetNext() (*common.Address, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem common.Address
	if err := r.input.Decode(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (m *JsonInitDataReader) GetAddressTableReader() (AddressReader, error) {
	listreader, err := m.getListReader(m.data.AddressTableContentsPath)
	if err != nil {
		return nil, err
	}
	return &JsonAddressReader{
		JsonListReader: listreader,
	}, nil
}

type JsonAccountDataReaderr struct {
	JsonListReader
}

func (r *JsonAccountDataReaderr) GetNext() (*AccountInitializationInfo, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem AccountInitializationInfo
	if err := r.input.Decode(&elem); err != nil {
		return nil, err
	}
	return &elem, nil
}

func (m *JsonInitDataReader) GetAccountDataReader() (AccountDataReader, error) {
	listreader, err := m.getListReader(m.data.AccountsPath)
	if err != nil {
		return nil, err
	}
	return &JsonAccountDataReaderr{
		JsonListReader: listreader,
	}, nil
}
