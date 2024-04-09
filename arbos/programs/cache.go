// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import "github.com/ethereum/go-ethereum/common"

type RecentPrograms struct {
	queue []common.Hash
	items map[common.Hash]struct{}
}

func NewRecentProgramsTracker() *RecentPrograms {
	return &RecentPrograms{
		queue: make([]common.Hash, 0, initialTxCacheSize),
		items: make(map[common.Hash]struct{}, initialTxCacheSize),
	}
}

func (p *RecentPrograms) Insert(item common.Hash, params *StylusParams) bool {
	if _, ok := p.items[item]; ok {
		return true
	}
	p.queue = append(p.queue, item)
	p.items[item] = struct{}{}

	if len(p.queue) > int(params.TxCacheSize) {
		p.queue = p.queue[1:]
		delete(p.items, item)
	}
	return false
}
