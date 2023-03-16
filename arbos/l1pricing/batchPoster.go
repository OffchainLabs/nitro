// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"errors"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

const totalFundsDueOffset = 0

var (
	PosterAddrsKey = []byte{0}
	PosterInfoKey  = []byte{1}

	ErrAlreadyExists = errors.New("tried to add a batch poster that already exists")
	ErrNotExist      = errors.New("tried to open a batch poster that does not exist")
)

// BatchPostersTable is the layout of storage in the table
type BatchPostersTable struct {
	posterAddrs   *addressSet.AddressSet
	posterInfo    *storage.Storage
	totalFundsDue storage.StorageBackedBigInt
}

type BatchPosterState struct {
	fundsDue     storage.StorageBackedBigInt
	payTo        storage.StorageBackedAddress
	postersTable *BatchPostersTable
}

func InitializeBatchPostersTable(storage *storage.Storage) error {
	totalFundsDue := storage.OpenStorageBackedBigInt(totalFundsDueOffset)
	if err := totalFundsDue.SetChecked(common.Big0); err != nil {
		return err
	}
	return addressSet.Initialize(storage.OpenSubStorage(PosterAddrsKey))
}

func OpenBatchPostersTable(storage *storage.Storage, arbosVersion uint64) *BatchPostersTable {
	return &BatchPostersTable{
		posterAddrs:   addressSet.OpenAddressSet(storage.OpenSubStorage(PosterAddrsKey), arbosVersion),
		posterInfo:    storage.OpenSubStorage(PosterInfoKey),
		totalFundsDue: storage.OpenStorageBackedBigInt(totalFundsDueOffset),
	}
}

func (bpt *BatchPostersTable) OpenPoster(poster common.Address, createIfNotExist bool) (*BatchPosterState, error) {
	isBatchPoster, err := bpt.posterAddrs.IsMember(poster)
	if err != nil {
		return nil, err
	}
	if !isBatchPoster {
		if !createIfNotExist {
			return nil, ErrNotExist
		}
		return bpt.AddPoster(poster, poster)
	}
	return bpt.internalOpen(poster), nil
}

func (bpt *BatchPostersTable) internalOpen(poster common.Address) *BatchPosterState {
	bpStorage := bpt.posterInfo.OpenSubStorage(poster.Bytes())
	return &BatchPosterState{
		fundsDue:     bpStorage.OpenStorageBackedBigInt(0),
		payTo:        bpStorage.OpenStorageBackedAddress(1),
		postersTable: bpt,
	}
}

func (bpt *BatchPostersTable) ContainsPoster(poster common.Address) (bool, error) {
	return bpt.posterAddrs.IsMember(poster)
}

func (bpt *BatchPostersTable) AddPoster(posterAddress common.Address, payTo common.Address) (*BatchPosterState, error) {
	isBatchPoster, err := bpt.posterAddrs.IsMember(posterAddress)
	if err != nil {
		return nil, err
	}
	if isBatchPoster {
		return nil, ErrAlreadyExists
	}
	bpState := bpt.internalOpen(posterAddress)
	if err := bpState.fundsDue.SetChecked(common.Big0); err != nil {
		return nil, err
	}
	if err := bpState.payTo.Set(payTo); err != nil {
		return nil, err
	}

	if err := bpt.posterAddrs.Add(posterAddress); err != nil {
		return nil, err
	}

	return bpState, nil
}

func (bpt *BatchPostersTable) AllPosters(maxNumToGet uint64) ([]common.Address, error) {
	return bpt.posterAddrs.AllMembers(maxNumToGet)
}

func (bpt *BatchPostersTable) TotalFundsDue() (*big.Int, error) {
	return bpt.totalFundsDue.Get()
}

func (bps *BatchPosterState) FundsDue() (*big.Int, error) {
	return bps.fundsDue.Get()
}

func (bps *BatchPosterState) SetFundsDue(val *big.Int) error {
	fundsDue := bps.fundsDue
	totalFundsDue := bps.postersTable.totalFundsDue
	prev, err := fundsDue.Get()
	if err != nil {
		return err
	}
	prevTotal, err := totalFundsDue.Get()
	if err != nil {
		return err
	}
	if err := totalFundsDue.SetSaturatingWithWarning(arbmath.BigSub(arbmath.BigAdd(prevTotal, val), prev), "batch poster total funds due"); err != nil {
		return err
	}
	return bps.fundsDue.SetSaturatingWithWarning(val, "batch poster funds due")
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
	allPosters, err := bpt.AllPosters(math.MaxUint64)
	if err != nil {
		return nil, err
	}
	for _, posterAddr := range allPosters {
		poster, err := bpt.OpenPoster(posterAddr, false)
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
