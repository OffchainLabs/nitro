// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./Messages.sol";
import "./DelayBufferTypes.sol";

/**
 * @title   Manages the delay buffer for the sequencer (SequencerInbox.sol)
 * @notice  Messages are expected to be delayed up to a threshold, beyond which they are unexpected
 *          and deplete a delay buffer. Buffer depletion is preveneted from decreasing too quickly by only
 *          depleting by as many blocks as elapsed in the delayed message queue.
 */
library DelayBuffer {
    uint256 public constant BASIS = 10000;

    /// @dev    Depletion is limited by the elapsed blocks in the delayed message queue to avoid double counting and potential L2 reorgs.
    ///         Eg. 2 simultaneous batches sequencing multiple delayed messages with the same 100 blocks delay each
    ///         should count once as a single 100 block delay, not twice as a 200 block delay. This also prevents L2 reorg risk in edge cases.
    ///         Eg. If the buffer is 300 blocks, decrementing the buffer when processing the first batch would allow the second delay message to be force included before the sequencer could add the second batch.
    ///         Buffer depletion also saturates at the threshold instead of zero to allow a recovery margin.
    ///         Eg. when the sequencer recovers from an outage, it is able to wait threshold > finality time before queueing delayed messages to avoid L1 reorgs.
    /// @notice Conditionally updates the buffer. Replenishes the buffer and depletes if delay is unexpected.
    /// @param start The beginning reference point
    /// @param end The ending reference point
    /// @param buffer The buffer to be updated
    /// @param sequenced The reference point when messages were sequenced
    /// @param threshold The threshold to saturate at
    /// @param max The maximum buffer
    /// @param replenishRateInBasis The amount to replenish the buffer per block in basis points.
    function calcBuffer(
        uint256 start,
        uint256 end,
        uint256 buffer,
        uint256 sequenced,
        uint256 threshold,
        uint256 max,
        uint256 replenishRateInBasis
    ) internal pure returns (uint256) {
        uint256 elapsed = end > start ? end - start : 0;
        uint256 delay = sequenced > start ? sequenced - start : 0;
        // replenishment rounds down and will not overflow since all inputs including
        // replenishRateInBasis are cast from uint64 in calcPendingBuffer
        buffer += (elapsed * replenishRateInBasis) / BASIS;

        uint256 unexpectedDelay = delay > threshold ? delay - threshold : 0;
        if (unexpectedDelay > elapsed) {
            unexpectedDelay = elapsed;
        }

        // decrease the buffer
        if (buffer > unexpectedDelay) {
            buffer -= unexpectedDelay;
            if (buffer > threshold) {
                // saturating above at the max
                return buffer > max ? max : buffer;
            }
        }
        // saturating below at the threshold
        return threshold;
    }

    /// @notice Applies update to buffer data
    /// @param self The delay buffer data
    /// @param blockNumber The update block number
    function update(BufferData storage self, uint64 blockNumber) internal {
        self.bufferBlocks = calcPendingBuffer(self, blockNumber);

        // store a new starting reference point
        // any buffer updates will be applied retroactively in the next batch post
        self.prevBlockNumber = blockNumber;
        self.prevSequencedBlockNumber = uint64(block.number);
    }

    /// @dev    The delay buffer can change due to pending depletion / replenishment due to previous delays.
    ///         This function applies pending buffer changes to calculate buffer updates.
    /// @notice Calculates the buffer changes up to the requested block number
    /// @param self The delay buffer data
    /// @param blockNumber The block number to process the delay up to
    function calcPendingBuffer(BufferData storage self, uint64 blockNumber)
        internal
        view
        returns (uint64)
    {
        // bufferUpdate will not overflow since inputs are uint64
        return
            uint64(
                calcBuffer({
                    start: self.prevBlockNumber,
                    end: blockNumber,
                    buffer: self.bufferBlocks,
                    threshold: self.threshold,
                    sequenced: self.prevSequencedBlockNumber,
                    max: self.max,
                    replenishRateInBasis: self.replenishRateInBasis
                })
            );
    }

    /// @dev    This is the `sync validity window` during which no proofs are required.
    /// @notice Returns true if the inbox is in a synced state (no unexpected delays are possible)
    function isSynced(BufferData storage self) internal view returns (bool) {
        return block.number - self.prevBlockNumber <= self.threshold;
    }

    function isUpdatable(BufferData storage self) internal view returns (bool) {
        // if synced, the buffer can't be depleted
        // if full, the buffer can't be replenished
        // if neither synced nor full, the buffer updatable (depletable / replenishable)
        return !isSynced(self) || self.bufferBlocks < self.max;
    }

    function isValidBufferConfig(BufferConfig memory config) internal pure returns (bool) {
        return
            config.threshold != 0 &&
            config.max != 0 &&
            config.replenishRateInBasis <= BASIS &&
            config.threshold <= config.max;
    }
}
