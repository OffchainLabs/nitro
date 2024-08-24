// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package das

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbstate/daprovider"
	"github.com/offchainlabs/nitro/das/dastree"
	"github.com/offchainlabs/nitro/util/pretty"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	flag "github.com/spf13/pflag"
)

// Most of the time we will use the SimpleDASReaderAggregator only to  aggregate
// RestfulDasClients, so the configuration and factory function are given more
// specific names.
type RestfulClientAggregatorConfig struct {
	Enable                       bool                               `koanf:"enable"`
	Urls                         []string                           `koanf:"urls"`
	OnlineUrlList                string                             `koanf:"online-url-list"`
	OnlineUrlListFetchInterval   time.Duration                      `koanf:"online-url-list-fetch-interval"`
	Strategy                     string                             `koanf:"strategy"`
	StrategyUpdateInterval       time.Duration                      `koanf:"strategy-update-interval"`
	WaitBeforeTryNext            time.Duration                      `koanf:"wait-before-try-next"`
	MaxPerEndpointStats          int                                `koanf:"max-per-endpoint-stats"`
	SimpleExploreExploitStrategy SimpleExploreExploitStrategyConfig `koanf:"simple-explore-exploit-strategy"`
	SyncToStorage                SyncToStorageConfig                `koanf:"sync-to-storage"`
}

var DefaultRestfulClientAggregatorConfig = RestfulClientAggregatorConfig{
	Urls:                         []string{},
	OnlineUrlList:                "",
	OnlineUrlListFetchInterval:   1 * time.Hour,
	Strategy:                     "simple-explore-exploit",
	StrategyUpdateInterval:       10 * time.Second,
	WaitBeforeTryNext:            2 * time.Second,
	MaxPerEndpointStats:          20,
	SimpleExploreExploitStrategy: DefaultSimpleExploreExploitStrategyConfig,
	SyncToStorage:                DefaultSyncToStorageConfig,
}

type SimpleExploreExploitStrategyConfig struct {
	ExploreIterations int `koanf:"explore-iterations"`
	ExploitIterations int `koanf:"exploit-iterations"`
}

var DefaultSimpleExploreExploitStrategyConfig = SimpleExploreExploitStrategyConfig{
	ExploreIterations: 20,
	ExploitIterations: 1000,
}

func RestfulClientAggregatorConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Bool(prefix+".enable", DefaultRestfulClientAggregatorConfig.Enable, "enable retrieval of sequencer batch data from a list of remote REST endpoints; if other DAS storage types are enabled, this mode is used as a fallback")
	f.StringSlice(prefix+".urls", DefaultRestfulClientAggregatorConfig.Urls, "list of URLs including 'http://' or 'https://' prefixes and port numbers to REST DAS endpoints; additive with the online-url-list option")
	f.String(prefix+".online-url-list", DefaultRestfulClientAggregatorConfig.OnlineUrlList, "a URL to a list of URLs of REST das endpoints that is checked at startup; additive with the url option")
	f.Duration(prefix+".online-url-list-fetch-interval", DefaultRestfulClientAggregatorConfig.OnlineUrlListFetchInterval, "time interval to periodically fetch url list from online-url-list")
	f.String(prefix+".strategy", DefaultRestfulClientAggregatorConfig.Strategy, "strategy to use to determine order and parallelism of calling REST endpoint URLs; valid options are 'simple-explore-exploit'")
	f.Duration(prefix+".strategy-update-interval", DefaultRestfulClientAggregatorConfig.StrategyUpdateInterval, "how frequently to update the strategy with endpoint latency and error rate data")
	f.Duration(prefix+".wait-before-try-next", DefaultRestfulClientAggregatorConfig.WaitBeforeTryNext, "time to wait until trying the next set of REST endpoints while waiting for a response; the next set of REST endpoints is determined by the strategy selected")
	f.Int(prefix+".max-per-endpoint-stats", DefaultRestfulClientAggregatorConfig.MaxPerEndpointStats, "number of stats entries (latency and success rate) to keep for each REST endpoint; controls whether strategy is faster or slower to respond to changing conditions")
	SimpleExploreExploitStrategyConfigAddOptions(prefix+".simple-explore-exploit-strategy", f)
	SyncToStorageConfigAddOptions(prefix+".sync-to-storage", f)
}

