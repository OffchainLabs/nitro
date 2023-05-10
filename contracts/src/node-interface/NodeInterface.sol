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
     * @notice Simulate the execution of a retryable ticket
     * @dev Use eth_estimateGas on this call to estimate gas usage of retryable ticket
     *      Since gas usage is not yet known, you may need to add extra deposit (e.g. 1e18 wei) during estimation
     * @param sender unaliased sender of the L1 and L2 transaction
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
     * @notice Constructs an outbox proof of an l2->l1 send's existence in the outbox accumulator.
     * @dev Use eth_call to call.
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
     * @notice Finds the L1 batch containing a requested L2 block, reverting if none does.
     * Use eth_call to call.
     * Throws if block doesn't exist, or if block number is 0. Use eth_call
     * @param blockNum The L2 block being queried
     * @return batch The sequencer batch number containing the requested L2 block
     */
    function findBatchContainingBlock(uint64 blockNum) external view returns (uint64 batch);

    /**
     * @notice Gets the number of L1 confirmations of the sequencer batch producing the requested L2 block
     * This gets the number of L1 confirmations for the input message producing the L2 block,
     * which happens well before the L1 rollup contract confirms the L2 block.
     * Throws if block doesnt exist in the L2 chain.
     * @dev Use eth_call to call.
     * @param blockHash The hash of the L2 block being queried
     * @return confirmations The number of L1 confirmations the sequencer batch has. Returns 0 if block not yet included in an L1 batch.
     */
    function getL1Confirmations(bytes32 blockHash) external view returns (uint64 confirmations);

    /**
     * @notice Same as native gas estimation, but with additional info on the l1 costs.
     * @dev Use eth_call to call.
     * @param data the tx's calldata. Everything else like "From" and "Gas" are copied over
     * @param to the tx's "To" (ignored when contractCreation is true)
     * @param contractCreation whether "To" is omitted
     * @return gasEstimate an estimate of the total amount of gas needed for this tx
     * @return gasEstimateForL1 an estimate of the amount of gas needed for the l1 component of this tx
     * @return baseFee the l2 base fee
     * @return l1BaseFeeEstimate ArbOS's l1 estimate of the l1 base fee
     */
    function gasEstimateComponents(
        address to,
        bool contractCreation,
        bytes calldata data
    )
        external
        payable
        returns (
            uint64 gasEstimate,
            uint64 gasEstimateForL1,
            uint256 baseFee,
            uint256 l1BaseFeeEstimate
        );

    /**
     * @notice Estimates a transaction's l1 costs.
     * @dev Use eth_call to call.
     *      This method is similar to gasEstimateComponents, but doesn't include the l2 component
     *      so that the l1 component can be known even when the tx may fail.
     *      This method also doesn't pad the estimate as gas estimation normally does.
     *      If using this value to submit a transaction, we'd recommend first padding it by 10%.
     * @param data the tx's calldata. Everything else like "From" and "Gas" are copied over
     * @param to the tx's "To" (ignored when contractCreation is true)
     * @param contractCreation whether "To" is omitted
     * @return gasEstimateForL1 an estimate of the amount of gas needed for the l1 component of this tx
     * @return baseFee the l2 base fee
     * @return l1BaseFeeEstimate ArbOS's l1 estimate of the l1 base fee
     */
    function gasEstimateL1Component(
        address to,
        bool contractCreation,
        bytes calldata data
    )
        external
        payable
        returns (
            uint64 gasEstimateForL1,
            uint256 baseFee,
            uint256 l1BaseFeeEstimate
        );

    /**
     * @notice Returns the proof necessary to redeem a message
     * @param batchNum index of outbox entry (i.e., outgoing messages Merkle root) in array of outbox entries
     * @param index index of outgoing message in outbox entry
     * @return proof Merkle proof of message inclusion in outbox entry
     * @return path Merkle path to message
     * @return l2Sender sender if original message (i.e., caller of ArbSys.sendTxToL1)
     * @return l1Dest destination address for L1 contract call
     * @return l2Block l2 block number at which sendTxToL1 call was made
     * @return l1Block l1 block number at which sendTxToL1 call was made
     * @return timestamp l2 Timestamp at which sendTxToL1 call was made
     * @return amount value in L1 message in wei
     * @return calldataForL1 abi-encoded L1 message data
     */
    function legacyLookupMessageBatchProof(uint256 batchNum, uint64 index)
        external
        view
        returns (
            bytes32[] memory proof,
            uint256 path,
            address l2Sender,
            address l1Dest,
            uint256 l2Block,
            uint256 l1Block,
            uint256 timestamp,
            uint256 amount,
            bytes memory calldataForL1
        );

    // @notice Returns the first block produced using the Nitro codebase
    // @dev returns 0 for chains like Nova that don't contain classic blocks
    // @return number the block number
    function nitroGenesisBlock() external pure returns (uint256 number);
}
