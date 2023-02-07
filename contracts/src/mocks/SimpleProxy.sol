// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

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
