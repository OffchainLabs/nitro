// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

package l1pricing

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/offchainlabs/nitro/arbos/addressSet"
	"github.com/offchainlabs/nitro/arbos/storage"
	"github.com/offchainlabs/nitro/util/arbmath"
)

var (
	PosterAddrsKey = []byte{0}
	PosterInfoKey  = []byte{1}

	ErrAlreadyExists = errors.New("tried to add a batch poster that already exists")
)

// layout of storage in the table
type BatchPostersTable struct {
	posterAddrs *addressSet.AddressSet
	posterInfo  *storage.Storage
}

type BatchPosterState struct {
	fundsDue storage.StorageBackedBigInt
	payTo    storage.StorageBackedAddress
}

func InitializeBatchPostersTable(storage *storage.Storage) error {
	// no initialization needed for posterInfo
	return addressSet.Initialize(storage.OpenSubStorage(PosterAddrsKey))
}

func OpenBatchPostersTable(storage *storage.Storage) *BatchPostersTable {
	return &BatchPostersTable{
		posterAddrs: addressSet.OpenAddressSet(storage.OpenSubStorage(PosterAddrsKey)),
		posterInfo:  storage.OpenSubStorage(PosterInfoKey),
	}
}

func (bpt *BatchPostersTable) OpenPoster(poster common.Address) (*BatchPosterState, error) {
	isBatchPoster, err := bpt.posterAddrs.IsMember(poster)
	if err != nil {
		return nil, err
	}
	if !isBatchPoster {
		return bpt.AddPoster(poster, poster)
	}
	return bpt.internalOpen(poster), nil
}

func (bpt *BatchPostersTable) internalOpen(poster common.Address) *BatchPosterState {
	bpStorage := bpt.posterInfo.OpenSubStorage(poster.Bytes())
	return &BatchPosterState{
		fundsDue: bpStorage.OpenStorageBackedBigInt(0),
		payTo:    bpStorage.OpenStorageBackedAddress(1),
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
	if err := bpState.fundsDue.Set(common.Big0); err != nil {
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

func (bpt *BatchPostersTable) AllPosters() ([]common.Address, error) {
	return bpt.posterAddrs.AllMembers()
}

func (bpt *BatchPostersTable) TotalFundsDue() (*big.Int, error) {
	allPosters, err := bpt.AllPosters()
	if err != nil {
		return nil, err
	}
	ret := common.Big0
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
