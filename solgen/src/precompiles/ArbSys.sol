// SPDX-License-Identifier: Apache-2.0

/*
 * Copyright 2020, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

pragma solidity >=0.4.21 <0.9.0;

/**
* @title System level functionality
* @notice For use by contracts to interact with core L2-specific functionality.
* Precompiled contract that exists in every Arbitrum chain at address(100), 0x0000000000000000000000000000000000000064.
 */
interface ArbSys {
    /**
    * @notice Get Arbitrum block number (distinct from L1 block number; Arbitrum genesis block has block number 0)
    * @return block number as int
     */
    function arbBlockNumber() external view returns (uint);

    /**
    * @notice Get Arbitrum block hash (reverts unless currentBlockNum-256 <= arbBlockNum < currentBlockNum)
    * @return block hash
     */
    function arbBlockHash(uint arbBlockNum) external view returns (bytes32);

    /**
    * @notice Gets the rollup's unique chain identifier
    * @return Chain identifier as int
     */
    function arbChainID() external view returns(uint);

    /**
    * @notice Get internal version number identifying an ArbOS build
    * @return version number as int
     */
    function arbOSVersion() external view returns (uint);

    /**
     * @notice Returns 0 since Nitro has no concept of storage gas
     * @return int 0
     */
    function getStorageGasAvailable() external returns(uint);

    /**
    * @notice check if current call is coming from l1
    * @return true if the caller of this was called directly from L1
    */
    function isTopLevelCall() external view returns (bool);

    /**
     * @notice map L1 sender contract address to its L2 alias
     * @param sender sender address
     * @param unused argument no longer used
     * @return aliased sender address
     */
    function mapL1SenderContractAddressToL2Alias(address sender, address unused) external pure returns(address);

    /**
     * @notice check if the caller (of this caller of this) is an aliased L1 contract address
     * @return true iff the caller's address is an alias for an L1 contract address
     */
    function wasMyCallersAddressAliased() external view returns (bool);

    /**
     * @notice return the address of the caller (of this caller of this), without applying L1 contract address aliasing
     * @return address of the caller's caller, without applying L1 contract address aliasing
     */
    function myCallersAddressWithoutAliasing() external view returns (address);

    /**
    * @notice Send given amount of Eth to dest from sender.
    * This is a convenience function, which is equivalent to calling sendTxToL1 with empty calldataForL1.
    * @param destination recipient address on L1
    * @return unique identifier for this L2-to-L1 transaction.
    */
    function withdrawEth(address destination) external payable returns(uint);

    /**
    * @notice Send a transaction to L1
    * @param destination recipient address on L1
    * @param calldataForL1 (optional) calldata for L1 contract call
    * @return a unique identifier for this L2-to-L1 transaction.
    */
    function sendTxToL1(address destination, bytes calldata calldataForL1) external payable returns(uint);

    /**
    * @notice Get send Merkle tree state
    * @return size number of sends in the history
    * @return root root hash of the send history
    * @return partials hashes of partial subtrees in the send history tree
    */
    function sendMerkleTreeState() external view returns(uint size, bytes32 root, bytes32[] memory partials);

    /**
     * @notice creates a send txn from L2 to L1
     * @param position = (level << 192) + leaf = (0 << 192) + leaf = leaf
    */
    event L2ToL1Transaction(
        address caller,
        address indexed destination,
        uint indexed hash,
        uint indexed position,
        uint indexInBatch,
        uint arbBlockNum,
        uint ethBlockNum,
        uint timestamp,
        uint callvalue,
        bytes data
    );

    /**
     * @notice logs a merkle branch for proof sythesis
     * @param reserved an index meant only to align the 4th index with L2ToL1Transaction's 4th event
     * @param hash the merkle hash
     * @param position = (level << 192) + leaf
    */
    event SendMerkleUpdate(
        uint indexed reserved,
        bytes32 indexed hash,
        uint indexed position
    );
}
