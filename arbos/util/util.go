// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package util

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var AddressAliasOffset *big.Int
var InverseAddressAliasOffset *big.Int
var ParseRedeemScheduledLog func(interface{}, *types.Log) error
var ParseL2ToL1TransactionLog func(interface{}, *types.Log) error
var ParseL2ToL1TxLog func(interface{}, *types.Log) error
var PackInternalTxDataStartBlock func(...interface{}) ([]byte, error)
var UnpackInternalTxDataStartBlock func([]byte) (map[string]interface{}, error)
var PackInternalTxDataBatchPostingReport func(...interface{}) ([]byte, error)
var UnpackInternalTxDataBatchPostingReport func([]byte) (map[string]interface{}, error)
var PackArbRetryableTxRedeem func(...interface{}) ([]byte, error)

func init() {
	offset, success := new(big.Int).SetString("0x1111000000000000000000000000000000001111", 0)
	if !success {
		panic("Error initializing AddressAliasOffset")
	}
	AddressAliasOffset = offset
	InverseAddressAliasOffset = arbmath.BigSub(new(big.Int).Lsh(big.NewInt(1), 160), AddressAliasOffset)

	// Create a mechanism for parsing event logs
	logParser := func(source string, name string) func(interface{}, *types.Log) error {
		precompile, err := abi.JSON(strings.NewReader(source))
		if err != nil {
			panic(fmt.Sprintf("failed to parse ABI for %s: %s", name, err))
		}
		inputs := precompile.Events[name].Inputs
		indexed := abi.Arguments{}
		for _, input := range inputs {
			if input.Indexed {
				indexed = append(indexed, input)
			}
		}

		return func(event interface{}, log *types.Log) error {
			unpacked, err := inputs.Unpack(log.Data)
			if err != nil {
				return err
			}
			if err := inputs.Copy(event, unpacked); err != nil {
				return err
			}
			return abi.ParseTopics(event, indexed, log.Topics[1:])
		}
	}

	ParseRedeemScheduledLog = logParser(precompilesgen.ArbRetryableTxABI, "RedeemScheduled")
	ParseL2ToL1TxLog = logParser(precompilesgen.ArbSysABI, "L2ToL1Tx")
	ParseL2ToL1TransactionLog = logParser(precompilesgen.ArbSysABI, "L2ToL1Transaction")

	acts := precompilesgen.ArbosActsABI
	PackInternalTxDataStartBlock, UnpackInternalTxDataStartBlock = NewCallParser(acts, "startBlock")
	PackInternalTxDataBatchPostingReport, UnpackInternalTxDataBatchPostingReport = NewCallParser(acts, "batchPostingReport")
	PackArbRetryableTxRedeem, _ = NewCallParser(precompilesgen.ArbRetryableTxABI, "redeem")
}

// Create a mechanism for packing and unpacking calls
func NewCallParser(source string, name string) (func(...interface{}) ([]byte, error), func([]byte) (map[string]interface{}, error)) {
	contract, err := abi.JSON(strings.NewReader(source))
	if err != nil {
		panic(fmt.Sprintf("failed to parse ABI for %s: %s", name, err))
	}
	method, ok := contract.Methods[name]
	if !ok {
		panic(fmt.Sprintf("method %v does not exist", name))
	}
	pack := func(args ...interface{}) ([]byte, error) {
		return contract.Pack(name, args...)
	}
	unpack := func(data []byte) (map[string]interface{}, error) {
		if len(data) < 4 {
			return nil, errors.New("data not long enough")
		}
		args := make(map[string]interface{})
		return args, method.Inputs.UnpackIntoMap(args, data[4:])
	}
	return pack, unpack
}

func AddressToHash(address common.Address) common.Hash {
	return common.BytesToHash(address.Bytes())
}

func HashFromReader(rd io.Reader) (common.Hash, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(buf), nil
}

func HashToWriter(val common.Hash, wr io.Writer) error {
	_, err := wr.Write(val.Bytes())
	return err
}

func AddressFromReader(rd io.Reader) (common.Address, error) {
	buf := make([]byte, 20)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(buf), nil
}

func AddressFrom256FromReader(rd io.Reader) (common.Address, error) {
	h, err := HashFromReader(rd)
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(h.Bytes()[12:]), nil
}

func AddressToWriter(val common.Address, wr io.Writer) error {
	_, err := wr.Write(val.Bytes())
	return err
}

func AddressTo256ToWriter(val common.Address, wr io.Writer) error {
	if _, err := wr.Write(make([]byte, 12)); err != nil {
		return err
	}
	return AddressToWriter(val, wr)
}

func Uint64FromReader(rd io.Reader) (uint64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf), nil
}

func Uint64ToWriter(val uint64, wr io.Writer) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], val)
	_, err := wr.Write(buf[:])
	return err
}

func BytestringFromReader(rd io.Reader, maxBytesToRead uint64) ([]byte, error) {
	size, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	if size > maxBytesToRead {
		return nil, errors.New("size too large in ByteStringFromReader")
	}
	buf := make([]byte, size)
	if _, err = io.ReadFull(rd, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func BytestringToWriter(val []byte, wr io.Writer) error {
	if err := Uint64ToWriter(uint64(len(val)), wr); err != nil {
		return err
	}
	_, err := wr.Write(val)
	return err
}

func IntToHash(val int64) common.Hash {
	return common.BigToHash(big.NewInt(val))
}

func UintToHash(val uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(val))
}

func HashPlusInt(x common.Hash, y int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(x.Big(), big.NewInt(y))) //BUGBUG: BigToHash(x) converts abs(x) to a Hash
}

func RemapL1Address(l1Addr common.Address) common.Address {
	sumBytes := new(big.Int).Add(new(big.Int).SetBytes(l1Addr.Bytes()), AddressAliasOffset).Bytes()
	if len(sumBytes) > 20 {
		sumBytes = sumBytes[len(sumBytes)-20:]
	}
	return common.BytesToAddress(sumBytes)
}

func InverseRemapL1Address(l1Addr common.Address) common.Address {
	sumBytes := new(big.Int).Add(new(big.Int).SetBytes(l1Addr.Bytes()), InverseAddressAliasOffset).Bytes()
	if len(sumBytes) > 20 {
		sumBytes = sumBytes[len(sumBytes)-20:]
	}
	return common.BytesToAddress(sumBytes)
}

func DoesTxTypeAlias(txType *byte) bool {
	if txType == nil {
		return false
	}
	switch *txType {
	case types.ArbitrumUnsignedTxType:
		fallthrough
	case types.ArbitrumContractTxType:
		fallthrough
	case types.ArbitrumRetryTxType:
		return true
	}
	return false
}

func TxTypeHasPosterCosts(txType byte) bool {
	switch txType {
	case types.ArbitrumUnsignedTxType:
		fallthrough
	case types.ArbitrumContractTxType:
		fallthrough
	case types.ArbitrumRetryTxType:
		fallthrough
	case types.ArbitrumInternalTxType:
		fallthrough
	case types.ArbitrumSubmitRetryableTxType:
		return false
	}
	return true
}

func SafeMapGet[T any](kvs map[string]interface{}, field string) T {
	value, ok := kvs[field]
	if !ok {
		panic(fmt.Sprintf("map does not contain field %v", field))
	}
	cast, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("field %v is of the wrong type", field))
	}
	return cast
}
