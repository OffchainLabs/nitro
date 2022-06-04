// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	"math/big"
)

var (
	ErrNotExist      = errors.New("batch poster does not exist in table")
	ErrAlreadyExists = errors.New("tried to add a batch poster that already exists")
)

// layout of storage in the table
type BatchPostersTable struct {
	storage    *storage.Storage
	numPosters storage.StorageBackedUint64
}

type BatchPosterState struct {
	existsIfNonzero storage.StorageBackedUint64
	fundsDue        storage.StorageBackedBigInt
	payTo           storage.StorageBackedAddress
}

func InitializeBatchPostersTable(storage *storage.Storage) error {
	// no initialization needed at present
	return nil
}

func OpenBatchPostersTable(storage *storage.Storage) *BatchPostersTable {
	return &BatchPostersTable{
		storage:    storage,
		numPosters: storage.OpenStorageBackedUint64(0),
	}
}

func (bpt *BatchPostersTable) OpenPoster(poster common.Address) (*BatchPosterState, error) {
	bpState := bpt.internalOpen(poster)
	existsIfNonzero, err := bpState.existsIfNonzero.Get()
	if err != nil {
		return nil, err
	}
	if existsIfNonzero == 0 {
		return nil, ErrNotExist
	}
	return bpState, nil
}

func (bpt *BatchPostersTable) internalOpen(poster common.Address) *BatchPosterState {
	bpStorage := bpt.storage.OpenSubStorage(poster.Bytes())
	return &BatchPosterState{
		existsIfNonzero: bpStorage.OpenStorageBackedUint64(0),
		fundsDue:        bpStorage.OpenStorageBackedBigInt(1),
		payTo:           bpStorage.OpenStorageBackedAddress(2),
	}
}

func (bpt *BatchPostersTable) ContainsPoster(poster common.Address) (bool, error) {
	_, err := bpt.OpenPoster(poster)
	if errors.Is(err, ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (bpt *BatchPostersTable) AddPoster(posterAddress common.Address, payTo common.Address) (*BatchPosterState, error) {
	bpState := bpt.internalOpen(posterAddress)
	alreadyExists, err := bpState.existsIfNonzero.Get()
	if err != nil {
		return nil, err
	}
	if alreadyExists != 0 {
		return nil, ErrAlreadyExists
	}
	if err := bpState.fundsDue.Set(big.NewInt(0)); err != nil {
		return nil, err
	}
	if err := bpState.payTo.Set(payTo); err != nil {
		return nil, err
	}
	if err := bpState.existsIfNonzero.Set(1); err != nil {
		return nil, err
	}

	numPosters, err := bpt.numPosters.Get()
	if err != nil {
		return nil, err
	}
	if err := bpt.storage.SetByUint64(numPosters+1, util.AddressToHash(posterAddress)); err != nil {
		return nil, err
	}
	if err := bpt.numPosters.Set(numPosters + 1); err != nil {
		return nil, err
	}

	return bpState, nil
}

func (bpt *BatchPostersTable) AllPosters() ([]common.Address, error) {
	numPosters, err := bpt.numPosters.Get()
	if err != nil {
		return nil, err
	}
	ret := []common.Address{}
	for i := uint64(0); i < numPosters; i++ {
		posterAddrAsHash, err := bpt.storage.GetByUint64(i + 1)
		if err != nil {
			return nil, err
		}
		ret = append(ret, common.BytesToAddress(posterAddrAsHash.Bytes()))
	}
	return ret, nil
}

func (bpt *BatchPostersTable) TotalFundsDue() (*big.Int, error) {
	allPosters, err := bpt.AllPosters()
	if err != nil {
		return nil, err
	}
	ret := big.NewInt(0)
	for _, posterAddr := range allPosters {
		poster, err := bpt.OpenPoster(posterAddr)
		if err != nil {
			return nil, err
		}
		fundsDue, err := poster.FundsDue()
		if err != nil {
			return nil, err
		}
		ret = arbmath.BigAdd(ret, fundsDue)
	}
	return ret, nil
}

func (bps *BatchPosterState) FundsDue() (*big.Int, error) {
	return bps.fundsDue.Get()
}

func (bps *BatchPosterState) SetFundsDue(val *big.Int) error {
	return bps.fundsDue.Set(val)
}

func (bps *BatchPosterState) PayTo() (common.Address, error) {
	return bps.payTo.Get()
}

func (bps *BatchPosterState) SetPayTo(addr common.Address) error {
	return bps.payTo.Set(addr)
}

type FundsDueItem struct {
	dueTo   common.Address
	balance *big.Int
}

func (bpt *BatchPostersTable) GetFundsDueList() ([]FundsDueItem, error) {
	ret := []FundsDueItem{}
	allPosters, err := bpt.AllPosters()
	if err != nil {
		return nil, err
	}
	for _, posterAddr := range allPosters {
		poster, err := bpt.OpenPoster(posterAddr)
		if err != nil {
			return nil, err
		}
		due, err := poster.FundsDue()
		if err != nil {
			return nil, err
		}
		if due.Sign() > 0 {
			ret = append(ret, FundsDueItem{
				dueTo:   posterAddr,
				balance: due,
			})
		}
	}
	return ret, nil
}
