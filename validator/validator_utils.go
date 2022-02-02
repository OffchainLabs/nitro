//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package validator

import (
	"context"
	"math/big"

	"github.com/offchainlabs/arbstate/arbutil"
	"github.com/offchainlabs/arbstate/solgen/go/rollupgen"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

type ConfirmType uint8

const (
	CONFIRM_TYPE_NONE ConfirmType = iota
	CONFIRM_TYPE_VALID
	CONFIRM_TYPE_INVALID
)

type ConflictType uint8

const (
	CONFLICT_TYPE_NONE ConflictType = iota
	CONFLICT_TYPE_FOUND
	CONFLICT_TYPE_INDETERMINATE
	CONFLICT_TYPE_INCOMPLETE
)

type ValidatorUtils struct {
	con           *rollupgen.ValidatorUtils
	client        arbutil.L1Interface
	address       common.Address
	rollupAddress common.Address
	baseCallOpts  bind.CallOpts
}

func NewValidatorUtils(address, rollupAddress common.Address, client arbutil.L1Interface, callOpts bind.CallOpts) (*ValidatorUtils, error) {
	con, err := rollupgen.NewValidatorUtils(address, client)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &ValidatorUtils{
		con:           con,
		client:        client,
		address:       address,
		rollupAddress: rollupAddress,
		baseCallOpts:  callOpts,
	}, nil
}

func (v *ValidatorUtils) getCallOpts(ctx context.Context) *bind.CallOpts {
	opts := v.baseCallOpts
	opts.Context = ctx
	return &opts
}

func (v *ValidatorUtils) RefundableStakers(ctx context.Context) ([]common.Address, error) {
	addresses, err := v.con.RefundableStakers(v.getCallOpts(ctx), v.rollupAddress)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return addresses, nil
}

func (v *ValidatorUtils) TimedOutChallenges(ctx context.Context, max int) ([]common.Address, error) {
	var count uint64 = 1024
	addresses := make([]common.Address, 0)
	for i := uint64(0); ; i += count {
		newAddrs, hasMore, err := v.con.TimedOutChallenges(v.getCallOpts(ctx), v.rollupAddress, i, count)
		addresses = append(addresses, newAddrs...)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if !hasMore {
			break
		}
		if len(addresses) >= max {
			break
		}
	}
	if len(addresses) > max {
		addresses = addresses[:max]
	}
	return addresses, nil
}

type RollupConfig struct {
	ConfirmPeriodBlocks      *big.Int
	ExtraChallengeTimeBlocks *big.Int
	ArbGasSpeedLimitPerBlock *big.Int
	BaseStake                *big.Int
	StakeToken               common.Address
}

func (v *ValidatorUtils) GetStakers(ctx context.Context) ([]common.Address, error) {
	addresses, _, err := v.con.GetStakers(v.getCallOpts(ctx), v.rollupAddress, 0, ^uint64(0))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return addresses, nil
}

func (v *ValidatorUtils) LatestStaked(ctx context.Context, staker common.Address) (uint64, [32]byte, error) {
	amount, hash, err := v.con.LatestStaked(v.getCallOpts(ctx), v.rollupAddress, staker)
	return amount, hash, errors.WithStack(err)
}

func (v *ValidatorUtils) StakedNodes(ctx context.Context, staker common.Address) ([]uint64, error) {
	nodes, err := v.con.StakedNodes(v.getCallOpts(ctx), v.rollupAddress, staker)
	return nodes, errors.WithStack(err)
}

func (v *ValidatorUtils) AreUnresolvedNodesLinear(ctx context.Context) (bool, error) {
	linear, err := v.con.AreUnresolvedNodesLinear(v.getCallOpts(ctx), v.rollupAddress)
	return linear, errors.WithStack(err)
}

func (v *ValidatorUtils) CheckDecidableNextNode(ctx context.Context) (ConfirmType, error) {
	confirmType, err := v.con.CheckDecidableNextNode(
		v.getCallOpts(ctx),
		v.rollupAddress,
	)
	if err != nil {
		return CONFIRM_TYPE_NONE, errors.WithStack(err)
	}
	return ConfirmType(confirmType), nil
}

func (v *ValidatorUtils) FindStakerConflict(ctx context.Context, staker1, staker2 common.Address) (ConflictType, uint64, uint64, error) {
	conflictType, staker1Node, staker2Node, err := v.con.FindStakerConflict(
		v.getCallOpts(ctx),
		v.rollupAddress,
		staker1,
		staker2,
		math.MaxBig256,
	)
	if err != nil {
		return CONFLICT_TYPE_NONE, 0, 0, errors.WithStack(err)
	}
	for ConflictType(conflictType) == CONFLICT_TYPE_INCOMPLETE {
		conflictType, staker1Node, staker2Node, err = v.con.FindNodeConflict(
			v.getCallOpts(ctx),
			v.rollupAddress,
			staker1Node,
			staker2Node,
			math.MaxBig256,
		)
		if err != nil {
			return CONFLICT_TYPE_NONE, 0, 0, errors.WithStack(err)
		}
	}
	return ConflictType(conflictType), staker1Node, staker2Node, nil
}
