// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
)

type dummyReader struct {
	int
}

func (*dummyReader) GetByHash(context.Context, common.Hash) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (*dummyReader) HealthCheck(context.Context) error {
	return errors.New("not implemented")
}

func (*dummyReader) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	return -1, errors.New("not implemented")
}

func TestDAS_SimpleExploreExploit(t *testing.T) {
	readers := []daprovider.DASReader{&dummyReader{0}, &dummyReader{1}, &dummyReader{2}, &dummyReader{3}, &dummyReader{4}, &dummyReader{5}}
	stats := make(map[daprovider.DASReader]readerStats)
	stats[readers[0]] = []readerStat{ // weighted avg 10s
		{10 * time.Second, true},
	}
	stats[readers[1]] = []readerStat{ // weighted avg 5s
		{6 * time.Second, true},
		{4 * time.Second, true},
	}
	stats[readers[2]] = []readerStat{ // weighted avg 3 / (1/2) = 6s
		{3 * time.Second, true},
		{3 * time.Second, false},
	}
	stats[readers[3]] = []readerStat{ // weighted avg max int
		{1 * time.Second, false},
		{1 * time.Second, false},
	}
	stats[readers[4]] = []readerStat{ // weighted avg 3 / (1/3) = 9s
		{3 * time.Second, true},
		{3 * time.Second, false},
		{3 * time.Second, false},
	}
	stats[readers[5]] = []readerStat{ // weighted avg 8s
		{8 * time.Second, true},
	}

	expectedOrdering := []daprovider.DASReader{readers[1], readers[2], readers[5], readers[4], readers[0], readers[3]}

	expectedExploreIterations, expectedExploitIterations := uint32(5), uint32(5)
	strategy := simpleExploreExploitStrategy{
		exploreIterations: expectedExploreIterations,
		exploitIterations: expectedExploitIterations,
	}
	strategy.update(readers, stats)

	checkMatch := func(expected, was []daprovider.DASReader, doMatch bool) {
		if len(expected) != len(was) {
			Fail(t, fmt.Sprintf("Incorrect number of nextReaders %d, expected %d", len(was), len(expected)))
		}

		for i := 0; i < len(was) && doMatch; i++ {
			if expected[i].(*dummyReader).int != was[i].(*dummyReader).int {
				Fail(t, fmt.Sprintf("expected %d, was %d", expected[i].(*dummyReader).int, was[i].(*dummyReader).int))
			}
		}
	}

	// In Explore mode we just care about the exponential growth
	for i := uint32(0); i < expectedExploreIterations-1; i++ {
		si := strategy.newInstance()
		checkMatch(expectedOrdering[:1], si.nextReaders(), false)
		checkMatch(expectedOrdering[1:3], si.nextReaders(), false)
		checkMatch(expectedOrdering[3:6], si.nextReaders(), false)
		checkMatch(expectedOrdering[6:], si.nextReaders(), false)
	}

	// In Exploit mode we can check the ordering too.
	for i := uint32(0); i < expectedExploitIterations; i++ {
		si := strategy.newInstance()
		checkMatch(expectedOrdering[:1], si.nextReaders(), true)
		checkMatch(expectedOrdering[1:3], si.nextReaders(), true)
		checkMatch(expectedOrdering[3:6], si.nextReaders(), true)
		checkMatch(expectedOrdering[6:], si.nextReaders(), true)
	}

	// Cycle through explore/exploit one more time
	for i := uint32(0); i < expectedExploreIterations; i++ {
		si := strategy.newInstance()
		checkMatch(expectedOrdering[:1], si.nextReaders(), false)
		checkMatch(expectedOrdering[1:3], si.nextReaders(), false)
		checkMatch(expectedOrdering[3:6], si.nextReaders(), false)
		checkMatch(expectedOrdering[6:], si.nextReaders(), false)
	}

	for i := uint32(0); i < expectedExploitIterations; i++ {
		si := strategy.newInstance()
		checkMatch(expectedOrdering[:1], si.nextReaders(), true)
		checkMatch(expectedOrdering[1:3], si.nextReaders(), true)
		checkMatch(expectedOrdering[3:6], si.nextReaders(), true)
		checkMatch(expectedOrdering[6:], si.nextReaders(), true)
	}

}
