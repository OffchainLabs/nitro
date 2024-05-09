// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {IERC20} from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import {SafeERC20} from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import {IAbsBoldStakingPool} from "./interfaces/IAbsBoldStakingPool.sol";

/// @notice Abstract contract for handling deposits and withdrawals of trustless edge/assertion staking pools.
/// @dev    The total deposited amount can exceed the required stake amount. 
///         If the total deposited amount exceeds the required amount, any depositor can withdraw some stake early even after the protocol move has been made.
///         This is okay because the protocol move will still be created once the required stake amount is reached, 
///         and all depositors will still be eventually refunded.
abstract contract AbsBoldStakingPool is IAbsBoldStakingPool {
    using SafeERC20 for IERC20;

    /// @inheritdoc IAbsBoldStakingPool
    address public immutable stakeToken;
    /// @inheritdoc IAbsBoldStakingPool
    mapping(address => uint256) public depositBalance;

    constructor(address _stakeToken) {
        stakeToken = _stakeToken;
    }

    /// @inheritdoc IAbsBoldStakingPool
    function depositIntoPool(uint256 amount) external {
        if (amount == 0) {
            revert ZeroAmount();
        }

        depositBalance[msg.sender] += amount;
        IERC20(stakeToken).safeTransferFrom(msg.sender, address(this), amount);

        emit StakeDeposited(msg.sender, amount);
    }

    /// @inheritdoc IAbsBoldStakingPool
    function withdrawFromPool(uint256 amount) public {
        if (amount == 0) {
            revert ZeroAmount();
        }
        uint256 balance = depositBalance[msg.sender];
        if (amount > balance) {
            revert AmountExceedsBalance(msg.sender, amount, balance);
        }

        depositBalance[msg.sender] = balance - amount;
        IERC20(stakeToken).safeTransfer(msg.sender, amount);
        
        emit StakeWithdrawn(msg.sender, amount);
    }

    /// @inheritdoc IAbsBoldStakingPool
    function withdrawFromPool() external {
        withdrawFromPool(depositBalance[msg.sender]);
    }
}
