// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./AbsBoldStakingPool.sol";
import "./interfaces/IEdgeStakingPool.sol";
import "../challengeV2/EdgeChallengeManager.sol";

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/// @notice Trustless staking pool contract for creating layer zero edges.
///
///         Allows users to deposit stake, create an edge once required stake amount is reached,
///         and reclaim their stake when and if the edge is confirmed.
///
///         Tokens sent directly to this contract will be lost.
///         It is assumed that the challenge manager will not return more tokens than the amount deposited by the pool.
///         Any tokens exceeding the deposited amount to the pool will be stuck in the pool forever.
///
/// @dev    Unlike the assertion staking pool, there is no need for a function to claim the stake back into the pool.
///         (see `EdgeChallengeManager.refundStake(bytes32 edgeId)`)
contract EdgeStakingPool is AbsBoldStakingPool, IEdgeStakingPool {
    using SafeERC20 for IERC20;

    /// @inheritdoc IEdgeStakingPool
    address public immutable challengeManager;
    /// @inheritdoc IEdgeStakingPool
    bytes32 public immutable edgeId;

    /// @param _challengeManager EdgeChallengeManager contract
    /// @param _edgeId The ID of the edge to be created (see ChallengeEdgeLib.id)
    constructor(
        address _challengeManager,
        bytes32 _edgeId
    ) AbsBoldStakingPool(address(EdgeChallengeManager(_challengeManager).stakeToken())) {
        if (_edgeId == bytes32(0)) {
            revert EmptyEdgeId();
        }
        challengeManager = _challengeManager;
        edgeId = _edgeId;
    }

    /// @inheritdoc IEdgeStakingPool
    function createEdge(CreateEdgeArgs calldata args) external {
        uint256 requiredStake = EdgeChallengeManager(challengeManager).stakeAmounts(args.level);
        IERC20(stakeToken).safeIncreaseAllowance(address(challengeManager), requiredStake);
        bytes32 newEdgeId = EdgeChallengeManager(challengeManager).createLayerZeroEdge(args);
        if (newEdgeId != edgeId) {
            revert IncorrectEdgeId(newEdgeId, edgeId);
        }
    }
}
