// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./AbsOutbox.sol";

contract ERC20Outbox is AbsOutbox {
    // it is assumed that arb-os never assigns this value to a valid leaf to be redeemed
    uint256 private constant AMOUNT_DEFAULT_CONTEXT = type(uint256).max;

    function l2ToL1WithdrawalAmount() external view returns (uint256) {
        uint256 amount = context.withdrawalAmount;
        if (amount == AMOUNT_DEFAULT_CONTEXT) return 0;
        return amount;
    }

    /// @inheritdoc AbsOutbox
    function _defaultContextAmount() internal pure override returns (uint256) {
        // we use type(uint256).max as representation of 0 native token withdrawal amount
        return AMOUNT_DEFAULT_CONTEXT;
    }

    /// @inheritdoc AbsOutbox
    function _amountToSetInContext(uint256 value) internal pure override returns (uint256) {
        // native token withdrawal amount which can be fetched from context
        return value;
    }
}
