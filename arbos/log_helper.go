package arbos

import (
	"fmt"

	log_helper "github.com/EspressoSystems/espresso-sequencer-go/log-helper"
	"github.com/ethereum/go-ethereum/log"
)

var (
	logHelper                = log_helper.NewLogger()
	channelFetchHeader       = log_helper.Channel("fetch header channel")
	channelFetchTransactions = log_helper.Channel("fetch transactions channel")
)

func init() {
	logHelper.AddLogAfterRetryStrategy(channelFetchHeader, "0", 20)
	logHelper.AddLogAfterRetryStrategy(channelFetchTransactions, "0", 20)
}

func LogFailedToFetchHeader(block uint64) {
	logHelper.Attempt(channelFetchHeader, fmt.Sprintf("%d", block), func() {
		log.Warn("Unable to fetch header for block number, will retry", "block_num", block)
	})
}

func LogFailedToFetchTransactions(block uint64, err error) {
	logHelper.Attempt(channelFetchHeader, fmt.Sprintf("%d", block), func() {
		log.Error("Error fetching transactions", "block", block, "err", err)
	})
}
