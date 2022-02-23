//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.4;

import "../libraries/EthCallAware.sol";

contract EthCallAwareTester is EthCallAware {

    event TxSuccess(uint256 num, bytes data);

    function testFunction(uint256 num, bytes calldata data) revertOnCall public {
        emit TxSuccess(num, data);
    }
}