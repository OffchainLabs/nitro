package arbos

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
	storage *ArbosState,
	id common.Hash,
	timeout common.Hash,
	from common.Address,
	to common.Address,
	callvalue common.Hash,
	calldata []byte,
) (*Retryable, error) {
	ret := &Retryable{
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
	seg, err := storage.AllocateSegmentForBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	ret.storageOffset = seg.offset

	return ret, nil
}

func Open(storage *ArbosState, offset common.Hash) (*Retryable, error) {
	seg, err := storage.OpenSegment(offset)
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
	id, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	timeout, err := HashFromReader(rd)
	if err != nil {
		return nil, err
	}
	from, err := AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	to, err := AddressFromReader(rd)
	if err != nil {
		return nil, err
	}
	callvalue, err := HashFromReader(rd)
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

	return &Retryable{
		offset,
		id,
		timeout,
		from,
		to,
		callvalue,
		calldata,
	}, nil
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
