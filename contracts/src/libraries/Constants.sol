// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

// 90% of Geth's 128KB tx size limit, leaving ~13KB for proving
uint256 constant MAX_DATA_SIZE = 117964;

uint64 constant NO_CHAL_INDEX = 0;

// Expected seconds per block in Ethereum PoS
uint256 constant ETH_POS_BLOCK_TIME = 12;

address constant UNISWAP_L1_TIMELOCK = 0x1a9C8182C09F50C8318d769245beA52c32BE35BC;
address constant UNISWAP_L2_FACTORY = 0x1F98431c8aD98523631AE4a59f267346ea31F984;