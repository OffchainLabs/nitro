//
// Copyright 2022, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/Proxy.sol";

contract SimpleProxy is Proxy {
    address private immutable impl;

    constructor(address impl_) {
        impl = impl_;
    }

    function _implementation() internal view override returns (address) {
        return impl;
    }
}
