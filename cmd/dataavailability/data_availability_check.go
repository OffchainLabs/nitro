// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbstate"
	"github.com/offchainlabs/nitro/cmd/util/confighelpers"
	"github.com/offchainlabs/nitro/das"
	"github.com/offchainlabs/nitro/solgen/go/bridgegen"
	"github.com/offchainlabs/nitro/util/metricsutil"
	"github.com/offchainlabs/nitro/util/stopwaiter"

	flag "github.com/spf13/pflag"
)

// Data availability check is done to as to make sure that the data that is being stored by DAS is available at all time.
// This done by taking the latest stored hash and an old stored hash (12 days) and it is checked if these two hashes are
// present across all the DAS provided in the list, if a DAS does not have these hashes an error is thrown.
// This approach does not guarantee 100% data availability, but it's an efficient and easy heuristic for our use case.
//
// This can be used in following manner (not an exhaustive list)
// 1. Continuously call the function by exposing a REST API and create alert if error is returned.
// 2. Call the function in an adhoc manner to check if the provided DAS is live and functioning properly.

const metricBaseOldHash = "arb/das/dataavailability/oldhash/"
const metricBaseNewHash = "arb/das/dataavailability/oldhash/"

type DataAvailabilityCheckConfig struct {
	OnlineUrlList         string        `koanf:"online-url-list"`
	L1NodeURL             string        `koanf:"l1-node-url"`
	L1ConnectionAttempts  int           `koanf:"l1-connection-attempts"`
	SequencerInboxAddress string        `koanf:"sequencer-inbox-address"`
	L1BlocksPerRead       uint64        `koanf:"l1-blocks-per-read"`
	CheckInterval         time.Duration `koanf:"check-interval"`
}

var DefaultDataAvailabilityCheckConfig = DataAvailabilityCheckConfig{
	OnlineUrlList:        "",
	L1ConnectionAttempts: 15,
	L1BlocksPerRead:      100,
	CheckInterval:        5 * time.Minute,
}

type DataAvailabilityCheck struct {
	stopwaiter.StopWaiter
	l1Client       *ethclient.Client
	config         *DataAvailabilityCheckConfig
	inboxAddr      *common.Address
	inboxContract  *bridgegen.SequencerInbox
	urlToReaderMap map[string]arbstate.DataAvailabilityReader
	checkInterval  time.Duration
}

func newDataAvailabilityCheck(ctx context.Context, dataAvailabilityCheckConfig *DataAvailabilityCheckConfig) (*DataAvailabilityCheck, error) {
	l1Client, err := das.GetL1Client(ctx, dataAvailabilityCheckConfig.L1ConnectionAttempts, dataAvailabilityCheckConfig.L1NodeURL)
	if err != nil {
		return nil, err
	}
	seqInboxAddress, err := das.OptionalAddressFromString(dataAvailabilityCheckConfig.SequencerInboxAddress)
	if err != nil {
		return nil, err
	}
	inboxContract, err := bridgegen.NewSequencerInbox(*seqInboxAddress, l1Client)
	if err != nil {
		return nil, err
	}
	onlineUrls, err := das.RestfulServerURLsFromList(ctx, dataAvailabilityCheckConfig.OnlineUrlList)
	if err != nil {
		return nil, err
	}
	urlToReaderMap := make(map[string]arbstate.DataAvailabilityReader, len(onlineUrls))
	for _, url := range onlineUrls {
		reader, err := das.NewRestfulDasClientFromURL(url)
		if err != nil {
			return nil, err
		}
		urlToReaderMap[url] = reader
	}
	return &DataAvailabilityCheck{
		l1Client:       l1Client,
		config:         dataAvailabilityCheckConfig,
		inboxAddr:      seqInboxAddress,
		inboxContract:  inboxContract,
		urlToReaderMap: urlToReaderMap,
		checkInterval:  dataAvailabilityCheckConfig.CheckInterval,
	}, nil
}

