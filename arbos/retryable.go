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
	timeout       common.Hash
	from          common.Address
	to            common.Address
	callvalue     common.Hash
	calldata      []byte
}

func (r *Retryable) Timeout() common.Hash {
	return r.timeout
}

func (r *Retryable) SetTimeout(t common.Hash) {
	r.timeout = t
}

func (r *Retryable) AddToTimeout(delta common.Hash) {
	r.timeout = common.BigToHash(new(big.Int).Add(r.timeout.Big(), delta.Big()))
}

func Create(
	state *ArbosState,
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
	seg, err := state.AllocateSegmentForBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	ret.storageOffset = seg.offset

	if err := state.retryableQueue.Put(ret.storageOffset); err != nil {
		return nil, err
	}

	return ret, nil
}

func OpenRetryable(storage *ArbosState, offset common.Hash) (*Retryable, error) {
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

func TryTrimOneRetryable(state *ArbosState) error {
	q := state.retryableQueue
	if ! q.IsEmpty() {
		headOffset, err := q.Get()
		if err != nil {
			return err
		}

		retryable, err := OpenRetryable(state, headOffset)
		if err != nil {
			return err
		}

		if retryable.timeout.Big().Cmp(state.lastTimestampSeen.Big()) < 0 {
			// retryable timed out, so delete it
			if _, err := q.Get(); err != nil {
				return err
			}
			seg, err := state.OpenSegment(retryable.storageOffset)
			if err != nil {
				return err
			}
			seg.Clear();
		} else {
			// retryable is still alive, put it at the end of the queue
			if err := q.Put(headOffset); err != nil {
				return err
			}
		}
	}

	return nil
}
