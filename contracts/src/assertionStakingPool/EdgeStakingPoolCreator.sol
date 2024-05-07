// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//

pragma solidity ^0.8.0;

import "./EdgeStakingPool.sol";
import "./StakingPoolCreatorUtils.sol";
import "./interfaces/IEdgeStakingPoolCreator.sol";

/// @notice Creates EdgeStakingPool contracts.
contract EdgeStakingPoolCreator is IEdgeStakingPoolCreator {
    /// @inheritdoc IEdgeStakingPoolCreator
    function createPool(
        address challengeManager,
        bytes32 edgeId
    ) external returns (IEdgeStakingPool) {
        EdgeStakingPool pool = new EdgeStakingPool{salt: 0}(challengeManager, edgeId);
        emit NewEdgeStakingPoolCreated(challengeManager, edgeId);
        return pool;
    }

    /// @inheritdoc IEdgeStakingPoolCreator
    function getPool(
        address challengeManager,
        bytes32 edgeId
    ) public view returns (IEdgeStakingPool) {
        return IEdgeStakingPool(
            StakingPoolCreatorUtils.getPool(
                type(EdgeStakingPool).creationCode,
                abi.encode(challengeManager, edgeId)
            )
        );
    }
}
