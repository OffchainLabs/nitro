// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {AlreadyInit, NotRollup} from "../libraries/Error.sol";

/// @dev The provided proof was too long
/// @param proofLength The length of the too-long proof
error ProofTooLong(uint256 proofLength);

/// @dev The output index was greater than the maximum
/// @param index The output index
/// @param maxIndex The max the index could be
error PathNotMinimal(uint256 index, uint256 maxIndex);

/// @dev The calculated root does not exist
/// @param root The calculated root
error UnknownRoot(bytes32 root);

/// @dev The record has already been spent
/// @param index The index of the spent record
error AlreadySpent(uint256 index);

/// @dev A call to the bridge failed with no return data
error BridgeCallFailed();

interface IOutbox {
    event SendRootUpdated(bytes32 indexed blockHash, bytes32 indexed outputRoot);
    event OutBoxTransactionExecuted(
        address indexed to,
        address indexed l2Sender,
        uint256 indexed zero,
        uint256 transactionIndex
    );

    function l2ToL1Sender() external view returns (address);

    function l2ToL1Block() external view returns (uint256);

    function l2ToL1EthBlock() external view returns (uint256);

    function l2ToL1Timestamp() external view returns (uint256);

    // @deprecated batch number is now always 0
    function l2ToL1BatchNum() external view returns (uint256);

    function l2ToL1OutputId() external view returns (bytes32);

    function updateSendRoot(bytes32 sendRoot, bytes32 l2BlockHash) external;
}
