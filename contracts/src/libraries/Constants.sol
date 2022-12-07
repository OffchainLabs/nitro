// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

// 90% of Geth's 128KB tx size limit, leaving ~13KB for proving
uint256 constant MAX_DATA_SIZE = 117964;

uint64 constant NO_CHAL_INDEX = 0;

// Expected seconds per block in Ethereum PoS
uint256 constant ETH_POS_BLOCK_TIME = 12;

// This address call Inbox.createRetryableTicket expecting unsafe behavior
// because it is deployed before the Inbox callvalue check is implemented
address constant UNSAFE_CREATERETRYABLETICKET_CALLER = 0x4dC25eA85FAD2F578685A4d8E404C12164eA405B;
