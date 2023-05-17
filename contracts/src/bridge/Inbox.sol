// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.4;

import {
    NotOrigin,
    DataTooLarge,
    InsufficientValue,
    InsufficientSubmissionCost,
    RetryableData,
    L1Forked,
    NotForked,
    GasLimitTooLarge
} from "../libraries/Error.sol";
import "./AbsInbox.sol";
import "./IEthInbox.sol";
import "./IBridge.sol";
import "./IEthBridge.sol";
import "../libraries/AddressAliasHelper.sol";
import {
    L2_MSG,
    L1MessageType_L2FundedByL1,
    L1MessageType_submitRetryableTx,
    L1MessageType_ethDeposit,
    L2MessageType_unsignedEOATx,
    L2MessageType_unsignedContractTx
} from "../libraries/MessageTypes.sol";
import {MAX_DATA_SIZE, UNISWAP_L1_TIMELOCK, UNISWAP_L2_FACTORY} from "../libraries/Constants.sol";
import "../precompiles/ArbSys.sol";

import "@openzeppelin/contracts-upgradeable/utils/AddressUpgradeable.sol";

/**
 * @title Inbox for user and contract originated messages
 * @notice Messages created via this inbox are enqueued in the delayed accumulator
 * to await inclusion in the SequencerInbox
 */
