package vectorx

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type VectorX struct {
	Abi    abi.ABI
	Client ethclient.Client
	Query  ethereum.FilterQuery
}

func (v *VectorX) subscribeForHeaderUpdate() (int, error) {
	// Subscribe to the event stream
	logs := make(chan types.Log)
	sub, err := v.Client.SubscribeFilterLogs(context.Background(), v.Query, logs)
	if err != nil {
		return -1, err
	}
	defer sub.Unsubscribe()

	log.Info("🎧 Listening for vectorx HeadUpdate event")

	// Loop to process incoming events
	for {
		select {
		case err := <-sub.Err():
			return -1, err
		case vLog := <-logs:
			// Decode the event log data
			// event := struct {
			// 	Message string
			// }{}
			event, err := v.Abi.Unpack("HeadUpdate", vLog.Data)
			if err != nil {
				return -1, err
			}

			log.Info("Received message:", event)
			return event[0], nil
		}
	}
}
