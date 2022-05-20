// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

// Most of the time we will use the SimpleDASReaderAggregator only to  aggregate
// RestfulDasClients, so the configuration and factory function are given more
// specific names.
type RestfulClientAggregatorConfig struct {
	Urls                               []string                           `koanf:"urls"`
	Strategy                           string                             `koanf:"strategy"`
	StrategyUpdateInterval             time.Duration                      `koanf:"strategy-update-interval"`
	WaitBeforeTryNext                  time.Duration                      `koanf:"wait-before-try-next"`
	MaxPerEndpointStats                int                                `koanf:"max-per-endpoint-stats"`
	SimpleExploreExploitStrategyConfig SimpleExploreExploitStrategyConfig `koanf:"simple-explore-exploit-strategy"`
}

type SimpleExploreExploitStrategyConfig struct {
	exploreIterations int `koanf:"explore-iterations"`
	exploitIterations int `koanf:"exploit-iterations"`
}

func RestfulClientAggregatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.StringSlice("urls", []string{}, "List of URLs including 'http://' or 'https://' prefixes and port numbers to REST DAS endpoints.")
	f.String("strategy", "simple-explore-exploit", "Strategy to use to determine order and parallelism of calling REST endpoint URLs. Valid options are 'simple-explore-exploit'")
	f.Duration("strategy-update-interval", 10*time.Second, "How frequently to update the strategy with endpoint latency and error rate data.")
	f.Duration("wait-before-try-next", 2*time.Second, "Time to wait until trying the next set of REST endpoints while waiting for a response. The next set of REST endpoints is determined by the strategy selected.")
	f.Int("max-per-endpoint-stats", 20, "Number of stats entries (latency and success rate) to keep for each REST endpoint.")
	SimpleExploreExploitStrategyConfigAddOptions(prefix+"simple-explore-exploit-strategy", f)
}

func SimpleExploreExploitStrategyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int("explore-iterations", 20, "Number of consecutive GetByHash calls to the aggregator where each call will cause it to randomly select from REST endpoints until one returns successfully, before switching to exploit mode.")
	f.Int("exploit-iterations", 1000, "Number of consecutive GetByHash calls to the aggregator where each call will cause it to select from REST endpoints in order of best latency and success rate, before switching to explore mode.")
}

func NewRestfulClientAggregator(config *RestfulClientAggregatorConfig) (*SimpleDASReaderAggregator, error) {
	a := SimpleDASReaderAggregator{
		config: config,
		stats:  make(map[arbstate.SimpleDASReader]readerStats),
	}

	for _, url := range config.Urls {
		reader, err := NewRestfulDasClientFromURL(url)
		if err != nil {
			return nil, err
		}
		a.readers = append(a.readers, reader)
		a.stats[reader] = make([]readerStat, 0, config.MaxPerEndpointStats)
	}
	a.statMessages = make(chan readerStatMessage, len(config.Urls)*2)

	switch strings.ToLower(config.Strategy) {
	case "simple-explore-exploit":
		a.strategy = &simpleExploreExploitStrategy{
			exploreIterations: uint32(config.SimpleExploreExploitStrategyConfig.exploreIterations),
			exploitIterations: uint32(config.SimpleExploreExploitStrategyConfig.exploitIterations),
		}
	case "testing-sequential":
		a.strategy = &testingSequentialStrategy{}
	default:
		return nil, fmt.Errorf("Unknown RestfulClientAggregator strategy '%s', use --help to see available strategies.", config.Strategy)
	}
	a.strategy.update(a.readers, a.stats)
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
	avgLatency := float64(totalLatency) / successes
	successRatio := successes / totalAttempts
	return time.Duration(avgLatency / successRatio)
}

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

	config *RestfulClientAggregatorConfig

	// readers and stats are only to be updated by the stats goroutine
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

	go func() {
		si := a.strategy.newInstance()
		for readers := si.nextReaders(); len(readers) != 0 && subCtx.Err() == nil; readers = si.nextReaders() {
			wg := sync.WaitGroup{}
			waitChan := make(chan interface{})
			for _, reader := range readers {
				wg.Add(1)
				go func(reader arbstate.SimpleDASReader) {
					defer wg.Done()
					data, err := a.tryGetByHash(subCtx, hash, reader)
					results <- dataErrorPair{data, err}
				}(reader)
			}
			go func() {
				wg.Wait()
				close(waitChan)
			}()
			select {
			case <-subCtx.Done():
				return
			case <-time.After(a.config.WaitBeforeTryNext):
			case <-waitChan:
				// Yield to give the collector a chance to run in case a request succeeded
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()

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
		updateStrategyTicker := time.NewTicker(a.config.StrategyUpdateInterval)
		defer updateStrategyTicker.Stop()
		for {
			select {
			case <-ctx.Done():
			case stat := <-a.statMessages:
				a.stats[stat.reader] = append(a.stats[stat.reader], stat.readerStat)
				statsLen := len(a.stats[stat.reader])
				if statsLen > a.config.MaxPerEndpointStats {
					a.stats[stat.reader] = a.stats[stat.reader][statsLen-a.config.MaxPerEndpointStats:]
				}
			case <-updateStrategyTicker.C:
				// Strategy update happens in same goroutine as updates to the stats
				// to avoid needing extra synchronization.
				a.strategy.update(a.readers, a.stats)
			}
		}
	})

}

// func (a *SimpleDASReaderAggregator) StopAndWait
