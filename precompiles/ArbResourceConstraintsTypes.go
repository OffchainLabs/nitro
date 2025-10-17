// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package precompiles

import (
	"fmt"

	"github.com/ethereum/go-ethereum/arbitrum/multigas"

	"github.com/offchainlabs/nitro/arbos/constraints"
)

type resourceWeight = struct {
	Resource uint8  `json:"resource"`
	Weight   uint64 `json:"weight"`
}
type resourceConstraint = struct {
	Resources    []resourceWeight `json:"resources"`
	PeriodSecs   uint32           `json:"periodSecs"`
	TargetPerSec uint64           `json:"targetPerSec"`
}

func fromArbOsResourceConstraints(rcs []*constraints.ResourceConstraint) []resourceConstraint {
	var out []resourceConstraint
	for _, rc := range rcs {
		var res []resourceWeight
		for r, w := range rc.Resources.All() {
			res = append(res, resourceWeight{
				Resource: uint8(r),
				Weight:   uint64(w),
			})
		}

		out = append(out, resourceConstraint{
			Resources:    res,
			PeriodSecs:   uint32(rc.Period),
			TargetPerSec: rc.TargetPerSec,
		})
	}
	return out
}

func toArbOsWeightedResourceSet(resources []resourceWeight) (constraints.WeightedResourceSet, error) {
	rs := constraints.NewWeightedResourceSet()
	if len(resources) == 0 {
		return constraints.WeightedResourceSet{}, fmt.Errorf("at least one resource is required")
	}
	for _, r := range resources {
		if r.Weight == 0 || r.Weight > constraints.MaxResourceWeight {
			return constraints.WeightedResourceSet{}, fmt.Errorf("resource weight must be in range [1, %d]", constraints.MaxResourceWeight)
		}
		resourceKind, err := multigas.CheckResourceKind(r.Resource)
		if err != nil {
			return constraints.WeightedResourceSet{}, err
		}

		if rs.HasResource(resourceKind) {
			return constraints.WeightedResourceSet{}, fmt.Errorf("duplicate resource kind %d", resourceKind)
		}
		rs = rs.WithResource(resourceKind, constraints.ResourceWeight(r.Weight))
	}
	return rs, nil
}

func toArbOsResourceSet(resources []uint8) (constraints.ResourceSet, error) {
	res := constraints.EmptyResourceSet()
	for _, resource := range resources {
		kind, err := multigas.CheckResourceKind(resource)
		if err != nil {
			return constraints.ResourceSet{}, err
		}
		res = res.WithResource(kind)
	}
	return res, nil
}
