// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package arbosState

import (
	"errors"
	"math/big"
	"regexp"
	"sort"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/l2pricing"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func MakeGenesisBlock(parentHash common.Hash, blockNumber uint64, timestamp uint64, stateRoot common.Hash, chainConfig *params.ChainConfig) *types.Block {
	head := &types.Header{
		Number:     new(big.Int).SetUint64(blockNumber),
		Nonce:      types.EncodeNonce(1), // the genesis block reads the init message
		Time:       timestamp,
		ParentHash: parentHash,
		Extra:      nil,
		GasLimit:   l2pricing.GethBlockGasLimit,
		GasUsed:    0,
		BaseFee:    big.NewInt(l2pricing.InitialBaseFeeWei),
		Difficulty: big.NewInt(1),
		MixDigest:  common.Hash{},
		Coinbase:   common.Address{},
		Root:       stateRoot,
	}

	genesisHeaderInfo := types.HeaderInfo{
		SendRoot:           common.Hash{},
		SendCount:          0,
		L1BlockNumber:      0,
		ArbOSFormatVersion: chainConfig.ArbitrumChainParams.InitialArbOSVersion,
	}
	genesisHeaderInfo.UpdateHeaderWithInfo(head)

	return types.NewBlock(head, nil, nil, trie.NewStackTrie(nil))
}

func InitializeArbosInDatabase(db ethdb.Database, cacheConfig *core.CacheConfig, initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage, timestamp uint64, accountsPerSync uint) (root common.Hash, err error) {
	triedbConfig := cacheConfig.TriedbConfig()
	triedbConfig.Preimages = false
	stateDatabase := state.NewDatabaseWithConfig(db, triedbConfig)
	defer func() {
		err = errors.Join(err, stateDatabase.TrieDB().Close())
	}()
	statedb, err := state.New(common.Hash{}, stateDatabase, nil)
	if err != nil {
		panic("failed to init empty statedb :" + err.Error())
	}

	noStateTrieChangesToCommitError := regexp.MustCompile("^triedb layer .+ is disk layer$")

	// commit avoids keeping the entire state in memory while importing the state.
	// At some time it was also used to avoid reprocessing the whole import in case of a crash.
	commit := func() (common.Hash, error) {
		root, err := statedb.Commit(chainConfig.ArbitrumChainParams.GenesisBlockNum, true)
		if err != nil {
			return common.Hash{}, err
		}
		err = stateDatabase.TrieDB().Commit(root, true)
		if err != nil {
			// pathdb returns an error when there are no state trie changes to commit and we try to commit.
			// This checks if the error is the expected one and ignores it.
			if (cacheConfig.StateScheme != rawdb.PathScheme) || !noStateTrieChangesToCommitError.MatchString(err.Error()) {
				return common.Hash{}, err
			}
		}
		statedb, err = state.New(root, stateDatabase, nil)
		if err != nil {
			return common.Hash{}, err
		}
		return root, nil
	}

	burner := burn.NewSystemBurner(nil, false)
	arbosState, err := InitializeArbosState(statedb, burner, chainConfig, initMessage)
	if err != nil {
		panic("failed to open the ArbOS state :" + err.Error())
	}

	chainOwner, err := initData.GetChainOwner()
	if err != nil {
		return common.Hash{}, err
	}
	if chainOwner != (common.Address{}) {
		err = arbosState.ChainOwners().Add(chainOwner)
		if err != nil {
			return common.Hash{}, err
		}
	}
	addrTable := arbosState.AddressTable()
	addrTableSize, err := addrTable.Size()
	if err != nil {
		return common.Hash{}, err
	}
	if addrTableSize != 0 {
		return common.Hash{}, errors.New("address table must be empty")
	}
	addressReader, err := initData.GetAddressTableReader()
	if err != nil {
		return common.Hash{}, err
	}
	for i := uint64(0); addressReader.More(); i++ {
		addr, err := addressReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		slot, err := addrTable.Register(*addr)
		if err != nil {
			return common.Hash{}, err
		}
		if i != slot {
			return common.Hash{}, errors.New("address table slot mismatch")
		}
	}
	if err := addressReader.Close(); err != nil {
		return common.Hash{}, err
	}

	log.Info("addresss table import complete")

	retryableReader, err := initData.GetRetryableDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	err = initializeRetryables(statedb, arbosState.RetryableState(), retryableReader, timestamp)
	if err != nil {
		return common.Hash{}, err
	}

	log.Info("retryables import complete")

	if accountsPerSync > 0 {
		_, err := commit()
		if err != nil {
			return common.Hash{}, err
		}
	}

	accountDataReader, err := initData.GetAccountDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	accountsRead := uint(0)
	for accountDataReader.More() {
		account, err := accountDataReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		err = initializeArbosAccount(statedb, arbosState, *account)
		if err != nil {
			return common.Hash{}, err
		}
		statedb.SetBalance(account.Addr, uint256.MustFromBig(account.EthBalance), tracing.BalanceChangeUnspecified)
		statedb.SetNonce(account.Addr, account.Nonce)
		if account.ContractInfo != nil {
			statedb.SetCode(account.Addr, account.ContractInfo.Code)
			for k, v := range account.ContractInfo.ContractStorage {
				statedb.SetState(account.Addr, k, v)
			}
		}
		accountsRead++
		if accountsPerSync > 0 && (accountsRead%accountsPerSync == 0) {
			log.Info("imported accounts", "count", accountsRead)
			_, err := commit()
			if err != nil {
				return common.Hash{}, err
			}
		}
	}
	if err := accountDataReader.Close(); err != nil {
		return common.Hash{}, err
	}
	return commit()
}

func initializeRetryables(statedb *state.StateDB, rs *retryables.RetryableState, initData statetransfer.RetryableDataReader, currentTimestamp uint64) error {
	var retryablesList []*statetransfer.InitializationDataForRetryable
	for initData.More() {
		r, err := initData.GetNext()
		if err != nil {
			return err
		}
		if r.Timeout <= currentTimestamp {
			statedb.AddBalance(r.Beneficiary, uint256.MustFromBig(r.Callvalue), tracing.BalanceChangeUnspecified)
			continue
		}
		retryablesList = append(retryablesList, r)
	}
	sort.Slice(retryablesList, func(i, j int) bool {
		a := retryablesList[i]
		b := retryablesList[j]
		if a.Timeout == b.Timeout {
			return arbmath.BigLessThan(a.Id.Big(), b.Id.Big())
		}
		return a.Timeout < b.Timeout
	})
	for _, r := range retryablesList {
		var to *common.Address
		if r.To != (common.Address{}) {
			addr := r.To
			to = &addr
		}
		statedb.AddBalance(retryables.RetryableEscrowAddress(r.Id), uint256.MustFromBig(r.Callvalue), tracing.BalanceChangeUnspecified)
		_, err := rs.CreateRetryable(r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
		if err != nil {
			return err
		}
	}
	return initData.Close()
}

func initializeArbosAccount(_ *state.StateDB, arbosState *ArbosState, account statetransfer.AccountInitializationInfo) error {
	l1pState := arbosState.L1PricingState()
	posterTable := l1pState.BatchPosterTable()
	if account.AggregatorInfo != nil {
		isPoster, err := posterTable.ContainsPoster(account.Addr)
		if err != nil {
			return err
		}
		if isPoster {
			// poster is already authorized, just set its fee collector
			poster, err := posterTable.OpenPoster(account.Addr, false)
			if err != nil {
				return err
			}
			err = poster.SetPayTo(account.AggregatorInfo.FeeCollector)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
