// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

package das

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/offchainlabs/nitro/arbstate"
)

var ErrNoReadersResponded = errors.New("no DAS readers responded successfully")

type aggregatorStrategy interface {
	newInstance() aggregatorStrategyInstance
	update([]arbstate.DataAvailabilityReader, map[arbstate.DataAvailabilityReader]readerStats)
}

type abstractAggregatorStrategy struct {
	sync.RWMutex
	readers []arbstate.DataAvailabilityReader
	stats   map[arbstate.DataAvailabilityReader]readerStats
}

func (s *abstractAggregatorStrategy) update(readers []arbstate.DataAvailabilityReader, stats map[arbstate.DataAvailabilityReader]readerStats) {
	s.Lock()
	defer s.Unlock()

	s.readers = make([]arbstate.DataAvailabilityReader, len(readers))
	copy(s.readers, readers)

	s.stats = make(map[arbstate.DataAvailabilityReader]readerStats)
	for k, v := range stats {
		s.stats[k] = v
	}
}

// Exponentially growing Explore Exploit Strategy
type simpleExploreExploitStrategy struct {
	iterations        uint32
	exploreIterations uint32
	exploitIterations uint32

	abstractAggregatorStrategy
}

func (s *simpleExploreExploitStrategy) newInstance() aggregatorStrategyInstance {
	iterations := atomic.AddUint32(&s.iterations, 1)

	readerSets := make([][]arbstate.DataAvailabilityReader, 0)
	s.RLock()
	defer s.RUnlock()

	readers := make([]arbstate.DataAvailabilityReader, len(s.readers))
	copy(readers, s.readers)

	if iterations%(s.exploreIterations+s.exploitIterations) < s.exploreIterations {
		// Explore phase
		rand.Shuffle(len(readers), func(i, j int) { readers[i], readers[j] = readers[j], readers[i] })
	} else {
		// Exploit phase
		sort.Slice(readers, func(i, j int) bool {
			a, b := s.stats[readers[i]], s.stats[readers[j]]
			return a.successRatioWeightedMeanLatency() < b.successRatioWeightedMeanLatency()
		})
	}

	for i, maxTake := 0, 1; i < len(readers); maxTake = maxTake * 2 {
		readerSet := make([]arbstate.DataAvailabilityReader, 0, maxTake)
		for taken := 0; taken < maxTake && i < len(readers); i, taken = i+1, taken+1 {
			readerSet = append(readerSet, readers[i])
		}
		readerSets = append(readerSets, readerSet)
	}

	return &basicStrategyInstance{readerSets: readerSets}
}

// Sequential Strategy for Testing
type testingSequentialStrategy struct {
	abstractAggregatorStrategy
}

func (s *testingSequentialStrategy) newInstance() aggregatorStrategyInstance {
	s.RLock()
	defer s.RUnlock()

	si := basicStrategyInstance{}
	for _, reader := range s.readers {
		si.readerSets = append(si.readerSets, []arbstate.DataAvailabilityReader{reader})
	}

	return &si
}

// Instance of a strategy that returns readers in an order according to the strategy
type aggregatorStrategyInstance interface {
	nextReaders() []arbstate.DataAvailabilityReader
}

type basicStrategyInstance struct {
	readerSets [][]arbstate.DataAvailabilityReader
}

func (si *basicStrategyInstance) nextReaders() []arbstate.DataAvailabilityReader {
	if len(si.readerSets) == 0 {
		return nil
	}
	next := si.readerSets[0]
	si.readerSets = si.readerSets[1:]
	return next
}
