package arbos

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"math/big"
)

type Retryable struct {
	storageOffset common.Hash
	id            common.Hash
	timeout       *big.Int
	from          common.Address
	to            common.Address
	callvalue     *big.Int
	calldata      []byte
}

func Create(
	state *ArbosState,
	id common.Hash,
	timeout *big.Int,
	from common.Address,
	to common.Address,
	callvalue *big.Int,
	calldata []byte,
) *Retryable {
	state.ReapRetryableQueue()
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
		panic(err)
	}
	seg := state.AllocateSegmentForBytes(buf.Bytes())
	ret.storageOffset = seg.offset

	return ret
}

func OpenRetryable(storage *ArbosState, offset common.Hash) *Retryable {
	seg := storage.OpenSegment(offset)
	contents := seg.GetBytes()
	ret, err := NewFromReader(bytes.NewReader(contents), offset)
	if err != nil {
		panic(err)
	}
	return ret
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
		timeout.Big(),
		from,
		to,
		callvalue.Big(),
		calldata,
	}, nil
}

func (retryable *Retryable) serialize(wr io.Writer) error {
	if _, err := wr.Write(retryable.id[:]); err != nil {
		return err
	}
	if _, err := wr.Write(common.BigToHash(retryable.timeout).Bytes()); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.from[:]); err != nil {
		return err
	}
	if _, err := wr.Write(retryable.to[:]); err != nil {
		return err
	}
	if _, err := wr.Write(common.BigToHash(retryable.callvalue).Bytes()); err != nil {
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

func (state *ArbosState) ReapRetryableQueue() {
	queue := state.RetryableQueue()
	if !queue.IsEmpty() {
		offset := queue.Get()
		retryable := OpenRetryable(state, *offset)
		if retryable.timeout.Cmp(state.LastTimestampSeen()) < 0 {
			segment := state.OpenSegment(*offset)
			segment.Delete()
		} else {
			queue.Put(*offset)
		}
	}
}
