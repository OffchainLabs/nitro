//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
// SPDX-License-Identifier: UNLICENSED
//

pragma solidity ^0.8.4;

/// @dev Init was already called
error AlreadyInit();

/// @dev Thrown when non owner tries to access an only-owner function
/// @param sender The msg.sender who is not the owner
/// @param owner The owner address
error NotOwner(address sender, address owner);

/// @dev Thrown when the contract was not called directly from the origin ie msg.sender != tx.origin
error NotOrigin();

/// @dev Provided data was too large
/// @param dataLength The length of the data that is too large
/// @param dataLength The max length the data can be
error DataTooLarge(uint256 dataLength, uint256 maxDataLength);

/// @dev The provided is not a contract and was expected to be
/// @param addr The adddress in question
error NotContract(address addr);
