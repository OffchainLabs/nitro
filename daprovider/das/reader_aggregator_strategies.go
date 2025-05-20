// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package das

import (
	"errors"
	"maps"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/offchainlabs/nitro/daprovider/das/dasutil"
)

var ErrNoReadersResponded = errors.New("no DAS readers responded successfully")

type aggregatorStrategy interface {
	newInstance() aggregatorStrategyInstance
	update([]dasutil.DASReader, map[dasutil.DASReader]readerStats)
}

type abstractAggregatorStrategy struct {
	sync.RWMutex
	readers []dasutil.DASReader
	stats   map[dasutil.DASReader]readerStats
}

func (s *abstractAggregatorStrategy) update(readers []dasutil.DASReader, stats map[dasutil.DASReader]readerStats) {
	s.Lock()
	defer s.Unlock()

	s.readers = make([]dasutil.DASReader, len(readers))
	copy(s.readers, readers)

	s.stats = maps.Clone(stats)
}

// Exponentially growing Explore Exploit Strategy
type simpleExploreExploitStrategy struct {
	iterations        atomic.Uint32
	exploreIterations uint32
	exploitIterations uint32

	abstractAggregatorStrategy
}

func (s *simpleExploreExploitStrategy) newInstance() aggregatorStrategyInstance {
	iterations := s.iterations.Add(1)

	readerSets := make([][]dasutil.DASReader, 0)
	s.RLock()
	defer s.RUnlock()

	readers := make([]dasutil.DASReader, len(s.readers))
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
		readerSet := make([]dasutil.DASReader, 0, maxTake)
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
		si.readerSets = append(si.readerSets, []dasutil.DASReader{reader})
	}

	return &si
}

// Instance of a strategy that returns readers in an order according to the strategy
type aggregatorStrategyInstance interface {
	nextReaders() []dasutil.DASReader
}

type basicStrategyInstance struct {
	readerSets [][]dasutil.DASReader
}

func (si *basicStrategyInstance) nextReaders() []dasutil.DASReader {
	if len(si.readerSets) == 0 {
		return nil
	}
	next := si.readerSets[0]
	si.readerSets = si.readerSets[1:]
	return next
}
