package staker

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/stopwaiter"
	"github.com/offchainlabs/nitro/validator"
)

// TODO rename to ConfirmedNodeHelper?
type ConfirmedNodeHelper struct {
	stopwaiter.StopWaiter
	rollupAddress common.Address
	client        arbutil.L1Interface

	// TODO refactor subscribers
	subscribers     []LatestConfirmedNotifier
	subscribersLock sync.Mutex
}

func NewConfirmedNodeHelper(rollupAddress common.Address, client arbutil.L1Interface) *ConfirmedNodeHelper {
	return &ConfirmedNodeHelper{
		rollupAddress: rollupAddress,
		client:        client,
		subscribers:   []LatestConfirmedNotifier{},
	}
}

func (h *ConfirmedNodeHelper) Start(ctx context.Context) {
	h.StopWaiter.Start(ctx, h)
}

func (h *ConfirmedNodeHelper) UpdateLatestConfirmed(count arbutil.MessageIndex, globalState validator.GoGlobalState, node uint64) {
	// TODO propagate the update in a separate thread
	h.subscribersLock.Lock()
	defer h.subscribersLock.Unlock()
	for _, subscriber := range h.subscribers {
		subscriber.UpdateLatestConfirmed(count, globalState, node)
	}
}

func (h *ConfirmedNodeHelper) SubscribeLatest(subscriber execution.LatestConfirmedNotifier) error {
	h.subscribersLock.Lock()
	defer h.subscribersLock.Unlock()
	h.subscribers = append(h.subscribers, subscriber)
	return nil
}

func (h *ConfirmedNodeHelper) Validate(node uint64, blockHash common.Hash) (bool, error) {
	ctx, err := h.GetContextSafe()
	if err != nil {
		return false, err
	}
	// TODO do a binary search for block containing NodeConfirmed for validated node
	var query = ethereum.FilterQuery{
		FromBlock: nil,
		ToBlock:   nil,
		Addresses: []common.Address{h.rollupAddress},
		Topics:    [][]common.Hash{{nodeConfirmedID}, {blockHash}, nil},
	}
	logs, err := h.client.FilterLogs(ctx, query)
	if err != nil {
		return false, err
	}
	if len(logs) == 0 {
		return false, nil
	}
	if len(logs) > 1 {
		// TODO verify if it can happen, and if we should handle it better
		log.Error("Found more then one log when validating confirmed node", "node", node, "blockHash", blockHash, "logs", logs)
		return false, errors.New("unexpected number of logs for node confirmation")
	}
	// TODO validate the log?
	return true, nil
}
