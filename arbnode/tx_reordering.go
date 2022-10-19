// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbnode

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/util/arbmath"
	"math"
	"math/big"
	"time"
)

var (
	errNoMoreItems     = errors.New("no more items available")
	errTimeout         = errors.New("no item available before timeout")
	zeroEdgeWeight     = edgeWeight{0, common.Big0}
	infiniteEdgeWeight = edgeWeight{math.MaxUint64, common.Big0}
)

type txReorderer struct {
	source              genericItemSource
	outChan             chan txQueueItem
	reorderWindowMillis uint64
	nextUniqueId        uint64
	bufferedItem        *reorderItem
}

func newTxReorderer(source genericItemSource, reorderWindowMillis int, queueSize int) *txReorderer {
	return &txReorderer{
		source:              source,
		outChan:             make(chan txQueueItem, queueSize),
		reorderWindowMillis: uint64(reorderWindowMillis),
		nextUniqueId:        0,
		bufferedItem:        nil,
	}
}

type reorderItem struct {
	uniqueId       uint64
	timestamp      uint64
	bid            *big.Int
	cumulativeLoss *big.Int
	queueItem      *txQueueItem
}

func (reo *txReorderer) run(ctx context.Context) {
	defer close(reo.outChan)

	pendingTxs := []reorderItem{}
	visibilityWindowMillis := 2 * reo.reorderWindowMillis

	for {
		// read in txs up until earliest plus window
		timeLimit := uint64(math.MaxUint64)
		if len(pendingTxs) > 0 {
			timeLimit = arbmath.SaturatingUAdd(pendingTxs[0].timestamp, visibilityWindowMillis)
		}
		doneCollectingInputs := false
		for !doneCollectingInputs {
			item, err := reo.get(ctx, timeLimit)
			if errors.Is(err, errNoMoreItems) {
				doneCollectingInputs = true
				if len(pendingTxs) == 0 {
					return
				}
			} else if errors.Is(err, errTimeout) {
				doneCollectingInputs = true
			} else {
				if err != nil {
					return
				}
				if item.timestamp <= timeLimit {
					pendingTxs = append(pendingTxs, item)
					if len(pendingTxs) == 1 {
						timeLimit = arbmath.SaturatingUAdd(pendingTxs[0].timestamp, visibilityWindowMillis)
					}
				} else {
					reo.pushBack(item)
					doneCollectingInputs = true
				}
			}
		}

		sequenced, ptx := reorder_minimax(pendingTxs, reo.reorderWindowMillis)
		pendingTxs = ptx

		for _, tx := range sequenced {
			// emit the tx
			select {
			case reo.outChan <- *tx.queueItem:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (reo *txReorderer) get(ctx context.Context, timeoutMillis uint64) (reorderItem, error) {
	if reo.bufferedItem != nil {
		ret := *reo.bufferedItem
		reo.bufferedItem = nil
		return ret, nil
	}
	return reo.source.Get(ctx, timeoutMillis)
}

func (reo *txReorderer) pushBack(item reorderItem) {
	buffer := item
	reo.bufferedItem = &buffer
}

func reorder_minimax(pendingTxs []reorderItem, reorderWindow uint64) ([]reorderItem, []reorderItem) {
	numVertices := uint64(len(pendingTxs))
	removalOrder := findRemovals(pendingTxs, reorderWindow)
	emitted := []reorderItem{}
	remaining := []reorderItem{}
	wasEmitted := make([]bool, numVertices)

	done := false
	for i := len(removalOrder) - 1; !done && i >= 0; i-- {
		wasEmitted[removalOrder[i]] = true
		emitted = append(emitted, pendingTxs[removalOrder[i]])
		if removalOrder[i] == 0 {
			done = true
		}
	}

	for i := uint64(0); i < numVertices; i++ {
		localLoss := common.Big0
		if !wasEmitted[i] {
			for j := uint64(0); j < numVertices; j++ {
				if wasEmitted[j] {
					dir, heavy, weight := computeEdgeAndDirection(pendingTxs[i], pendingTxs[j], reorderWindow)
					if dir == Direction(ForwardDirection) && !heavy {
						localLoss = new(big.Int).Add(localLoss, weight)
					}
				}
			}
			if pendingTxs[i].cumulativeLoss == nil {
				pendingTxs[i].cumulativeLoss = common.Big0
			}
			pendingTxs[i].cumulativeLoss = new(big.Int).Add(pendingTxs[i].cumulativeLoss, localLoss)
			remaining = append(remaining, pendingTxs[i])
		}
	}
	return emitted, remaining
}

type edge struct {
	from   uint64
	to     uint64
	weight edgeWeight
}

type edgeWeight struct {
	hardEdges  uint64
	softWeight *big.Int
}

func newHardEdge() edgeWeight {
	return edgeWeight{
		hardEdges:  1,
		softWeight: common.Big0,
	}
}

func newSoftEdge(weight *big.Int) edgeWeight {
	return edgeWeight{
		hardEdges:  0,
		softWeight: weight,
	}
}

func (ew edgeWeight) Cmp(other edgeWeight) int {
	if ew.hardEdges > other.hardEdges {
		return 1
	}
	if ew.hardEdges < other.hardEdges {
		return -1
	}
	return ew.softWeight.Cmp(other.softWeight)
}

func (ew edgeWeight) Add(other edgeWeight) edgeWeight {
	return edgeWeight{
		hardEdges:  ew.hardEdges + other.hardEdges,
		softWeight: new(big.Int).Add(ew.softWeight, other.softWeight),
	}
}

func (ew edgeWeight) Sub(other edgeWeight) edgeWeight {
	return edgeWeight{
		hardEdges:  ew.hardEdges - other.hardEdges,
		softWeight: new(big.Int).Sub(ew.softWeight, other.softWeight),
	}
}

func findRemovals(pendingTxs []reorderItem, reorderWindow uint64) []uint64 {
	numVertices := uint64(len(pendingTxs))
	removed := make([]bool, numVertices)
	removalOrder := []uint64{}
	edges := generateEdges_altRules(pendingTxs, reorderWindow)

	// compute outgoing weights
	outWeight := make([]edgeWeight, numVertices)
	for i, tx := range pendingTxs {
		if tx.cumulativeLoss != nil {
			outWeight[i] = newSoftEdge(tx.cumulativeLoss)
		} else {
			outWeight[i] = zeroEdgeWeight
		}
	}
	for _, edge := range edges {
		outWeight[edge.from] = outWeight[edge.from].Add(edge.weight)
	}

	for num := numVertices; num > 0; num-- {
		idx := uint64(0)
		minOutWeight := infiniteEdgeWeight
		for i := uint64(0); i < numVertices; i++ {
			if !removed[i] && outWeight[i].Cmp(minOutWeight) < 0 {
				idx = i
				minOutWeight = outWeight[i]
			}
		}
		removed[idx] = true
		removalOrder = append(removalOrder, idx)
		edges = removeEdgesForVertex(edges, idx, outWeight)
	}
	return removalOrder
}

func generateEdges_altRules(pendingTxs []reorderItem, reorderWindow uint64) []*edge {
	edges := []*edge{}
	for i := uint64(0); i < uint64(len(pendingTxs)); i++ {
		for j := uint64(0); j < i; j++ {
			edges = append(edges, generateEdgeInList(pendingTxs[j], pendingTxs[i], j, i, reorderWindow))
		}
	}
	return edges
}

func removeEdgesForVertex(edges []*edge, whichVertex uint64, outDegrees []edgeWeight) []*edge {
	ret := []*edge{}
	for _, e := range edges {
		if e.from != whichVertex && e.to != whichVertex {
			ret = append(ret, e)
		} else {
			outDegrees[e.from] = outDegrees[e.from].Sub(e.weight)
		}
	}
	return ret
}

const (
	ForwardDirection uint8 = iota
	BackwardDirection
)

type Direction uint8

func computeEdgeAndDirection(from reorderItem, to reorderItem, reorderWindow uint64) (direction Direction, heavy bool, weight *big.Int) {
	if arbmath.SaturatingUAdd(from.timestamp, reorderWindow) < to.timestamp {
		return Direction(ForwardDirection), true, common.Big0
	} else if arbmath.SaturatingUAdd(to.timestamp, reorderWindow) < from.timestamp {
		return Direction(BackwardDirection), true, common.Big0
	} else {
		cmpBids := from.bid.Cmp(to.bid)
		if cmpBids > 0 {
			return Direction(ForwardDirection), false, new(big.Int).Sub(from.bid, to.bid)
		} else if cmpBids < 0 {
			return Direction(BackwardDirection), false, new(big.Int).Sub(to.bid, from.bid)
		} else if from.timestamp < to.timestamp {
			return Direction(ForwardDirection), false, common.Big1
		} else if to.timestamp < from.timestamp {
			return Direction(BackwardDirection), false, common.Big1
		} else {
			// this should never happen
			return Direction(ForwardDirection), false, common.Big0
		}
	}
}

func generateEdgeInList(fromTx, toTx reorderItem, fromIndex, toIndex uint64, reorderWindow uint64) *edge {
	dir, heavy, weight := computeEdgeAndDirection(fromTx, toTx, reorderWindow)
	if dir == Direction(BackwardDirection) {
		fromIndex, toIndex = toIndex, fromIndex
	}
	if heavy {
		return &edge{fromIndex, toIndex, newHardEdge()}
	} else {
		return &edge{fromIndex, toIndex, newSoftEdge(weight)}
	}
}

type genericItemSource interface {
	Get(context.Context, uint64) (reorderItem, error)
}

type itemSourceChan struct {
	in           <-chan txQueueItem
	nextUniqueId uint64
}

func newItemSourceFromChan(in chan txQueueItem) genericItemSource {
	return &itemSourceChan{in, 0}
}

func (src *itemSourceChan) Get(ctx context.Context, timeoutMillis uint64) (reorderItem, error) {
	now := time.Now()
	timeoutMillisAsInt64 := int64(timeoutMillis)
	if timeoutMillisAsInt64 < 0 {
		timeoutMillisAsInt64 = math.MaxInt64
	}
	deadline := time.UnixMilli(timeoutMillisAsInt64)
	if now.After(deadline) {
		return reorderItem{}, errTimeout
	}
	timeout := time.NewTimer(deadline.Sub(now))
	defer timeout.Stop()

	select {
	case item, ok := <-src.in:
		if !ok {
			return reorderItem{}, errNoMoreItems
		}
		ret := reorderItem{
			uniqueId:       0,
			timestamp:      uint64(time.Now().UnixMilli()),
			bid:            item.tx.GasTipCap(),
			cumulativeLoss: common.Big0,
			queueItem:      &item,
		}
		return ret, nil
	case <-timeout.C:
		return reorderItem{}, errTimeout
	case <-ctx.Done():
		return reorderItem{}, ctx.Err()
	}
}

type itemSourceSlice struct {
	remainingTxs []reorderItem
}

func newItemSourceSlice(txs []reorderItem) genericItemSource {
	return &itemSourceSlice{txs}
}

func (p *itemSourceSlice) Get(ctx context.Context, timeoutMillis uint64) (reorderItem, error) {
	if len(p.remainingTxs) == 0 {
		return reorderItem{}, errNoMoreItems
	}
	ret := p.remainingTxs[0]
	p.remainingTxs = p.remainingTxs[1:]
	return ret, nil
}
