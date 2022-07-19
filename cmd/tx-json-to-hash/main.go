// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
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
		var tx types.Transaction
		err := tx.UnmarshalJSON(line)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v\n", tx.Hash())
	}
}
