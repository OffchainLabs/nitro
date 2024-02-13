// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package statetransfer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
)

type ArbosInitFileContents struct {
	NextBlockNumber          uint64 `json:"NextBlockNumber"`
	AddressTableContentsPath string `json:"AddressTableContentsPath"`
	RetryableDataPath        string `json:"RetryableDataPath"`
	AccountsPath             string `json:"AccountsPath"`
}

type JsonInitDataReader struct {
	basePath string
	data     ArbosInitFileContents
}

func (r *JsonInitDataReader) GetNextBlockNumber() (uint64, error) {
	return r.data.NextBlockNumber, nil
}

type JsonListReader struct {
	input *json.Decoder
	file  *os.File
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

func (r *JsonInitDataReader) getListReader(fileName string) (JsonListReader, error) {
	if fileName == "" {
		return JsonListReader{}, nil
	}
	filePath := path.Join(r.basePath, fileName)
	inboundFile, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return JsonListReader{}, err
	}
	res := JsonListReader{
		file:  inboundFile,
		input: json.NewDecoder(inboundFile),
	}
	return res, nil
}

func NewJsonInitDataReader(filepath string) (InitDataReader, error) {
	data, err := os.ReadFile(filepath)
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

func (r *JsonInitDataReader) Close() error {
	return nil
}

type JsonRetryableDataReader struct {
	JsonListReader
}

type InitializationDataForRetryableJson struct {
	Id          common.Hash
	Timeout     uint64
	From        common.Address
	To          common.Address
	Callvalue   string
	Beneficiary common.Address
	Calldata    []byte
}

func stringToBig(input string) (*big.Int, error) {
	output, success := new(big.Int).SetString(input, 0)
	if !success {
		return nil, fmt.Errorf("%s cannot be parsed as big.Int", input)
	}
	return output, nil
}

func (r *JsonRetryableDataReader) GetNext() (*InitializationDataForRetryable, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem InitializationDataForRetryableJson
	if err := r.input.Decode(&elem); err != nil {
		return nil, fmt.Errorf("decoding retryable: %w", err)
	}
	callValueBig, err := stringToBig(elem.Callvalue)
	if err != nil {
		return nil, err
	}
	return &InitializationDataForRetryable{
		Id:          elem.Id,
		Timeout:     elem.Timeout,
		From:        elem.From,
		To:          elem.To,
		Callvalue:   callValueBig,
		Beneficiary: elem.Beneficiary,
		Calldata:    elem.Calldata,
	}, nil
}

func (r *JsonInitDataReader) GetRetryableDataReader() (RetryableDataReader, error) {
	listreader, err := r.getListReader(r.data.RetryableDataPath)
	if err != nil {
		return nil, err
	}
	return &JsonRetryableDataReader{
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

func (r *JsonInitDataReader) GetAddressTableReader() (AddressReader, error) {
	listreader, err := r.getListReader(r.data.AddressTableContentsPath)
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

type AccountInitializationInfoJson struct {
	Addr         common.Address
	Nonce        uint64
	Balance      string
	ContractInfo *AccountInitContractInfo
	ClassicHash  common.Hash
}

func (r *JsonAccountDataReaderr) GetNext() (*AccountInitializationInfo, error) {
	if !r.More() {
		return nil, errNoMore
	}
	var elem AccountInitializationInfoJson
	if err := r.input.Decode(&elem); err != nil {
		return nil, err
	}
	balanceBig, err := stringToBig(elem.Balance)
	if err != nil {
		return nil, err
	}
	return &AccountInitializationInfo{
		Addr:            elem.Addr,
		Nonce:           elem.Nonce,
		EthBalance:      balanceBig,
		ContractInfo:    elem.ContractInfo,
		AggregatorInfo:  nil,
		AggregatorToPay: nil,
		ClassicHash:     elem.ClassicHash,
	}, nil
}

func (r *JsonInitDataReader) GetAccountDataReader() (AccountDataReader, error) {
	listreader, err := r.getListReader(r.data.AccountsPath)
	if err != nil {
		return nil, err
	}
	return &JsonAccountDataReaderr{
		JsonListReader: listreader,
	}, nil
}
