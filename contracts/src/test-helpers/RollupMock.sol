// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

contract RollupMock {
    event WithdrawTriggered();
    event ZombieTriggered();

    function withdrawStakerFunds() external returns (uint256) {
        emit WithdrawTriggered();
        return 0;
    }

    function removeOldZombies(
        uint256 /* startIndex */
    ) external {
        emit ZombieTriggered();
    }
}
