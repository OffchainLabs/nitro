//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package util

import (
	"encoding/binary"
	"errors"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var AddressAliasOffset *big.Int
var InverseAddressAliasOffset *big.Int

func init() {
	offset, success := new(big.Int).SetString("0x1111000000000000000000000000000000001111", 0)
	if !success {
		panic("Error initializing AddressAliasOffset")
	}
	AddressAliasOffset = offset
	InverseAddressAliasOffset = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), AddressAliasOffset)
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

func DoesTxTypeAlias(txType byte) bool {
	switch txType {
	case types.ArbitrumUnsignedTxType:
	case types.ArbitrumContractTxType:
	case types.ArbitrumRetryTxType:
		return true
	}
	return false
}
