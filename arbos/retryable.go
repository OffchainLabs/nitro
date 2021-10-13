package arbos

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"io"
	"math/big"
)

type Retryable struct {
	id            common.Hash    // the retryable's ID is also the offset where its segment lives in storage
	numTries      *big.Int
	timeout       *big.Int
	from          common.Address
	to            common.Address
	callvalue     *big.Int
	calldata      []byte
}

func CreateRetryable(
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
		id,
		big.NewInt(0),
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

	// set up a segment to hold the retryable
	_ = state.AllocateSegmentAtOffsetForBytes(buf.Bytes(), id)

	// insert the new retryable into the queue so it can be reaped later
	state.RetryableQueue().Put(id)

	return ret
}

func OpenRetryable(state *ArbosState, id common.Hash) *Retryable {
	seg := state.OpenSegment(id)
	if seg == nil {
		// retryable has been deleted
		return nil
	}
	contents := seg.GetBytes()
	ret, err := NewFromReader(bytes.NewReader(contents), id)
	if err != nil {
		panic(err)
	}
	if ret.timeout.Cmp(state.LastTimestampSeen()) < 0 {
		// retryable has expired, so delete it
		seg.Delete()
		return nil
	}
	return ret
}

func NewFromReader(rd io.Reader, id common.Hash) (*Retryable, error) {
	numTries, err := HashFromReader(rd)
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
		id,
		numTries.Big(),
		timeout.Big(),
		from,
		to,
		callvalue.Big(),
		calldata,
	}, nil
}

func (retryable *Retryable) serialize(wr io.Writer) error {
	if _, err := wr.Write(common.BigToHash(retryable.numTries).Bytes()); err != nil {
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
		id := queue.Get()
		retryable := OpenRetryable(state, *id)
		if retryable != nil {
			// OpenRetryable returned non-nil, so we know the retryable hasn't expired
			queue.Put(*id)
		}
	}
}
