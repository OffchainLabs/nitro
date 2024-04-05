// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "./AbsOutbox.sol";

contract Outbox is AbsOutbox {
    /// @inheritdoc AbsOutbox
    function _defaultContextAmount() internal pure override returns (uint256) {
        // In ETH-based chains withdrawal amount can be read from msg.value. For that reason
        // amount slot in context will never be accessed and it has 0 default value
        return 0;
    }

    /// @inheritdoc AbsOutbox
    function _amountToSetInContext(uint256) internal pure override returns (uint256) {
        // In ETH-based chains withdrawal amount can be read from msg.value. For that reason
        // amount slot in context will never be accessed, we keep it as 0 all the time
        return 0;
    }
}
