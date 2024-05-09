// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./IEdgeStakingPool.sol";

interface IEdgeStakingPoolCreator {
    /// @notice Event emitted when a new staking pool is created
    event NewEdgeStakingPoolCreated(address indexed challengeManager, bytes32 indexed edgeId);

    /// @notice Create an edge staking pool contract
    /// @param challengeManager EdgeChallengeManager contract
    /// @param edgeId The ID of the edge to be created (see ChallengeEdgeLib.id)
    function createPool(
        address challengeManager,
        bytes32 edgeId
    ) external returns (IEdgeStakingPool);

    /// @notice get staking pool deployed with provided inputs; reverts if pool contract doesn't exist.
    /// @param challengeManager EdgeChallengeManager contract
    /// @param edgeId The ID of the edge to be created (see ChallengeEdgeLib.id)
    function getPool(
        address challengeManager,
        bytes32 edgeId
    ) external view returns (IEdgeStakingPool);
}