func SimpleExploreExploitStrategyConfigAddOptions(prefix string, f *flag.FlagSet) {
	f.Int(prefix+".explore-iterations", DefaultSimpleExploreExploitStrategyConfig.ExploreIterations, "number of consecutive GetByHash calls to the aggregator where each call will cause it to randomly select from REST endpoints until one returns successfully, before switching to exploit mode")
	f.Int(prefix+".exploit-iterations", DefaultSimpleExploreExploitStrategyConfig.ExploitIterations, "number of consecutive GetByHash calls to the aggregator where each call will cause it to select from REST endpoints in order of best latency and success rate, before switching to explore mode")
}

func NewRestfulClientAggregator(ctx context.Context, config *RestfulClientAggregatorConfig) (*SimpleDASReaderAggregator, error) {
	a := SimpleDASReaderAggregator{
		config: config,
		stats:  make(map[daprovider.DASReader]readerStats),
	}

	combinedUrls := make(map[string]bool)
	for _, url := range config.Urls {
		combinedUrls[url] = true
	}
	if config.OnlineUrlList != DefaultRestfulClientAggregatorConfig.OnlineUrlList {
		onlineUrls, err := RestfulServerURLsFromList(ctx, config.OnlineUrlList)
		if err != nil {
			return nil, err
		}
		for _, url := range onlineUrls {
			combinedUrls[url] = true
		}
	}
	if len(combinedUrls) == 0 {
		return nil, errors.New("no URLs were specified with either of rest-aggregator.urls or rest-aggregator.online-url-list")
	}

	urls := make([]string, 0, len(combinedUrls))
	for url := range combinedUrls {
		urls = append(urls, url)
	}

	log.Info("REST Aggregator URLs", "urls", urls)

	for _, url := range urls {
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
			exploreIterations: uint32(config.SimpleExploreExploitStrategy.ExploreIterations),
			exploitIterations: uint32(config.SimpleExploreExploitStrategy.ExploitIterations),
		}
	case "testing-sequential":
		a.strategy = &testingSequentialStrategy{}
	default:
		return nil, fmt.Errorf("unknown RestfulClientAggregator strategy '%s', use --help to see available strategies", config.Strategy)
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
		return time.Duration(math.MaxInt64)
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
	reader daprovider.DASReader
}

type SimpleDASReaderAggregator struct {
	stopwaiter.StopWaiter

	config *RestfulClientAggregatorConfig

	readersMutex sync.RWMutex
	// readers and stats are only to be updated by the stats goroutine
	readers []daprovider.DASReader
	stats   map[daprovider.DASReader]readerStats

	strategy aggregatorStrategy

	statMessages chan readerStatMessage
}

func (a *SimpleDASReaderAggregator) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	a.readersMutex.RLock()
	defer a.readersMutex.RUnlock()
	log.Trace("das.SimpleDASReaderAggregator.GetByHash", "key", pretty.PrettyHash(hash), "this", a)

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
				go func(reader daprovider.DASReader) {
					defer wg.Done()
					data, err := a.tryGetByHash(subCtx, hash, reader)
					if err != nil && errors.Is(ctx.Err(), context.Canceled) {
						// Don't record a stats data point when a different
						// client returned faster than this one.
						return
					}
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
			return nil, ctx.Err()
		case result := <-results:
			if result.err != nil {
				errorCollection = append(errorCollection, result.err)
			} else {
				return result.data, nil
			}
		}
	}

	return nil, fmt.Errorf("data wasn't able to be retrieved from any DAS Reader: %v", errorCollection)
}

