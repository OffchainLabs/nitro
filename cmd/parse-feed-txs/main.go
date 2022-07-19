// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/broadcaster"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	var line []byte
	for {
		line = line[:0]
		for {
			partial, isPartial, err := reader.ReadLine()
			if err != nil {
				if errors.Is(err, io.EOF) {
					if len(line) == 0 {
						return
					} else {
						break
					}
				}
				panic(err)
			}
			line = append(line, partial...)
			if !isPartial {
				break
			}
		}
		if len(line) == 0 {
			continue
		}
		var msg broadcaster.BroadcastMessage
		err := json.Unmarshal(line, &msg)
		if err != nil {
			panic(err)
		}
		for _, msg := range msg.Messages {
			var isBatchPostingReport bool
			batchFetcher := func(uint64) []byte {
				isBatchPostingReport = true
				return nil
			}
			txs, err := msg.Message.Message.ParseL2Transactions(nil, batchFetcher)
			if isBatchPostingReport {
				continue
			}
			if err != nil {
				log.Warn("error parsing message", "err", err)
			}
			for _, tx := range txs {
				if tx.Type() >= 100 {
					continue
				}
				bytes, err := tx.MarshalJSON()
				if err != nil {
					panic(err)
				}
				fmt.Printf("%v\n", string(bytes))
			}
		}
	}
}
