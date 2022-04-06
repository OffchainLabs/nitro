//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "../libraries/EthCallAware.sol";

contract EthCallAwareTester {
    event TxSuccess(uint256 num, bytes data);

    function testFunction(
        uint256 num,
        bytes calldata data,
        bool skip
    ) public {
        if (!skip) EthCallAware.revertOnCall(data);
        emit TxSuccess(num, data);
    }
}
