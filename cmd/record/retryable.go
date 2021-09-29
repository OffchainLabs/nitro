package main

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"io"
)

type Retryable struct {
	storageOffset common.Hash
	id            common.Hash
	timeout       common.Hash
	from          common.Address
	to            common.Address
	callvalue     common.Hash
	calldata      []byte
}

func Create(
	storage *ArbosStorage,
	id common.Hash,
	timeout common.Hash,
	from common.Address,
	to common.Address,
	callvalue common.Hash,
	calldata []byte,
) (*Retryable, error) {
	ret := &Retryable {
		common.Hash{},   // will fill in later
		id,
		timeout,
		from,
		to,
		callvalue,
		calldata,
	};
	buf := bytes.Buffer{}
	if err := ret.serialize(&buf); err != nil {
		return nil, err
	}
	seg, err := storage.AllocateForBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	ret.storageOffset = seg.offset

	return ret, nil
}

func Open(storage *ArbosStorage, offset common.Hash) (*Retryable, error) {
	seg, err := storage.Open(offset)
	if err != nil {
		return nil, err
	}
	contents, err := seg.GetBytes()
	if err != nil {
		return nil, err
	}
	return NewFromReader(bytes.NewReader(contents), offset)
}

func NewFromReader(rd io.Reader, offset common.Hash) (*Retryable, error) {
	id, err := hashFromReader(rd)
	if err != nil {
		return nil, err
	}
	timeout, err := hashFromReader(rd)
	if err != nil {
		return nil, err
	}
	from, err := addressFromReader(rd)
	if err != nil {
		return nil, err
	}
	to, err := addressFromReader(rd)
	if err != nil {
		return nil, err
	}
	callvalue, err := hashFromReader(rd)
	if err != nil {
		return nil, err
	}
	sizeBuf := make([]byte, 8)
	if _, err := io.ReadFull(rd, sizeBuf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint64(sizeBuf)
	calldata := make([]byte, size)
	if _, err := io.ReadFull(rd, calldata); err != nil {
		return nil, err
	}

	return &Retryable {
		offset,
		id,
		timeout,
		from,
		to,
		callvalue,
		calldata,
	}, nil
}

func hashFromReader(rd io.Reader) (common.Hash, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(buf[:]), nil
}


func addressFromReader(rd io.Reader) (common.Address, error) {
	buf := make([]byte, 20)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(buf[:]), nil
}

func (retryable *Retryable) serialize(wr io.Writer) error {
	if _, err := wr.Write(retryable.id[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.timeout[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.from[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.to[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.callvalue[:]); err != nil {
		return err
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(len(retryable.calldata)))
	if _, err := wr.Write(b); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.calldata); err != nil {
		return err
	}
	return nil
}
