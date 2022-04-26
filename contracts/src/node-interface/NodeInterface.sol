// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.4.21 <0.9.0;

/** @title Interface for providing gas estimation for retryable auto-redeems and constructing outbox proofs
 *  @notice This contract doesn't exist on-chain. Instead it is a virtual interface accessible at
 *  0x00000000000000000000000000000000000000C8
 *  This is a cute trick to allow an Arbitrum node to provide data without us having to implement additional RPCs
 */
interface NodeInterface {
    /**
     * @notice Estimate the cost of putting a message in the L2 inbox that is reexecuted
     * @param sender sender of the L1 and L2 transaction
     * @param deposit amount to deposit to sender in L2
     * @param to destination L2 contract address
     * @param l2CallValue call value for retryable L2 message
     * @param excessFeeRefundAddress gasLimit x maxFeePerGas - execution cost gets credited here on L2 balance
     * @param callValueRefundAddress l2Callvalue gets credited here on L2 if retryable txn times out or gets cancelled
     * @param data ABI encoded data of L2 message
     */
    function estimateRetryableTicket(
        address sender,
        uint256 deposit,
        address to,
        uint256 l2CallValue,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        bytes calldata data
    ) external;

    /**
     * @notice Constructs an outbox proof of an l2->l1 send's existence in the outbox accumulator
     * @param size the number of elements in the accumulator
     * @param leaf the position of the send in the accumulator
     * @return send the l2->l1 send's hash
     * @return root the root of the outbox accumulator
     * @return proof level-by-level branch hashes constituting a proof of the send's membership at the given size
     */
    function constructOutboxProof(uint64 size, uint64 leaf)
        external
        view
        returns (
            bytes32 send,
            bytes32 root,
            bytes32[] memory proof
        );

    /**
     * @notice Finds the L1 batch containing a requested L2 block, reverting if none does
     * Throws if block doesn't exist, or if block number is 0
     * @param blockNum The L2 block being queried
     * @return batch The L1 block containing the requested L2 block
     */
    function findBatchContainingBlock(uint64 blockNum) external view returns (uint64 batch);

    /**
     * @notice Gets the number of L1 confirmations of the sequencer batch producing the requested L2 block
     * This gets the number of L1 confirmations for the input message producing the L2 block,
     * which happens well before the L1 rollup contract confirms the L2 block.
     * Throws if block doesnt exist in the L2 chain.
     * @param blockHash The hash of the L2 block being queried
     * @return confirmations The number of L1 confirmations the sequencer batch has. Returns 0 if block not yet included in an L1 batch.
     */
    function getL1Confirmations(bytes32 blockHash) external view returns (uint64 confirmations);
}
