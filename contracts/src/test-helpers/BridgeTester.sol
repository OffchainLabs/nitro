// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";

import {
    NotContract,
    NotRollupOrOwner,
    NotDelayedInbox,
    NotSequencerInbox,
    NotOutbox,
    InvalidOutboxSet
} from "../libraries/Error.sol";
import "../bridge/IBridge.sol";
import "../bridge/Messages.sol";
import "../libraries/DelegateCallAware.sol";

/**
 * @title Staging ground for incoming and outgoing messages
 * @notice Holds the inbox accumulator for delayed messages, and is the ETH escrow
 * for value sent with these messages.
 * Since the escrow is held here, this contract also contains a list of allowed
 * outboxes that can make calls from here and withdraw this escrow.
 */
contract BridgeTester is Initializable, DelegateCallAware, IBridge {
    using AddressUpgradeable for address;

    struct InOutInfo {
        uint256 index;
        bool allowed;
    }

    mapping(address => InOutInfo) private allowedInboxesMap;
    mapping(address => InOutInfo) private allowedOutboxesMap;

    address[] public allowedDelayedInboxList;
    address[] public allowedOutboxList;

    address private _activeOutbox;

    IOwnable public rollup;
    address public sequencerInbox;

    modifier onlyRollupOrOwner() {
        if (msg.sender != address(rollup)) {
            address rollupOwner = rollup.owner();
            if (msg.sender != rollupOwner) {
                revert NotRollupOrOwner(msg.sender, address(rollup), rollupOwner);
            }
        }
        _;
    }

    function setSequencerInbox(address _sequencerInbox) external override onlyRollupOrOwner {
        sequencerInbox = _sequencerInbox;
        emit SequencerInboxUpdated(_sequencerInbox);
    }

    /// @dev Accumulator for delayed inbox messages; tail represents hash of the current state; each element represents the inclusion of a new message.
    bytes32[] public override delayedInboxAccs;

    bytes32[] public override sequencerInboxAccs;
    uint256 public override sequencerReportedSubMessageCount;

    address private constant EMPTY_ACTIVEOUTBOX = address(type(uint160).max);

    function initialize(IOwnable rollup_) external initializer {
        _activeOutbox = EMPTY_ACTIVEOUTBOX;
        rollup = rollup_;
    }

    function updateRollupAddress(IOwnable _rollup) external onlyDelegated onlyProxyOwner {
        rollup = _rollup;
    }

    function activeOutbox() public view returns (address) {
        if (_activeOutbox == EMPTY_ACTIVEOUTBOX) return address(uint160(0));
        return _activeOutbox;
    }

    function allowedDelayedInboxes(address inbox) external view override returns (bool) {
        return allowedInboxesMap[inbox].allowed;
    }

    function allowedOutboxes(address outbox) external view override returns (bool) {
        return allowedOutboxesMap[outbox].allowed;
    }

    function enqueueSequencerMessage(
        bytes32 dataHash,
        uint256 afterDelayedMessagesRead,
        uint256 prevMessageCount,
        uint256 newMessageCount
    )
        external
        returns (
            uint256 seqMessageIndex,
            bytes32 beforeAcc,
            bytes32 delayedAcc,
            bytes32 acc
        )
    {
        // TODO: implement stub logic
    }

    function submitBatchSpendingReport(address batchPoster, bytes32 dataHash)
        external
        returns (uint256)
    {
        // TODO: implement stub
    }

    /**
     * @dev Enqueue a message in the delayed inbox accumulator.
     * These messages are later sequenced in the SequencerInbox, either by the sequencer as
     * part of a normal batch, or by force inclusion.
     */
    function enqueueDelayedMessage(
        uint8 kind,
        address sender,
        bytes32 messageDataHash
    ) external payable override returns (uint256) {
        if (!allowedInboxesMap[msg.sender].allowed) revert NotDelayedInbox(msg.sender);
        return
            addMessageToDelayedAccumulator(
                kind,
                sender,
                uint64(block.number),
                uint64(block.timestamp), // solhint-disable-line not-rely-on-time
                block.basefee,
                messageDataHash
            );
    }

    function addMessageToDelayedAccumulator(
        uint8 kind,
        address sender,
        uint64 blockNumber,
        uint64 blockTimestamp,
        uint256 baseFeeL1,
        bytes32 messageDataHash
    ) internal returns (uint256) {
        uint256 count = delayedInboxAccs.length;
        bytes32 messageHash = Messages.messageHash(
            kind,
            sender,
            blockNumber,
            blockTimestamp,
            count,
            baseFeeL1,
            messageDataHash
        );
        bytes32 prevAcc = 0;
        if (count > 0) {
            prevAcc = delayedInboxAccs[count - 1];
        }
        delayedInboxAccs.push(Messages.accumulateInboxMessage(prevAcc, messageHash));
        emit MessageDelivered(
            count,
            prevAcc,
            msg.sender,
            kind,
            sender,
            messageDataHash,
            baseFeeL1,
            blockTimestamp
        );
        return count;
    }

    function executeCall(
        address to,
        uint256 value,
        bytes calldata data
    ) external override returns (bool success, bytes memory returnData) {
        if (!allowedOutboxesMap[msg.sender].allowed) revert NotOutbox(msg.sender);
        if (data.length > 0 && !to.isContract()) revert NotContract(to);
        address prevOutbox = _activeOutbox;
        _activeOutbox = msg.sender;
        // We set and reset active outbox around external call so activeOutbox remains valid during call

        // We use a low level call here since we want to bubble up whether it succeeded or failed to the caller
        // rather than reverting on failure as well as allow contract and non-contract calls
        // solhint-disable-next-line avoid-low-level-calls
        (success, returnData) = to.call{value: value}(data);
        _activeOutbox = prevOutbox;
        emit BridgeCallTriggered(msg.sender, to, value, data);
    }

    function setDelayedInbox(address inbox, bool enabled) external override onlyRollupOrOwner {
        InOutInfo storage info = allowedInboxesMap[inbox];
        bool alreadyEnabled = info.allowed;
        emit InboxToggle(inbox, enabled);
        if (alreadyEnabled == enabled) {
            return;
        }
        if (enabled) {
            allowedInboxesMap[inbox] = InOutInfo(allowedDelayedInboxList.length, true);
            allowedDelayedInboxList.push(inbox);
        } else {
            allowedDelayedInboxList[info.index] = allowedDelayedInboxList[
                allowedDelayedInboxList.length - 1
            ];
            allowedInboxesMap[allowedDelayedInboxList[info.index]].index = info.index;
            allowedDelayedInboxList.pop();
            delete allowedInboxesMap[inbox];
        }
    }

    function setOutbox(address outbox, bool enabled) external override onlyRollupOrOwner {
        InOutInfo storage info = allowedOutboxesMap[outbox];
        bool alreadyEnabled = info.allowed;
        emit OutboxToggle(outbox, enabled);
        if (alreadyEnabled == enabled) {
            return;
        }
        if (enabled) {
            allowedOutboxesMap[outbox] = InOutInfo(allowedOutboxList.length, true);
            allowedOutboxList.push(outbox);
        } else {
            allowedOutboxList[info.index] = allowedOutboxList[allowedOutboxList.length - 1];
            allowedOutboxesMap[allowedOutboxList[info.index]].index = info.index;
            allowedOutboxList.pop();
            delete allowedOutboxesMap[outbox];
        }
    }

    function delayedMessageCount() external view override returns (uint256) {
        return delayedInboxAccs.length;
    }

    function sequencerMessageCount() external view override returns (uint256) {
        return sequencerInboxAccs.length;
    }

    receive() external payable {}

    function acceptFundsFromOldBridge() external payable {}
}
