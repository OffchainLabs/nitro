// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";

import "../libraries/DelegateCallAware.sol";

/**
 * @title Manage keysets for distributed Data Availability Service
 * @notice Manages the valid keysets for distributed Data Availability Service
 */
contract DasKeysetManager is OwnableUpgradeable, DelegateCallAware {
    using AddressUpgradeable for address;

    uint256 public constant MAX_VALID_KEYSETS = 16;
    bytes32[] private validKeysets;
    mapping(bytes32 => uint256) private validKeysetIndex; // if keyset is at index in validKeysets, this is index+1
    mapping(bytes32 => bytes) private contents;

    function initialize() external initializer onlyDelegated {
        __Ownable_init();
    }

    function isValidKeysetHash(bytes32 ksHash) external view returns (bool) {
        return validKeysetIndex[ksHash] != 0;
    }

    function addKeyset(bytes calldata keyset) external onlyOwner {
        require(validKeysets.length >= MAX_VALID_KEYSETS, "too many keysets");
        bytes32 ksHash = keccak256(keyset);
        if (this.isValidKeysetHash(ksHash)) {
            return;
        }
        validKeysetIndex[ksHash] = validKeysets.length + 1;
        validKeysets.push(ksHash);
        contents[ksHash] = keyset;
    }

    function invalidateKeyset(bytes32 ksHash) external onlyOwner {
        require(this.isValidKeysetHash(ksHash), "keyset is not valid");
        uint256 slot = validKeysetIndex[ksHash] - 1;
        if (slot != validKeysets.length - 1) {
            bytes32 lastItem = validKeysets[validKeysets.length - 1];
            validKeysets[slot] = lastItem;
            validKeysetIndex[lastItem] = slot + 1;
        }
        validKeysets.pop();

        // do NOT delete the keyset from the contents mapping, because validators might need to retrieve it later
    }

    function getKeysetByHash(bytes32 ksHash) external view returns (bytes memory) {
        return contents[ksHash];
    }

    function allValidKeysetHashes() external view returns (bytes32[] memory) {
        return validKeysets;
    }
}
