// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.0;
import "../rollup/IRollupLogic.sol";

error NoBalanceToWithdraw(address sender);

error PoolDoesntExist(address rollup, AssertionInputs assertionInputs, bytes32 assertionHash);

error AmountExceedsBalance(address sender, uint256 amount, uint256 balance);