contract Inbox is AbsInbox, IEthInbox {

    /// @inheritdoc IInbox
    function initialize(
        IBridge _bridge,
        ISequencerInbox _sequencerInbox
    ) external initializer onlyDelegated {
       __AbsInbox_init(_bridge, _sequencerInbox);
    }

    /// @inheritdoc IEthInbox
    function postUpgradeInit(IBridge) external onlyDelegated onlyProxyOwner {}

    /// @inheritdoc IEthInbox
    function sendL1FundedUnsignedTransaction(
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 nonce,
        address to,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        // arbos will discard unsigned tx with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }
        return
            _deliverMessage(
                L1MessageType_L2FundedByL1,
                msg.sender,
                abi.encodePacked(
                    L2MessageType_unsignedEOATx,
                    gasLimit,
                    maxFeePerGas,
                    nonce,
                    uint256(uint160(to)),
                    msg.value,
                    data
                ),
                msg.value
            );
    }

    /// @inheritdoc IEthInbox
    function sendL1FundedContractTransaction(
        uint256 gasLimit,
        uint256 maxFeePerGas,
        address to,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        // arbos will discard unsigned tx with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }
        return
            _deliverMessage(
                L1MessageType_L2FundedByL1,
                msg.sender,
                abi.encodePacked(
                    L2MessageType_unsignedContractTx,
                    gasLimit,
                    maxFeePerGas,
                    uint256(uint160(to)),
                    msg.value,
                    data
                ),
                msg.value
            );
    }

    /// @inheritdoc IEthInbox
    function sendL1FundedUnsignedTransactionToFork(
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 nonce,
        address to,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        if (!_chainIdChanged()) revert NotForked();
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        // arbos will discard unsigned tx with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }
        return
            _deliverMessage(
                L1MessageType_L2FundedByL1,
                // undoing sender alias here to cancel out the aliasing
                AddressAliasHelper.undoL1ToL2Alias(msg.sender),
                abi.encodePacked(
                    L2MessageType_unsignedEOATx,
                    gasLimit,
                    maxFeePerGas,
                    nonce,
                    uint256(uint160(to)),
                    msg.value,
                    data
                ),
                msg.value
            );
    }

    /// @inheritdoc IEthInbox
    function sendUnsignedTransactionToFork(
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 nonce,
        address to,
        uint256 value,
        bytes calldata data
    ) external whenNotPaused onlyAllowed returns (uint256) {
        if (!_chainIdChanged()) revert NotForked();
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        // arbos will discard unsigned tx with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }
        return
            _deliverMessage(
                L2_MSG,
                // undoing sender alias here to cancel out the aliasing
                AddressAliasHelper.undoL1ToL2Alias(msg.sender),
                abi.encodePacked(
                    L2MessageType_unsignedEOATx,
                    gasLimit,
                    maxFeePerGas,
                    nonce,
                    uint256(uint160(to)),
                    value,
                    data
                ),
                0
            );
    }

    /// @inheritdoc IEthInbox
    function sendWithdrawEthToFork(
        uint256 gasLimit,
        uint256 maxFeePerGas,
        uint256 nonce,
        uint256 value,
        address withdrawTo
    ) external whenNotPaused onlyAllowed returns (uint256) {
        if (!_chainIdChanged()) revert NotForked();
        // solhint-disable-next-line avoid-tx-origin
        if (msg.sender != tx.origin) revert NotOrigin();
        // arbos will discard unsigned tx with gas limit too large
        if (gasLimit > type(uint64).max) {
            revert GasLimitTooLarge();
        }
        return
            _deliverMessage(
                L2_MSG,
                // undoing sender alias here to cancel out the aliasing
                AddressAliasHelper.undoL1ToL2Alias(msg.sender),
                abi.encodePacked(
                    L2MessageType_unsignedEOATx,
                    gasLimit,
                    maxFeePerGas,
                    nonce,
                    uint256(uint160(address(100))), // ArbSys address
                    value,
                    abi.encode(ArbSys.withdrawEth.selector, withdrawTo)
                ),
                0
            );
    }

    /// @inheritdoc IEthInbox
    function depositEth() public payable whenNotPaused onlyAllowed returns (uint256) {
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
                abi.encodePacked(dest, msg.value),
                msg.value
            );
    }

    /// @notice deprecated in favour of depositEth with no parameters
    function depositEth(uint256) external payable whenNotPaused onlyAllowed returns (uint256) {
        return depositEth();
    }

    /**
     * @notice deprecated in favour of unsafeCreateRetryableTicket
     * @dev deprecated in favour of unsafeCreateRetryableTicket
     * @dev Gas limit and maxFeePerGas should not be set to 1 as that is used to trigger the RetryableData error
     * @param to destination L2 contract address
     * @param l2CallValue call value for retryable L2 message
     * @param maxSubmissionCost Max gas deducted from user's L2 balance to cover base submission fee
     * @param excessFeeRefundAddress gasLimit x maxFeePerGas - execution cost gets credited here on L2 balance
     * @param callValueRefundAddress l2Callvalue gets credited here on L2 if retryable txn times out or gets cancelled
     * @param gasLimit Max gas deducted from user's L2 balance to cover L2 execution. Should not be set to 1 (magic value used to trigger the RetryableData error)
     * @param maxFeePerGas price bid for L2 execution. Should not be set to 1 (magic value used to trigger the RetryableData error)
     * @param data ABI encoded data of L2 message
     * @return unique message number of the retryable transaction
     */
    function createRetryableTicketNoRefundAliasRewrite(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
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
                data
            );
    }

    /// @inheritdoc IEthInbox
    function createRetryableTicket(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        return
            _createRetryableTicket(
                to,
                l2CallValue,
                maxSubmissionCost,
                excessFeeRefundAddress,
                callValueRefundAddress,
                gasLimit,
                maxFeePerGas,
                msg.value,
                data
            );
    }

    /// @inheritdoc IEthInbox
    function unsafeCreateRetryableTicket(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        bytes calldata data
    ) public payable whenNotPaused onlyAllowed returns (uint256) {
        return
            _unsafeCreateRetryableTicket(
                to,
                l2CallValue,
                maxSubmissionCost,
                excessFeeRefundAddress,
                callValueRefundAddress,
                gasLimit,
                maxFeePerGas,
                msg.value,
                data
            );
    }

    /// @inheritdoc IInbox
    function calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee)
        public
        view
        override(AbsInbox, IInbox)
        returns (uint256)
    {
        // Use current block basefee if baseFee parameter is 0
        return (1400 + 6 * dataLength) * (baseFee == 0 ? block.basefee : baseFee);
    }

    /// @notice This is an one-time-exception to resolve a misconfiguration of Uniswap Arbitrum deployment
    ///         Only the Uniswap L1 Timelock may call this function and it is allowed to create a crosschain
    ///         retryable ticket without address aliasing. More info here:
    ///         https://gov.uniswap.org/t/consensus-check-fix-the-cross-chain-messaging-bridge-on-arbitrum/18547
    /// @dev    This function will be removed in future releases
    function uniswapCreateRetryableTicket(
        address to,
        uint256 l2CallValue,
        uint256 maxSubmissionCost,
        address excessFeeRefundAddress,
        address callValueRefundAddress,
        uint256 gasLimit,
        uint256 maxFeePerGas,
        bytes calldata data
    ) external payable whenNotPaused onlyAllowed returns (uint256) {
        // this can only be called by UNISWAP_L1_TIMELOCK
        require(msg.sender == UNISWAP_L1_TIMELOCK, "NOT_UNISWAP_L1_TIMELOCK");
        // the retryable can only call UNISWAP_L2_FACTORY
        require(to == UNISWAP_L2_FACTORY, "NOT_TO_UNISWAP_L2_FACTORY");

        // ensure the user's deposit alone will make submission succeed
        if (msg.value < (maxSubmissionCost + l2CallValue + gasLimit * maxFeePerGas)) {
            revert InsufficientValue(
                maxSubmissionCost + l2CallValue + gasLimit * maxFeePerGas,
                msg.value
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

        // gas price and limit of 1 should never be a valid input, so instead they are used as
        // magic values to trigger a revert in eth calls that surface data without requiring a tx trace
        if (gasLimit == 1 || maxFeePerGas == 1)
            revert RetryableData(
                msg.sender,
                to,
                l2CallValue,
                msg.value,
                maxSubmissionCost,
                excessFeeRefundAddress,
                callValueRefundAddress,
                gasLimit,
                maxFeePerGas,
                data
            );

        uint256 submissionFee = calculateRetryableSubmissionFee(data.length, block.basefee);
        if (maxSubmissionCost < submissionFee)
            revert InsufficientSubmissionCost(submissionFee, maxSubmissionCost);

        return
            _deliverMessage(
                L1MessageType_submitRetryableTx,
                AddressAliasHelper.undoL1ToL2Alias(msg.sender),
                abi.encodePacked(
                    uint256(uint160(to)),
                    l2CallValue,
                    msg.value,
                    maxSubmissionCost,
                    uint256(uint160(excessFeeRefundAddress)),
                    uint256(uint160(callValueRefundAddress)),
                    gasLimit,
                    maxFeePerGas,
                    data.length,
                    data
                ),
                msg.value
            );
    }

    function _deliverToBridge(
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 amount
    ) internal override returns (uint256) {
        return
            IEthBridge(address(bridge)).enqueueDelayedMessage{value: amount}(
                kind,
                AddressAliasHelper.applyL1ToL2Alias(sender),
                messageDataHash
            );
    }
}
