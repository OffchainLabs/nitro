// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/constraints"
)

type resourceWeight struct {
	ResourceKind multigas.ResourceKind
	Weight       uint64
}

type resourceConstraint struct {
	ResourceWeight []resourceWeight
	PeriodSecs     uint32
	TargetPerSec   uint64
}

func toArbOsResourceSet(resources []resourceWeight) (constraints.ResourceSet, error) {
	rs := constraints.EmptyResourceSet()
	if len(resources) == 0 {
		return constraints.ResourceSet{}, fmt.Errorf("at least one resource is required")
	}
	for _, r := range resources {
		if r.Weight == 0 || r.Weight > constraints.MaxResourceWeight {
			return constraints.ResourceSet{}, fmt.Errorf("resource weight for kind %d must be in range [1, %d]", r.ResourceKind, constraints.MaxResourceWeight)
		}
		if rs.HasResource(r.ResourceKind) {
			return constraints.ResourceSet{}, fmt.Errorf("duplicate resource kind %d", r.ResourceKind)
		}
		rs = rs.WithResource(r.ResourceKind, constraints.ResourceWeight(r.Weight))
	}
	return rs, nil
}

func fromArbOsResourceConstraint(rc *constraints.ResourceConstraint) resourceConstraint {
	weights := make([]resourceWeight, 0)
	for rk, w := range rc.Resources.All() {
		weights = append(weights, resourceWeight{
			ResourceKind: rk,
			Weight:       uint64(w),
		})
	}
	return resourceConstraint{
		ResourceWeight: weights,
		PeriodSecs:     uint32(rc.Period),
		TargetPerSec:   rc.TargetPerSec,
	}
}

func fromArbOsResourceConstraints(rcs []*constraints.ResourceConstraint) []resourceConstraint {
	result := make([]resourceConstraint, 0, len(rcs))
	for _, rc := range rcs {
		result = append(result, fromArbOsResourceConstraint(rc))
	}
	return result
}
