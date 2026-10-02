// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package blocks

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

const blocksToMine = 300

var historicalBlockHashTests = []systest.Scenario{
	systest.Test(testRunHistoricalBlockHash, systest.MinArbOS(params.ArbosVersion_40)),
}

func testRunHistoricalBlockHash(env *systest.Env) {
	for {
		env.L2.AdvanceBlocks(1)
		number, err := env.L2.Client.BlockNumber(env.Ctx)
		env.Require(err)
		if number > blocksToMine {
			break
		}
	}

	blocksWithStoredHash := env.L2.HeaderByNumber(nil).Number.Uint64()

	for i := range blocksWithStoredHash {
		expected := env.L2.HeaderByNumber(new(big.Int).SetUint64(i)).Hash()
		stored := storedBlockHash(env, params.HistoryStorageAddress, i)

		env.Equal(expected, stored, "history storage hash for block %d", i)
	}
}

func storedBlockHash(env *systest.Env, historyStorage common.Address, blockNumber uint64) common.Hash {
	var key common.Hash
	binary.BigEndian.PutUint64(key[24:], blockNumber)

	return common.BytesToHash(env.L2.CallContract(ethereum.CallMsg{To: &historyStorage, Data: key.Bytes()}, nil))
}
