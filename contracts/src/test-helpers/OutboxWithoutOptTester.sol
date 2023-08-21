// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {
    AlreadyInit,
    NotRollup,
    ProofTooLong,
    PathNotMinimal,
    UnknownRoot,
    AlreadySpent,
    BridgeCallFailed
} from "../libraries/Error.sol";
import "../bridge/IBridge.sol";
import "../bridge/IOutbox.sol";
import "../libraries/MerkleLib.sol";
import "../libraries/DelegateCallAware.sol";

contract OutboxWithoutOptTester is DelegateCallAware, IOutbox {
    address public rollup; // the rollup contract
    IBridge public bridge; // the bridge contract

    function spent(uint256) external pure override returns (bytes32) {
        revert("NOT_IMPLEMETED");
    }

    mapping(uint256 => bool) public isSpent; // maps leaf number => if spent
    mapping(bytes32 => bytes32) public roots; // maps root hashes => L2 block hash

    struct L2ToL1Context {
        uint128 l2Block;
        uint128 l1Block;
        uint128 timestamp;
        bytes32 outputId;
        address sender;
    }
    // Note, these variables are set and then wiped during a single transaction.
    // Therefore their values don't need to be maintained, and their slots will
    // be empty outside of transactions
    L2ToL1Context internal context;
    uint128 public constant OUTBOX_VERSION = 2;

    function initialize(IBridge _bridge) external {
        if (address(bridge) != address(0)) revert AlreadyInit();
        bridge = _bridge;
        rollup = address(_bridge.rollup());
    }

    function updateSendRoot(bytes32 root, bytes32 l2BlockHash) external override {
        //if (msg.sender != rollup) revert NotRollup(msg.sender, rollup);  //test only!!!
        roots[root] = l2BlockHash;
        emit SendRootUpdated(root, l2BlockHash);
    }

    function updateRollupAddress() external onlyDelegated onlyProxyOwner {
        rollup = address(bridge.rollup());
    }

    /// @notice When l2ToL1Sender returns a nonzero address, the message was originated by an L2 account
    /// When the return value is zero, that means this is a system message
    /// @dev the l2ToL1Sender behaves as the tx.origin, the msg.sender should be validated to protect against reentrancies
    function l2ToL1Sender() external view override returns (address) {
        return context.sender;
    }

    function l2ToL1Block() external view override returns (uint256) {
        return uint256(context.l2Block);
    }

    function l2ToL1EthBlock() external view override returns (uint256) {
        return uint256(context.l1Block);
    }

    function l2ToL1Timestamp() external view override returns (uint256) {
        return uint256(context.timestamp);
    }

    // @deprecated batch number is now always 0
    function l2ToL1BatchNum() external pure returns (uint256) {
        return 0;
    }

    function l2ToL1OutputId() external view override returns (bytes32) {
        return context.outputId;
    }

    /**
     * @notice Executes a messages in an Outbox entry.
     * @dev Reverts if dispute period hasn't expired, since the outbox entry
     * is only created once the rollup confirms the respective assertion.
     * @param proof Merkle proof of message inclusion in send root
     * @param index Merkle path to message
     * @param l2Sender sender if original message (i.e., caller of ArbSys.sendTxToL1)
     * @param to destination address for L1 contract call
     * @param l2Block l2 block number at which sendTxToL1 call was made
     * @param l1Block l1 block number at which sendTxToL1 call was made
     * @param l2Timestamp l2 Timestamp at which sendTxToL1 call was made
     * @param value wei in L1 message
     * @param data abi-encoded L1 message data
     */
    function executeTransaction(
        bytes32[] calldata proof,
        uint256 index,
        address l2Sender,
        address to,
        uint256 l2Block,
        uint256 l1Block,
        uint256 l2Timestamp,
        uint256 value,
        bytes calldata data
    ) external virtual override {
        bytes32 outputId;
        {
            bytes32 userTx = calculateItemHash(
                l2Sender,
                to,
                l2Block,
                l1Block,
                l2Timestamp,
                value,
                data
            );

            outputId = recordOutputAsSpent(proof, index, userTx);
            emit OutBoxTransactionExecuted(to, l2Sender, 0, index);
        }

        // we temporarily store the previous values so the outbox can naturally
        // unwind itself when there are nested calls to `executeTransaction`
        L2ToL1Context memory prevContext = context;

        context = L2ToL1Context({
            sender: l2Sender,
            l2Block: uint128(l2Block),
            l1Block: uint128(l1Block),
            timestamp: uint128(l2Timestamp),
            outputId: outputId
        });

        // set and reset vars around execution so they remain valid during call
        executeBridgeCall(to, value, data);

        context = prevContext;
    }

    function executeTransactionSimulation(
        uint256,
        address,
        address,
        uint256,
        uint256,
        uint256,
        uint256,
        bytes calldata
    ) external pure override {
        revert("Not implemented");
    }

    function recordOutputAsSpent(
        bytes32[] memory proof,
        uint256 index,
        bytes32 item
    ) internal returns (bytes32) {
        if (proof.length >= 256) revert ProofTooLong(proof.length);
        if (index >= 2**proof.length) revert PathNotMinimal(index, 2**proof.length);

        // Hash the leaf an extra time to prove it's a leaf
        bytes32 calcRoot = calculateMerkleRoot(proof, index, item);
        if (roots[calcRoot] == bytes32(0)) revert UnknownRoot(calcRoot);

        if (isSpent[index]) revert AlreadySpent(index);
        isSpent[index] = true;

        return bytes32(index);
    }

    function executeBridgeCall(
        address to,
        uint256 value,
        bytes memory data
    ) internal {
        (bool success, bytes memory returndata) = bridge.executeCall(to, value, data);
        if (!success) {
            if (returndata.length > 0) {
                // solhint-disable-next-line no-inline-assembly
                assembly {
                    let returndata_size := mload(returndata)
                    revert(add(32, returndata), returndata_size)
                }
            } else {
                revert BridgeCallFailed();
            }
        }
    }

    function calculateItemHash(
        address l2Sender,
        address to,
        uint256 l2Block,
        uint256 l1Block,
        uint256 l2Timestamp,
        uint256 value,
        bytes calldata data
    ) public pure override returns (bytes32) {
        return
            keccak256(abi.encodePacked(l2Sender, to, l2Block, l1Block, l2Timestamp, value, data));
    }

    function calculateMerkleRoot(
        bytes32[] memory proof,
        uint256 path,
        bytes32 item
    ) public pure override returns (bytes32) {
        return MerkleLib.calculateRoot(proof, path, keccak256(abi.encodePacked(item)));
    }
}
