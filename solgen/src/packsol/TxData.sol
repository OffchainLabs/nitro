// SPDX-License-Identifier: Apache-2.0

pragma solidity >=0.4.21 <0.9.0;

/** @title aa
 *  @notice aa
 */

interface InternalTxData {
    /**
     * @notice aa
     * @param l1BaseFee aa
     * @param l1BlockNumber aa
     * @param timeLastBlock aa
     */
    function startBlock(
        uint256 l1BaseFee,
        uint256 l2BaseFee,
        uint64 l1BlockNumber,
        uint64 timeLastBlock
    ) external;
}
