// Copyright 2024-2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract SelfDestructInConstructorWithoutDestination {
    constructor() public payable {
        selfdestruct(payable(address(this)));
    }
}

contract SelfDestructInConstructorWithDestination {
    constructor(
        address payable destination
    ) public payable {
        selfdestruct(destination);
    }
}

contract SelfDestructOutsideConstructor {
    constructor() public payable {}

    function selfDestructWithDestination(
        address payable destination
    ) public {
        selfdestruct(destination);
    }

    function selfDestructWithoutDestination() public {
        selfdestruct(payable(address(this)));
    }
}
