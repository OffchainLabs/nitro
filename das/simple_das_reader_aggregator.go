// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/stopwaiter"
)

// Most of the time we will use the SimpleDASReaderAggregator only to  aggregate
// RestfulDasClients, so the configuration and factory function are given more
// specific names.
type RestfulClientAggregatorConfig struct {
	urls []string `koanf:"urls"`
	//	policy string   `koanf:"policy"`
}

/*
type restfulAggregatorPolicy int

const (
	sequentialPolicy restfulAggregatorPolicy = iota
	broadcastPolicy
	expandingBroadcastPolicy
	pairwise
)
*/

const maxStatsHistory = 20

func NewRestfulClientAggregator(config *RestfulClientAggregatorConfig) (*SimpleDASReaderAggregator, error) {
	var a SimpleDASReaderAggregator
	for _, url := range config.urls {
		reader, err := NewRestfulDasClientFromURL(url)
		if err != nil {
			return nil, err
		}
		a.readers = append(a.readers, reader)
		a.stats[reader] = make([]readerStat, 0, maxStatsHistory)
	}
	a.statMessages = make(chan readerStatMessage, len(config.urls)*2)
	a.strategy = &simpleExploreExploitStrategy{}
	return &a, nil
}

type readerStats []readerStat

// Return the mean latency, weighted inversely by the ratio of successes : total attempts
func (s *readerStats) successRatioWeightedMeanLatency() time.Duration {
	successes, totalAttempts := 0.0, 0.0
	var totalLatency time.Duration
	for _, stat := range *s {
		if stat.success {
			successes++
			totalLatency += stat.latency
		}
		totalAttempts++

	}
	if successes == 0 {
		return time.Duration(^(uint64(1) << 63)) // max int64
	}
	return time.Duration((float64(totalLatency) * totalAttempts) / successes)
}

/*
func (s *readerStats) meanLatency() (time.Duration, error) {
	successes := int64(0)
	var total time.Duration
	for _, stat := range *s {
		if stat.success {
			successes++
			total += stat.latency
		}
	}
	if successes == 0 {
		return 0, errors.New("No readers have succeeded.")
	}
	return time.Duration(int64(total) / successes), nil
}

func (s *readerStats) successes() int {
	successes := 0
	for _, stat := range *s {
		if stat.success {
			successes++
		}
	}
	return successes
}
*/

type readerStat struct {
	latency time.Duration
	success bool
}

type readerStatMessage struct {
	readerStat
	reader arbstate.SimpleDASReader
}

type SimpleDASReaderAggregator struct {
	stopwaiter.StopWaiter
	readers []arbstate.SimpleDASReader
	stats   map[arbstate.SimpleDASReader]readerStats

	strategy aggregatorStrategy

	statMessages chan readerStatMessage
}

func (a *SimpleDASReaderAggregator) GetByHash(ctx context.Context, hash []byte) ([]byte, error) {
	type dataErrorPair struct {
		data []byte
		err  error
	}

	results := make(chan dataErrorPair, len(a.readers))
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	si := a.strategy.newInstance()
	for readers := si.nextReaders(); len(readers) != 0 && subCtx.Err() == nil; readers = si.nextReaders() {
		for _, reader := range readers {
			go func(reader arbstate.SimpleDASReader) {
				data, err := a.tryGetByHash(subCtx, hash, reader)
				results <- dataErrorPair{data, err}
			}(reader)
		}
		time.Sleep(time.Second * 5) // TODO make this configurable
	}

	var errorCollection []error
	for i := 0; i < len(a.readers); i++ {
		select {
		case <-ctx.Done():
		case result := <-results:
			if result.err != nil {
				errorCollection = append(errorCollection, result.err)
			} else {
				return result.data, nil
			}
		}
	}

	return nil, fmt.Errorf("Data wasn't able to be retrieved from any DAS Reader: %v", errorCollection)
}

func (a *SimpleDASReaderAggregator) tryGetByHash(ctx context.Context, hash []byte, reader arbstate.SimpleDASReader) ([]byte, error) {
	stat := readerStatMessage{reader: reader}
	stat.success = false

	start := time.Now()
	result, err := reader.GetByHash(ctx, hash)
	if err == nil {
		if bytes.Equal(crypto.Keccak256(result), hash) {
			stat.success = true
		} else {
			err = fmt.Errorf("SimpleDASReaderAggregator got result from reader(%v) not matching hash", reader)
		}
	}
	stat.latency = time.Since(start)

	// TODO: context cancelations including when one of n parallel requests finishes before the
	// other n-1 will be counted as failures, do we want this?
	select {
	case a.statMessages <- stat:
		// Non-blocking write to stat channel
	default:
		log.Warn("SimpleDASReaderAggregator stats processing goroutine is backed up, dropping", "dropped stats", stat)
	}

	return result, err
}

func (a *SimpleDASReaderAggregator) Start(ctx context.Context) {
	a.StopWaiter.Start(ctx)

	a.StopWaiter.LaunchThread(func(ctx context.Context) {
		select {
		case <-ctx.Done():
		case stat := <-a.statMessages:
			a.stats[stat.reader] = append(a.stats[stat.reader], stat.readerStat)
			statsLen := len(a.stats[stat.reader])
			if statsLen > maxStatsHistory {
				a.stats[stat.reader] = a.stats[stat.reader][statsLen-maxStatsHistory:]
			}
		}
	})

}

// func (a *SimpleDASReaderAggregator) StopAndWait
