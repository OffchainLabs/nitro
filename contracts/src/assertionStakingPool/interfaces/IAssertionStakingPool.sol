// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../../rollup/IRollupLogic.sol";
import "./IAbsBoldStakingPool.sol";

interface IAssertionStakingPool is IAbsBoldStakingPool {
    /// @notice Create assertion. Callable only if required stake has been reached and assertion has not been asserted yet.
    function createAssertion(AssertionInputs calldata assertionInputs) external;

    /// @notice Make stake withdrawable.
    /// @dev Separate call from withdrawStakeBackIntoPool since returnOldDeposit reverts with 0 balance (in e.g., case of admin forceRefundStaker)
    function makeStakeWithdrawable() external;

    /// @notice Move stake back from rollup contract to this contract.
    /// Callable only if this contract has already created an assertion and it's now inactive.
    /// @dev Separate call from makeStakeWithdrawable since returnOldDeposit reverts with 0 balance (in e.g., case of admin forceRefundStaker)
    function withdrawStakeBackIntoPool() external;

    /// @notice Combines makeStakeWithdrawable and withdrawStakeBackIntoPool into single call
    function makeStakeWithdrawableAndWithdrawBackIntoPool() external;

    /// @notice The targeted rollup contract
    function rollup() external view returns (address);

    /// @notice The assertion hash that this pool creates
    function assertionHash() external view returns (bytes32);
}
