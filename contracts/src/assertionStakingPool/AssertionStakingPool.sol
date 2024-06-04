// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "../rollup/IRollupLogic.sol";
import "./AbsBoldStakingPool.sol";
import "./interfaces/IAssertionStakingPool.sol";

/// @notice Staking pool contract for target assertion.
///
///         Allows users to deposit stake, create assertion once required stake amount is reached,
///         and reclaim their stake when and if the assertion is confirmed.
///
///         Tokens sent directly to this contract will be lost. 
///         It is assumed that the rollup will not return more tokens than the amount deposited by the pool.
///         Any tokens exceeding the deposited amount to the pool will be stuck in the pool forever.
contract AssertionStakingPool is AbsBoldStakingPool, IAssertionStakingPool {
    using SafeERC20 for IERC20;
    
    /// @inheritdoc IAssertionStakingPool
    address public immutable rollup;
    /// @inheritdoc IAssertionStakingPool
    bytes32 public immutable assertionHash;

    /// @param _rollup Rollup contract of target chain
    /// @param _assertionHash Assertion hash to be passed into Rollup.stakeOnNewAssertion
    constructor(
        address _rollup,
        bytes32 _assertionHash
    ) AbsBoldStakingPool(IRollupCore(_rollup).stakeToken()) {
        if(_assertionHash == bytes32(0)) {
            revert EmptyAssertionId();
        }
        rollup = _rollup;
        assertionHash = _assertionHash;
    }

    /// @inheritdoc IAssertionStakingPool
    function createAssertion(AssertionInputs calldata assertionInputs) external {
        uint256 requiredStake = assertionInputs.beforeStateData.configData.requiredStake;
        // approve spending from rollup for newStakeOnNewAssertion call
        IERC20(stakeToken).safeIncreaseAllowance(rollup, requiredStake);
        // reverts if pool doesn't have enough stake and if assertion has already been asserted
        IRollupUser(rollup).newStakeOnNewAssertion(requiredStake, assertionInputs, assertionHash, address(this));
    }

    /// @inheritdoc IAssertionStakingPool
    function makeStakeWithdrawable() public {
        // this checks for active staker
        IRollupUser(rollup).returnOldDeposit();
    }

    /// @inheritdoc IAssertionStakingPool
    function withdrawStakeBackIntoPool() public {
        IRollupUser(rollup).withdrawStakerFunds();
    }

    /// @inheritdoc IAssertionStakingPool
    function makeStakeWithdrawableAndWithdrawBackIntoPool() external {
        makeStakeWithdrawable();
        withdrawStakeBackIntoPool();
    }
}
