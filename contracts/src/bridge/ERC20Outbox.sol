// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./AbsOutbox.sol";

contract ERC20Outbox is AbsOutbox {
    uint256 private constant AMOUNT_DEFAULT_CONTEXT = type(uint256).max;

    function l2ToL1WithdrawalAmount() external view returns (uint256) {
        uint256 amount = context.withdrawalAmount;
        if (amount == AMOUNT_DEFAULT_CONTEXT) return 0;
        return amount;
    }

    function _defaultContextAmount() internal pure override returns (uint256) {
        return AMOUNT_DEFAULT_CONTEXT;
    }

    function _amountToSetInContext(uint256 value) internal pure override returns (uint256) {
        return value;
    }
}
