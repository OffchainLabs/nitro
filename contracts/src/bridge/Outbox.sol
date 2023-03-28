// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./AbsOutbox.sol";

contract Outbox is AbsOutbox {
    /// @inheritdoc AbsOutbox
    function _defaultContextAmount() internal pure override returns (uint256) {
        return 0;
    }

    /// @inheritdoc AbsOutbox
    function _amountToSetInContext(uint256) internal pure override returns (uint256) {
        return 0;
    }
}
