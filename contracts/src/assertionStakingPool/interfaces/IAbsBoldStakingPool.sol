// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

interface IAbsBoldStakingPool {
    /// @notice Emitted when stake is deposited
    event StakeDeposited(address indexed sender, uint256 amount);
    /// @notice Emitted when stake is withdrawn
    event StakeWithdrawn(address indexed sender, uint256 amount);

    /// @notice Cannot deposit or withdraw zero amount
    error ZeroAmount();
    /// @notice Withdraw amount exceeds balance
    error AmountExceedsBalance(address account, uint256 amount, uint256 balance);

    /// @notice Deposit stake into pool contract.
    /// @param amount amount of stake token to deposit
    function depositIntoPool(uint256 amount) external;

    /// @notice Send supplied amount of stake from this contract back to its depositor.
    /// @param amount stake amount to withdraw
    function withdrawFromPool(uint256 amount) external;

    /// @notice Send full balance of stake from this contract back to its depositor.
    function withdrawFromPool() external;

    /// @notice The token that is used for staking
    function stakeToken() external view returns (address);

    /// @notice The balance of the given account
    /// @param account The account to check the balance of
    function depositBalance(address account) external view returns (uint256);
}
