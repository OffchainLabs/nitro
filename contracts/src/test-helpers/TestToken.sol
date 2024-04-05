// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * Basic ERC20 token
 */
contract TestToken is ERC20 {
    constructor(uint256 initialSupply) ERC20("TestToken", "TT") {
        _mint(msg.sender, initialSupply);
    }
}
