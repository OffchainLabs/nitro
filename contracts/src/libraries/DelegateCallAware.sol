// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {NotOwner} from "./Error.sol";

/// @dev A stateless contract that allows you to infer if the current call has been delegated or not
/// Pattern used here is from UUPS implementation by the OpenZeppelin team
abstract contract DelegateCallAware {
    address private immutable __self = address(this);

    /**
     * @dev Check that the execution is being performed through a delegate call. This allows a function to be
     * callable on the proxy contract but not on the logic contract.
     */
    modifier onlyDelegated() {
        require(address(this) != __self, "Function must be called through delegatecall");
        _;
    }

    /**
     * @dev Check that the execution is not being performed through a delegate call. This allows a function to be
     * callable on the implementing contract but not through proxies.
     */
    modifier notDelegated() {
        require(address(this) == __self, "Function must not be called through delegatecall");
        _;
    }

    /// @dev Check that msg.sender is the current EIP 1967 proxy admin
    modifier onlyProxyOwner() {
        // Storage slot with the admin of the proxy contract
        // This is the keccak-256 hash of "eip1967.proxy.admin" subtracted by 1
        bytes32 slot = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;
        address admin;
        assembly {
            admin := sload(slot)
        }
        if (msg.sender != admin) revert NotOwner(msg.sender, admin);
        _;
    }
}
