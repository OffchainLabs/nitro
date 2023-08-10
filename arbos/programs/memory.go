// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"math"

	"github.com/offchainlabs/nitro/util/arbmath"
)

type MemoryModel struct {
	freePages uint16 // number of pages the tx gets for free
	pageGas   uint16 // base gas to charge per wasm page
}

func NewMemoryModel(freePages uint16, pageGas uint16) *MemoryModel {
	return &MemoryModel{
		freePages: freePages,
		pageGas:   pageGas,
	}
}

func (p Programs) memoryModel() (*MemoryModel, error) {
	freePages, err := p.FreePages()
	if err != nil {
		return nil, err
	}
	pageGas, err := p.PageGas()

	return NewMemoryModel(freePages, pageGas), err
}

// Determines the gas cost of allocating `new` pages given `open` are active and `ever` have ever been.
func (model *MemoryModel) GasCost(new, open, ever uint16) uint64 {
	newOpen := arbmath.SaturatingUAdd(open, new)
	newEver := arbmath.MaxInt(ever, newOpen)

	// free until expansion beyond the first few
	if newEver <= model.freePages {
		return 0
	}
	subFree := func(pages uint16) uint16 {
		return arbmath.SaturatingUSub(pages, model.freePages)
	}

	adding := arbmath.SaturatingUSub(subFree(newOpen), subFree(open))
	linear := arbmath.SaturatingUMul(uint64(adding), uint64(model.pageGas))
	expand := model.exp(newEver) - model.exp(ever)
	return arbmath.SaturatingUAdd(linear, expand)
}

func (model *MemoryModel) exp(pages uint16) uint64 {
	if int(pages) < len(memoryExponents) {
		return uint64(memoryExponents[pages])
	}
	return math.MaxUint64
}

var memoryExponents = [129]uint32{
	1, 1, 1, 1, 1, 1, 2, 2, 2, 3, 3, 4, 5, 5, 6, 7, 8, 9, 11, 12, 14, 17, 19, 22, 25, 29, 33, 38,
	43, 50, 57, 65, 75, 85, 98, 112, 128, 147, 168, 193, 221, 253, 289, 331, 379, 434, 497, 569,
	651, 745, 853, 976, 1117, 1279, 1463, 1675, 1917, 2194, 2511, 2874, 3290, 3765, 4309, 4932,
	5645, 6461, 7395, 8464, 9687, 11087, 12689, 14523, 16621, 19024, 21773, 24919, 28521, 32642,
	37359, 42758, 48938, 56010, 64104, 73368, 83971, 96106, 109994, 125890, 144082, 164904, 188735,
	216010, 247226, 282953, 323844, 370643, 424206, 485509, 555672, 635973, 727880, 833067, 953456,
	1091243, 1248941, 1429429, 1636000, 1872423, 2143012, 2452704, 2807151, 3212820, 3677113,
	4208502, 4816684, 5512756, 6309419, 7221210, 8264766, 9459129, 10826093, 12390601, 14181199,
	16230562, 18576084, 21260563, 24332984, 27849408, 31873999,
}
