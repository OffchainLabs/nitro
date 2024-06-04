// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../../challengeV2/EdgeChallengeManager.sol";
import "./IAbsBoldStakingPool.sol";

interface IEdgeStakingPool is IAbsBoldStakingPool {
    /// @notice The resulting edge does not match the expected edge
    error IncorrectEdgeId(bytes32 actual, bytes32 expected);

    /// @notice Thrown when edge id is empty
    error EmptyEdgeId();

    /// @notice Create the edge. Callable only if required stake has been reached and edge has not been created yet.
    function createEdge(CreateEdgeArgs calldata args) external;

    /// @notice The targeted challenge manager contract
    function challengeManager() external view returns (address);
    
    /// @notice The edge that this pool will create
    function edgeId() external view returns (bytes32);
}
