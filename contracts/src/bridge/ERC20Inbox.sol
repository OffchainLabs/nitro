// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {
    AlreadyInit,
    NotAllowedOrigin,
    DataTooLarge,
    AlreadyPaused,
    AlreadyUnpaused,
    Paused,
    InsufficientValue,
    InsufficientSubmissionCost,
    RetryableData,
    NotRollupOrOwner,
    GasLimitTooLarge
} from "../libraries/Error.sol";
import "./IERC20Inbox.sol";
import "./ISequencerInbox.sol";
import "./IERC20Bridge.sol";
import "./Messages.sol";
import "../libraries/AddressAliasHelper.sol";
import "../libraries/DelegateCallAware.sol";
import {
    L1MessageType_submitRetryableTx,
    L1MessageType_ethDeposit
} from "../libraries/MessageTypes.sol";
import {MAX_DATA_SIZE} from "../libraries/Constants.sol";
import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/security/PausableUpgradeable.sol";

/**
 * @title Inbox for user and contract originated messages
 * @notice Messages created via this inbox are enqueued in the delayed accumulator
 * to await inclusion in the SequencerInbox
 */
contract ERC20Inbox is DelegateCallAware, PausableUpgradeable, IERC20Inbox {
    IBridge public bridge;
    ISequencerInbox public sequencerInbox;

    /// ------------------------------------ allow list start ------------------------------------ ///

    bool public allowListEnabled;
    mapping(address => bool) public isAllowed;

    event AllowListAddressSet(address indexed user, bool val);
    event AllowListEnabledUpdated(bool isEnabled);

    function setAllowList(address[] memory user, bool[] memory val) external onlyRollupOrOwner {
        require(user.length == val.length, "INVALID_INPUT");

        for (uint256 i = 0; i < user.length; i++) {
            isAllowed[user[i]] = val[i];
            emit AllowListAddressSet(user[i], val[i]);
        }
    }

    function setAllowListEnabled(bool _allowListEnabled) external onlyRollupOrOwner {
        require(_allowListEnabled != allowListEnabled, "ALREADY_SET");
        allowListEnabled = _allowListEnabled;
        emit AllowListEnabledUpdated(_allowListEnabled);
    }

    /// @dev this modifier checks the tx.origin instead of msg.sender for convenience (ie it allows
    /// allowed users to interact with the token bridge without needing the token bridge to be allowList aware).
    /// this modifier is not intended to use to be used for security (since this opens the allowList to
    /// a smart contract phishing risk).
    modifier onlyAllowed() {
        // solhint-disable-next-line avoid-tx-origin
        if (allowListEnabled && !isAllowed[tx.origin]) revert NotAllowedOrigin(tx.origin);
        _;
    }

    /// ------------------------------------ allow list end ------------------------------------ ///

    modifier onlyRollupOrOwner() {
        IOwnable rollup = bridge.rollup();
        if (msg.sender != address(rollup)) {
            address rollupOwner = rollup.owner();
            if (msg.sender != rollupOwner) {
                revert NotRollupOrOwner(msg.sender, address(rollup), rollupOwner);
            }
        }
        _;
    }

    uint256 internal immutable deployTimeChainId = block.chainid;

    /// @inheritdoc IERC20Inbox
    function pause() external onlyRollupOrOwner {
        _pause();
    }

    /// @inheritdoc IERC20Inbox
    function unpause() external onlyRollupOrOwner {
        _unpause();
    }

    function initialize(IBridge _bridge, ISequencerInbox _sequencerInbox)
        external
        initializer
        onlyDelegated
    {
        bridge = _bridge;
        sequencerInbox = _sequencerInbox;
        allowListEnabled = false;
        __Pausable_init();
    }

    /// @inheritdoc IERC20Inbox
    function depositERC20(uint256 amount) public whenNotPaused onlyAllowed returns (uint256) {
        address dest = msg.sender;

        // solhint-disable-next-line avoid-tx-origin
        if (AddressUpgradeable.isContract(msg.sender) || tx.origin != msg.sender) {
            // isContract check fails if this function is called during a contract's constructor.
            dest = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        return
            _deliverMessage(
                L1MessageType_ethDeposit,
                msg.sender,
                abi.encodePacked(dest, amount),
                amount
            );
    }

    /// @inheritdoc IERC20Inbox
    function createRetryableTicket(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 tokenTotalFeeAmount,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        // ensure the user's deposit alone will make submission succeed
        if (tokenTotalFeeAmount < (maxSubmissionCost + l2CallValue + gasLimit * maxFeePerGas)) {
            revert InsufficientValue(
                maxSubmissionCost + l2CallValue + gasLimit * maxFeePerGas,
                tokenTotalFeeAmount
            );
        }

        // if a refund address is a contract, we apply the alias to it
        // so that it can access its funds on the L2
        // since the beneficiary and other refund addresses don't get rewritten by arb-os
        if (AddressUpgradeable.isContract(excessFeeRefundAddress)) {
            excessFeeRefundAddress = AddressAliasHelper.applyL1ToL2Alias(excessFeeRefundAddress);
        }
        if (AddressUpgradeable.isContract(callValueRefundAddress)) {
            // this is the beneficiary. be careful since this is the address that can cancel the retryable in the L2
            callValueRefundAddress = AddressAliasHelper.applyL1ToL2Alias(callValueRefundAddress);
        }

        // gas limit is validated to be within uint64 in unsafeCreateRetryableTicket
        return
            unsafeCreateRetryableTicket(
                to,
                l2CallValue,
                maxSubmissionCost,
                excessFeeRefundAddress,
                callValueRefundAddress,
                gasLimit,
                maxFeePerGas,
                tokenTotalFeeAmount,
                data
            );
    }

    // /// @inheritdoc IERC20Inbox
    function unsafeCreateRetryableTicket(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 tokenTotalFeeAmount,
        bytes calldata data
    ) public payable whenNotPaused onlyAllowed returns (uint256) {
        // gas price and limit of 1 should never be a valid input, so instead they are used as
        // magic values to trigger a revert in eth calls that surface data without requiring a tx trace
        if (gasLimit == 1 || maxFeePerGas == 1)
            revert RetryableData(
                msg.sender,
                to,
                l2CallValue,
                tokenTotalFeeAmount,
                maxSubmissionCost,
                excessFeeRefundAddress,
                callValueRefundAddress,
                gasLimit,
                maxFeePerGas,
                data
            );

        // arbos will discard retryable with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }

        uint256 submissionFee = calculateRetryableSubmissionFee(data.length, block.basefee);
        if (maxSubmissionCost < submissionFee)
            revert InsufficientSubmissionCost(submissionFee, maxSubmissionCost);

        return
            _deliverMessage(
                L1MessageType_submitRetryableTx,
                msg.sender,
                abi.encodePacked(
                    uint256(uint160(to)),
                    l2CallValue,
                    tokenTotalFeeAmount,
                    maxSubmissionCost,
                    uint256(uint160(excessFeeRefundAddress)),
                    uint256(uint160(callValueRefundAddress)),
                    gasLimit,
                    maxFeePerGas,
                    data.length,
                    data
                ),
                tokenTotalFeeAmount
            );
    }

    /// @inheritdoc IERC20Inbox
    function calculateRetryableSubmissionFee(uint256, uint256) public pure returns (uint256) {
        return 0;
    }

    function _deliverMessage(
        uint8 _kind,
        address _sender,
        bytes memory _messageData,
        uint256 tokenAmount
    ) internal returns (uint256) {
        if (_messageData.length > MAX_DATA_SIZE)
            revert DataTooLarge(_messageData.length, MAX_DATA_SIZE);
        uint256 msgNum = deliverToBridge(_kind, _sender, keccak256(_messageData), tokenAmount);
        emit InboxMessageDelivered(msgNum, _messageData);
        return msgNum;
    }

    function deliverToBridge(
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 tokenAmount
    ) internal returns (uint256) {
        return
            IERC20Bridge(address(bridge)).enqueueDelayedMessage(
                kind,
                AddressAliasHelper.applyL1ToL2Alias(sender),
                messageDataHash,
                tokenAmount
            );
    }
}
