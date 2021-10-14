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
	state.TryToReapOneRetryable()
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

	// mark the new retryable as valid
	state.ValidRetryablesSet().Set(id, common.Hash{ 1 })

	return ret
}

func OpenRetryable(state *ArbosState, id common.Hash) *Retryable {
	if state.ValidRetryablesSet().Get(id) == (common.Hash{}) {
		// that is not a valid retryable
		return nil
	}
	seg := state.OpenSegment(id)
	if seg == nil {
		// retryable has been deleted
		return nil
	}
	contents := seg.GetBytes()
	ret, err := NewRetryableFromReader(bytes.NewReader(contents), id)
	if err != nil {
		panic(err)
	}
	if ret.timeout.Cmp(state.LastTimestampSeen()) < 0 {
		// retryable has expired, so delete it
		seg.Delete()
		state.ValidRetryablesSet().Set(id, common.Hash{})
		return nil
	}
	return ret
}

func DeleteRetryable(state *ArbosState, id common.Hash) {
	vrs := state.ValidRetryablesSet()
	if vrs.Get(id) != (common.Hash{}) {
		vrs.Set(id, common.Hash{})
		seg := state.OpenSegment(id)
		if seg != nil {
			seg.Delete()
		}
	}
}

func NewRetryableFromReader(rd io.Reader, id common.Hash) (*Retryable, error) {
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

func (state *ArbosState) TryToReapOneRetryable() {
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

type PlannedRedeem struct {
	segment       *StorageSegment
	retryableId   common.Hash
	gasRefundAddr common.Address
	gas           uint64
	gasPrice      *big.Int
}

func NewPlannedRedeem(state *ArbosState, retryableId common.Hash, gasRefundAddr common.Address, gas uint64, gasPrice *big.Int) *PlannedRedeem {
	ret := &PlannedRedeem{
		nil,
		retryableId,
		gasRefundAddr,
		gas,
		gasPrice,
	}
	ret.segment = state.AllocateSegmentForBytes(ret.serialize())
	state.PendingRedeemQueue().Put(ret.segment.offset)
	return ret
}

func OpenPlannedRedeem(state *ArbosState, offset common.Hash) *PlannedRedeem {
	segment := state.OpenSegment(offset)
	buf := bytes.NewReader(segment.GetBytes())
	retryableId, err := HashFromReader(buf)
	if err != nil {
		return nil
	}
	gasRefundAddr, err := AddressFromReader(buf)
	if err != nil {
		return nil
	}
	gas, err := Uint64FromReader(buf)
	if err != nil {
		return nil
	}
	gasPrice, err := HashFromReader(buf)
	if err != nil {
		return nil
	}

	return &PlannedRedeem{
		segment,
		retryableId,
		gasRefundAddr,
		gas,
		gasPrice.Big(),
	}
}

func (pr *PlannedRedeem) serialize() []byte {
	var buf bytes.Buffer
	if err := HashToWriter(pr.retryableId, &buf); err != nil {
		return nil
	}
	if err := AddressToWriter(pr.gasRefundAddr, &buf); err != nil {
		return nil
	}
	if err := Uint64ToWriter(pr.gas, &buf); err != nil {
		return nil
	}
	if err := HashToWriter(common.BigToHash(pr.gasPrice), &buf); err != nil {
		return nil
	}
	return buf.Bytes()
}

func PeekNextPendingRedeem(state *ArbosState) *PlannedRedeem {  // peek at the next redeem but don't consume it
	queue := state.PendingRedeemQueue()
	if queue.IsEmpty() {
		return nil
	}
	return OpenPlannedRedeem(state, *queue.Peek())
}

func DiscardNextPendingRedeem(state *ArbosState, deleteTheRetryable bool) {
	queue := state.PendingRedeemQueue()
	if queue.IsEmpty() {
		return
	}
	redeem := OpenPlannedRedeem(state, *queue.Get())
	if deleteTheRetryable {
		DeleteRetryable(state, redeem.retryableId)
	}
	redeem.segment.Delete()
}



