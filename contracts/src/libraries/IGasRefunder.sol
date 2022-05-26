// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.6.11 <0.9.0;

interface IGasRefunder {
    function onGasSpent(
        address payable spender,
        uint256 gasUsed,
        uint256 calldataSize
    ) external returns (bool success);
}

abstract contract GasRefundEnabled {
    function getCalldataSize() private view returns (uint256 calldataSize) {
        assembly {
            calldataSize := calldatasize()
        }
    }

    /// @dev this method assumes that the spender was charged calldata as part of the tx input
    /// if triggered in a contract call, the spender may be overrefunded by appending dummy data to the call
    modifier refundsGas(IGasRefunder gasRefunder) {
        uint256 startGasLeft = gasleft();
        _;
        if (address(gasRefunder) != address(0)) {
            uint256 calldataSize = msg.sender == tx.origin ? getCalldataSize() : 0; 
            gasRefunder.onGasSpent(payable(msg.sender), startGasLeft - gasleft(), calldataSize);
        }
    }
}