func parseDataAvailabilityCheckConfig(args []string) (*DataAvailabilityCheckConfig, error) {
	f := flag.NewFlagSet("dataavailabilitycheck", flag.ContinueOnError)
	f.String("online-url-list", DefaultDataAvailabilityCheckConfig.OnlineUrlList, "a URL to a list of URLs of REST das endpoints that is checked for data availability")
	f.String("l1-node-url", DefaultDataAvailabilityCheckConfig.L1NodeURL, "URL for L1 node")
	f.Int("l1-connection-attempts", DefaultDataAvailabilityCheckConfig.L1ConnectionAttempts, "layer 1 RPC connection attempts (spaced out at least 1 second per attempt, 0 to retry infinitely)")
	f.String("sequencer-inbox-address", DefaultDataAvailabilityCheckConfig.SequencerInboxAddress, "L1 address of SequencerInbox contract")
	f.Uint64("l1-blocks-per-read", DefaultDataAvailabilityCheckConfig.L1BlocksPerRead, "max l1 blocks to read per poll")
	f.Duration("check-interval", DefaultDataAvailabilityCheckConfig.CheckInterval, "interval for running data availability check")
	k, err := confighelpers.BeginCommonParse(f, args)
	if err != nil {
		return nil, err
	}

	var config DataAvailabilityCheckConfig
	if err := confighelpers.EndCommonParse(k, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	dataAvailabilityCheckConfig, err := parseDataAvailabilityCheckConfig(os.Args[1:])
	if err != nil {
		panic(err)
	}
	dataAvailabilityCheck, err := newDataAvailabilityCheck(ctx, dataAvailabilityCheckConfig)
	if err != nil {
		panic(err)
	}
	dataAvailabilityCheck.StopWaiter.Start(ctx, dataAvailabilityCheck)
	dataAvailabilityCheck.CallIteratively(dataAvailabilityCheck.start)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	<-sigint
	signal.Stop(sigint)
	close(sigint)
	dataAvailabilityCheck.StopAndWait()
}

func (d *DataAvailabilityCheck) start(ctx context.Context) time.Duration {
	latestHeader, err := d.l1Client.HeaderByNumber(ctx, big.NewInt(rpc.FinalizedBlockNumber.Int64()))
	if err != nil {
		log.Error(err.Error())
		return d.checkInterval
	}
	latestBlockNumber := latestHeader.Number.Uint64()
	oldBlockNumber := latestBlockNumber - 86400 // 12 days old block number

	log.Info("Running new hash data availability check")
	newHashErr := d.checkDataAvailabilityForNewHashInBlockRange(ctx, latestBlockNumber, oldBlockNumber)
	log.Info("Completed new hash data availability check")

	log.Info("Running old hash data availability check")
	oldHashErr := d.checkDataAvailabilityForOldHashInBlockRange(ctx, oldBlockNumber, latestBlockNumber)
	log.Info("Completed old hash data availability check")

	if newHashErr != nil || oldHashErr != nil {
		log.Error(fmt.Sprintf("new hash check: %s, old hash check: %s", newHashErr, oldHashErr))
	}
	return d.checkInterval
}

func (d *DataAvailabilityCheck) checkDataAvailabilityForNewHashInBlockRange(ctx context.Context, latestBlock uint64, oldBlock uint64) error {
	currentBlock := latestBlock
	for currentBlock-d.config.L1BlocksPerRead >= oldBlock {
		query := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(currentBlock - d.config.L1BlocksPerRead),
			ToBlock:   new(big.Int).SetUint64(currentBlock),
			Addresses: []common.Address{*d.inboxAddr},
			Topics:    [][]common.Hash{{das.BatchDeliveredID}},
		}
		logs, err := d.l1Client.FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		for _, deliveredLog := range logs {
			isDasMessage, err := d.checkDataAvailability(ctx, deliveredLog, metricBaseNewHash)
			if err != nil {
				return err
			}
			if isDasMessage {
				return nil
			}
		}
		currentBlock = currentBlock - d.config.L1BlocksPerRead
	}
	return fmt.Errorf("no das message found between block %d and block %d", latestBlock, oldBlock)
}

func (d *DataAvailabilityCheck) checkDataAvailabilityForOldHashInBlockRange(ctx context.Context, oldBlock uint64, latestBlock uint64) error {
	currentBlock := oldBlock
	for currentBlock+d.config.L1BlocksPerRead <= latestBlock {
		query := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(currentBlock),
			ToBlock:   new(big.Int).SetUint64(currentBlock + d.config.L1BlocksPerRead),
			Addresses: []common.Address{*d.inboxAddr},
			Topics:    [][]common.Hash{{das.BatchDeliveredID}},
		}
		logs, err := d.l1Client.FilterLogs(ctx, query)
		if err != nil {
			return err
		}
		for _, deliveredLog := range logs {
			isDasMessage, err := d.checkDataAvailability(ctx, deliveredLog, metricBaseOldHash)
			if err != nil {
				return err
			}
			if isDasMessage {
				return nil
			}
		}
		currentBlock = currentBlock + d.config.L1BlocksPerRead
	}
	return fmt.Errorf("no das message found between block %d and block %d", oldBlock, latestBlock)
}

// Trys to find if DAS message is present in the given log and if present
// returns true and validates if the data is available in the storage service.
func (d *DataAvailabilityCheck) checkDataAvailability(ctx context.Context, deliveredLog types.Log, metricBase string) (bool, error) {
	deliveredEvent, err := d.inboxContract.ParseSequencerBatchDelivered(deliveredLog)
	if err != nil {
		return false, err
	}
	data, err := das.FindDASDataFromLog(ctx, d.inboxContract, deliveredEvent, *d.inboxAddr, d.l1Client, deliveredLog)
	if err != nil {
		return false, err
	}
	if data == nil {
		return false, nil
	}
	cert, err := arbstate.DeserializeDASCertFrom(bytes.NewReader(data))
	if err != nil {
		return true, err
	}
	var dataNotFound []string
	for url, reader := range d.urlToReaderMap {
		_, err = reader.GetByHash(ctx, cert.DataHash)
		canonicalUrl := metricsutil.CanonicalizeMetricName(url)
		if err != nil {
			metrics.GetOrRegisterCounter(metricBase+"/"+canonicalUrl+"/failure", nil).Inc(1)
			dataNotFound = append(dataNotFound, url)
			log.Error(fmt.Sprintf("Data with hash: %s not found for: %s\n", common.Hash(cert.DataHash).String(), url))
		} else {
			metrics.GetOrRegisterCounter(metricBase+"/"+canonicalUrl+"/success", nil).Inc(1)
		}
	}
	if len(dataNotFound) > 0 {
		return true, fmt.Errorf("data with hash: %s not found for das:%s", common.Hash(cert.DataHash).String(), dataNotFound)
	}
	return true, nil
}
