// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import {DoubleLogicERC1967Upgrade} from "./AdminFallbackProxy.sol";
import "@openzeppelin/contracts/proxy/utils/UUPSUpgradeable.sol";

/// @notice An extension to OZ's UUPSUpgradeable contract to be used for handling UUPS upgrades with a DoubleLogicERC1967Upgrade proxy
///         The should be used in the primary implementation slot of the DoubleLogicUUPS proxy
/// @dev upgrades should be handles by the primary logic contract in order to pass the `onlyProxy` check
abstract contract DoubleLogicUUPSUpgradeable is UUPSUpgradeable, DoubleLogicERC1967Upgrade {
    /// @inheritdoc UUPSUpgradeable
    function proxiableUUID() external view override notDelegated returns (bytes32) {
        return _IMPLEMENTATION_SLOT;
    }

    /**
     * @dev Function that should revert when `msg.sender` is not authorized to upgrade the secondary contract. Called by
     * {upgradeSecondaryTo} and {upgradeSecondaryToAndCall}.
     *
     * Normally, this function will use an xref:access.adoc[access control] modifier such as {Ownable-onlyOwner}.
     *
     * ```solidity
     * function _authorizeSecondaryUpgrade(address) internal override onlyOwner {}
     * ```
     */
    function _authorizeSecondaryUpgrade(address newImplementation) internal virtual;

    /**
     * @dev Upgrade the secondary implementation of the proxy to `newImplementation`.
     *
     * Calls {_authorizeSecondaryUpgrade}.
     *
     * Emits an {UpgradedSecondary} event.
     */
    function upgradeSecondaryTo(address newImplementation) external onlyProxy {
        _authorizeSecondaryUpgrade(newImplementation);
        _upgradeSecondaryToAndCallUUPS(newImplementation, new bytes(0), false);
    }

    /**
     * @dev Upgrade the secondary implementation of the proxy to `newImplementation`, and subsequently execute the function call
     * encoded in `data`.
     *
     * Calls {_authorizeSecondaryUpgrade}.
     *
     * Emits an {UpgradedSecondary} event.
     */
    function upgradeSecondaryToAndCall(address newImplementation, bytes memory data)
        external
        payable
        onlyProxy
    {
        _authorizeSecondaryUpgrade(newImplementation);
        _upgradeSecondaryToAndCallUUPS(newImplementation, data, true);
    }
}
