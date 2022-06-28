// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

/// @dev Init was already called
error AlreadyInit();

/// Init was called with param set to zero that must be nonzero
error HadZeroInit();

/// @dev Thrown when non owner tries to access an only-owner function
/// @param sender The msg.sender who is not the owner
/// @param owner The owner address
error NotOwner(address sender, address owner);

/// @dev Thrown when an address that is not the rollup tries to call an only-rollup function
/// @param sender The sender who is not the rollup
/// @param rollup The rollup address authorized to call this function
error NotRollup(address sender, address rollup);

/// @dev Thrown when the contract was not called directly from the origin ie msg.sender != tx.origin
error NotOrigin();

/// @dev Provided data was too large
/// @param dataLength The length of the data that is too large
/// @param maxDataLength The max length the data can be
error DataTooLarge(uint256 dataLength, uint256 maxDataLength);

/// @dev The provided is not a contract and was expected to be
/// @param addr The adddress in question
error NotContract(address addr);

/// @dev The merkle proof provided was too long
/// @param actualLength The length of the merkle proof provided
/// @param maxProofLength The max length a merkle proof can have
error MerkleProofTooLong(uint256 actualLength, uint256 maxProofLength);

/// @dev Thrown when an un-authorized address tries to access an admin function
/// @param sender The un-authorized sender
/// @param rollup The rollup, which would be authorized
/// @param owner The rollup's owner, which would be authorized
error NotRollupOrOwner(address sender, address rollup, address owner);
