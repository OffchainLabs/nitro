// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../../rollup/IRollupLogic.sol";
import "./IAssertionStakingPool.sol";

interface IAssertionStakingPoolCreator {
    /// @notice Event emitted when a new staking pool is created
    event NewAssertionPoolCreated(
        address indexed rollup,
        bytes32 indexed _assertionHash,
        address assertionPool
    );

    /// @notice Create a staking pool contract
    /// @param _rollup Rollup contract of target chain
    /// @param _assertionHash Assertion hash to be passed into Rollup.stakeOnNewAssertion
    function createPool(
        address _rollup,
        bytes32 _assertionHash
    ) external returns (IAssertionStakingPool);

    /// @notice get staking pool deployed with provided inputs; reverts if pool contract doesn't exist.
    /// @param _rollup Rollup contract of target chain
    /// @param _assertionHash Assertion hash to be passed into Rollup.stakeOnNewAssertion
    function getPool(
        address _rollup,
        bytes32 _assertionHash
    ) external view returns (IAssertionStakingPool);
}
