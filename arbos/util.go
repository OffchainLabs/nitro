package arbos

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

func HashFromReader(rd io.Reader) (common.Hash, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(buf[:]), nil
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
	return common.BytesToAddress(buf[:]), nil
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

func BytestringFromReader(rd io.Reader) ([]byte, error) {
	size, err := Uint64FromReader(rd)
	if err != nil {
		return nil, err
	}
	if size > MaxL2MessageSize {
		return nil, errors.New("attempted to extract too large of a slice from reader")
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
