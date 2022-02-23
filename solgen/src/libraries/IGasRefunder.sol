//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity >=0.6.11 <0.9.0;

interface IGasRefunder {
    function onGasSpent(
        address payable spender,
        uint256 gasUsed,
        uint256 calldataSize
    ) external returns (bool success);
}

abstract contract GasRefundEnabled {
    modifier refundsGasWithCalldata(IGasRefunder gasRefunder, address payable spender) {
        uint256 startGasLeft = gasleft();
        _;
        if (address(gasRefunder) != address(0)) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            gasRefunder.onGasSpent(spender, startGasLeft - gasleft(), calldataSize);
        }
    }

    modifier refundsGasNoCalldata(IGasRefunder gasRefunder, address payable spender) {
        uint256 startGasLeft = gasleft();
        _;
        if (address(gasRefunder) != address(0)) {
            gasRefunder.onGasSpent(spender, startGasLeft - gasleft(), 0);
        }
    }
}
