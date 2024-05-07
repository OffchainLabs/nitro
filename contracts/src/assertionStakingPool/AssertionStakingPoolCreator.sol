// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//

pragma solidity ^0.8.0;

import "./AssertionStakingPool.sol";
import "./StakingPoolCreatorUtils.sol";
import "./interfaces/IAssertionStakingPoolCreator.sol";

/// @notice Creates staking pool contract for a target assertion. Can be used for any child Arbitrum chain running on top of the deployed AssertionStakingPoolCreator's chain.
contract AssertionStakingPoolCreator is IAssertionStakingPoolCreator {
    /// @inheritdoc IAssertionStakingPoolCreator
    function createPool(
        address _rollup,
        bytes32 _assertionHash
    ) external returns (IAssertionStakingPool) {
        AssertionStakingPool assertionPool = new AssertionStakingPool{salt: 0}(_rollup, _assertionHash);
        emit NewAssertionPoolCreated(_rollup, _assertionHash, address(assertionPool));
        return assertionPool;
    }

    /// @inheritdoc IAssertionStakingPoolCreator
    function getPool(
        address _rollup,
        bytes32 _assertionHash
    ) public view returns (IAssertionStakingPool) {
        return IAssertionStakingPool(
            StakingPoolCreatorUtils.getPool(
                type(AssertionStakingPool).creationCode, 
                abi.encode(_rollup, _assertionHash)
            )
        );
    }
}
