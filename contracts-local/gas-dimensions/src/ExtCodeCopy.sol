// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Counter} from "./Counter.sol";

contract ExtCodeCopy {
    bytes32 knownHash;
    Counter counter;
    Counter coldCounter;

    constructor() {
        counter = new Counter();
        coldCounter = new Counter();
    }

    function extCodeCopyWarmNoMemExpansion() public returns (bytes32 codeHash) {
        uint256 codeSize;
        address contractAddress = address(counter);
        assembly {
            codeSize := extcodesize(contractAddress) // forces warm access state
        }

        // Create a much larger buffer to force memory expansion ahead of time
        bytes memory localCode = new bytes(codeSize);

        assembly {
            // should not trigger memory expansion
            extcodecopy(
                contractAddress, // address to copy from
                add(localCode, 32), // skip first 32 bytes which contain length
                0, // start reading from this position in the code
                codeSize // number of bytes to copy
            )
        }

        // Hash the local code
        codeHash = keccak256(localCode);
        knownHash = codeHash;
    }

    function extCodeCopyColdNoMemExpansion() public returns (bytes32 codeHash) {
        uint256 codeSize;
        address contractAddress = address(counter);
        assembly {
            codeSize := extcodesize(contractAddress)
        }

        address coldAddress = address(coldCounter);

        // Create a much larger buffer to force memory expansion ahead of time
        bytes memory localCode = new bytes(codeSize);

        assembly {
            // should not trigger memory expansion
            extcodecopy(
                coldAddress, // address to copy from
                add(localCode, 32), // skip first 32 bytes which contain length
                0, // start reading from this position in the code
                codeSize // number of bytes to copy
            )
        }

        // Hash the local code
        codeHash = keccak256(localCode);
        knownHash = codeHash;
    }

    function extCodeCopyWarmMemExpansion() public returns (bytes32 codeHash) {
        uint256 codeSize;
        address contractAddress = address(counter);
        assembly {
            codeSize := extcodesize(contractAddress) // forces warm access state
        }
        bytes memory localCode = new bytes(codeSize);
        assembly {
            let mSize := msize()
            extcodecopy(
                contractAddress, // address to copy from
                sub(mSize, 1), // place it in the last few bytes of memory to force expansion
                0, // start reading from this position in the code
                codeSize // number of bytes to copy
            )
        }

        // Hash the local code
        codeHash = keccak256(localCode);
        knownHash = codeHash;
    }

    function extCodeCopyColdMemExpansion() public returns (bytes32 codeHash) {
        uint256 codeSize;
        address contractAddress = address(counter);
        assembly {
            codeSize := extcodesize(contractAddress)
        }

        address coldAddress = address(coldCounter);
        bytes memory localCode = new bytes(codeSize);
        assembly {
            let mSize := msize()
            extcodecopy(
                coldAddress, // address to copy from
                sub(mSize, 1), // place it in the last few bytes of memory to force expansion
                0, // start reading from this position in the code
                codeSize // number of bytes to copy
            )
        }

        // Hash the local code
        codeHash = keccak256(localCode);
        knownHash = codeHash;
    }
}