func (a *SimpleDASReaderAggregator) tryGetByHash(
	ctx context.Context, hash common.Hash, reader daprovider.DASReader,
) ([]byte, error) {
	stat := readerStatMessage{reader: reader}
	stat.success = false

	start := time.Now()
	result, err := reader.GetByHash(ctx, hash)
	if err == nil {
		if dastree.ValidHash(hash, result) {
			stat.success = true
		} else {
			err = fmt.Errorf("SimpleDASReaderAggregator got result from reader(%v) not matching hash", reader)
		}
	}
	stat.latency = time.Since(start)

	select {
	case a.statMessages <- stat:
		// Non-blocking write to stat channel
	default:
		log.Warn("SimpleDASReaderAggregator stats processing goroutine is backed up, dropping", "dropped stats", stat)
	}

	return result, err
}

func (a *SimpleDASReaderAggregator) Start(ctx context.Context) {
	a.StopWaiter.Start(ctx, a)
	onlineUrlsChan := StartRestfulServerListFetchDaemon(a.StopWaiter.GetContext(), a.config.OnlineUrlList, a.config.OnlineUrlListFetchInterval)

	updateRestfulDasClients := func(urls []string) {
		a.readersMutex.Lock()
		defer a.readersMutex.Unlock()
		combinedUrls := a.config.Urls
		combinedUrls = append(combinedUrls, urls...)
		combinedReaders := make(map[daprovider.DASReader]bool)
		for _, url := range combinedUrls {
			reader, err := NewRestfulDasClientFromURL(url)
			if err != nil {
				return
			}
			combinedReaders[reader] = true
		}
		a.readers = make([]daprovider.DASReader, 0, len(combinedUrls))
		// Update reader and add newly added stats
		for reader := range combinedReaders {
			a.readers = append(a.readers, reader)
			if _, ok := a.stats[reader]; ok {
				continue
			}
			a.stats[reader] = make([]readerStat, 0, a.config.MaxPerEndpointStats)
		}
		// Delete stats for removed reader
		for reader := range a.stats {
			if combinedReaders[reader] {
				continue
			}
			delete(a.stats, reader)
		}
	}

	a.StopWaiter.LaunchThread(func(innerCtx context.Context) {
		updateStrategyTicker := time.NewTicker(a.config.StrategyUpdateInterval)
		defer updateStrategyTicker.Stop()
		for {
			select {
			case <-innerCtx.Done():
				return
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
			case onlineUrls := <-onlineUrlsChan:
				updateRestfulDasClients(onlineUrls)
			}
		}
	})
}

func (a *SimpleDASReaderAggregator) Close(ctx context.Context) error {
	a.StopWaiter.StopOnly()
	waitChan, err := a.StopWaiter.GetWaitChannel()
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitChan:
		return nil
	}
}

func (a *SimpleDASReaderAggregator) String() string {
	return fmt.Sprintf("das.SimpleDASReaderAggregator{%v}", a.config.Urls)
}

func (a *SimpleDASReaderAggregator) HealthCheck(ctx context.Context) error {
	return nil
}

func (a *SimpleDASReaderAggregator) ExpirationPolicy(ctx context.Context) (daprovider.ExpirationPolicy, error) {
	a.readersMutex.RLock()
	defer a.readersMutex.RUnlock()
	if len(a.readers) == 0 {
		return -1, errors.New("no DataAvailabilityService present")
	}
	expectedExpirationPolicy, err := a.readers[0].ExpirationPolicy(ctx)
	if err != nil {
		return -1, err
	}
	// Even if a single service is different from the rest,
	// then whole aggregator will be considered for mixed expiration timeout policy.
	for _, serv := range a.readers {
		ep, err := serv.ExpirationPolicy(ctx)
		if err != nil {
			return -1, err
		}
		if ep != expectedExpirationPolicy {
			return daprovider.MixedTimeout, nil
		}
	}
	return expectedExpirationPolicy, nil
}
