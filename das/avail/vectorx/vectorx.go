package vectorx

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const VectorxABI = `[
    {
        "type": "event",
        "name": "HeadUpdate",
        "inputs": [
            {
                "name": "blockNumber",
                "type": "uint32",
                "indexed": false,
                "internalType": "uint32"
            },
            {
                "name": "headerHash",
                "type": "bytes32",
                "indexed": false,
                "internalType": "bytes32"
            }
        ],
        "anonymous": false
    }
]`

type VectorX struct {
	Abi    abi.ABI
	Client *ethclient.Client
	Query  ethereum.FilterQuery
}

func (v *VectorX) SubscribeForHeaderUpdate(finalizedBlockNumber int, t time.Duration) error {
	// Subscribe to the event stream
	logs := make(chan types.Log)
	sub, err := v.Client.SubscribeFilterLogs(context.Background(), v.Query, logs)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	log.Info("🎧  Listening for vectorx HeadUpdate event with", "blockNumber", finalizedBlockNumber)
	timeout := time.After(t * time.Second)
	// Loop to process incoming events
	for {
		select {
		case err := <-sub.Err():
			return err
		case vLog := <-logs:
			event, err := v.Abi.Unpack("HeadUpdate", vLog.Data)
			if err != nil {
				return err
			}

			log.Info("🤝  New HeadUpdate event from vecotorx", "blockNumber", event[0])
			val, _ := event[0].(uint32)
			if val >= uint32(finalizedBlockNumber) {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("⌛️  Timeout of %d seconds reached without getting HeadUpdate event from vectorx for blockNumber %v", t, finalizedBlockNumber)
		}
	}
}
