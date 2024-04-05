// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../bridge/ISequencerInbox.sol";
import "../precompiles/ArbRetryableTx.sol";
import "../precompiles/ArbSys.sol";

contract Simple {
    uint64 public counter;
    uint256 public difficulty;

    event CounterEvent(uint64 count);
    event RedeemedEvent(address caller, address redeemer);
    event NullEvent();
    event LogAndIncrementCalled(uint256 expected, uint256 have);

    function increment() external {
        counter++;
    }

    function logAndIncrement(uint256 expected) external {
        emit LogAndIncrementCalled(expected, counter);
        counter++;
    }

    function incrementEmit() external {
        counter++;
        emit CounterEvent(counter);
    }

    function incrementRedeem() external {
        // solhint-disable-next-line avoid-tx-origin
        require(msg.sender == tx.origin, "SENDER_NOT_ORIGIN");
        require(ArbSys(address(0x64)).wasMyCallersAddressAliased(), "NOT_ALIASED");
        counter++;
        emit RedeemedEvent(msg.sender, ArbRetryableTx(address(110)).getCurrentRedeemer());
    }

    function emitNullEvent() external {
        emit NullEvent();
    }

    function checkBlockHashes() external view returns (uint256) {
        require(blockhash(block.number - 1) != blockhash(block.number - 2), "SAME_BLOCK_HASH");
        return block.number;
    }

    function storeDifficulty() external {
        difficulty = block.difficulty;
    }

    function getBlockDifficulty() external view returns (uint256) {
        return difficulty;
    }

    function noop() external pure {}

    function pleaseRevert() external pure {
        revert("SOLIDITY_REVERTING");
    }

    function checkIsTopLevelOrWasAliased(bool useTopLevel, bool expected) public view {
        if (useTopLevel) {
            require(ArbSys(address(100)).isTopLevelCall() == expected, "UNEXPECTED_RESULT");
        } else {
            require(
                ArbSys(address(100)).wasMyCallersAddressAliased() == expected,
                "UNEXPECTED_RESULT"
            );
        }
    }

    function checkCalls(
        bool useTopLevel,
        bool directCase,
        bool staticCase,
        bool delegateCase,
        bool callcodeCase,
        bool callCase
    ) public {
        // DIRECT CALL
        if (useTopLevel) {
            require(ArbSys(address(100)).isTopLevelCall() == directCase, "UNEXPECTED_RESULT");
        } else {
            require(
                ArbSys(address(100)).wasMyCallersAddressAliased() == directCase,
                "UNEXPECTED_RESULT"
            );
        }

        // STATIC CALL
        this.checkIsTopLevelOrWasAliased(useTopLevel, staticCase);

        // DELEGATE CALL
        bytes memory data = abi.encodeWithSelector(
            this.checkIsTopLevelOrWasAliased.selector,
            useTopLevel,
            delegateCase
        );
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = address(this).delegatecall(data);
        require(success, "DELEGATE_CALL_FAILED");

        // CALLCODE
        data = abi.encodeWithSelector(
            this.checkIsTopLevelOrWasAliased.selector,
            useTopLevel,
            callcodeCase
        );
        assembly {
            success := callcode(gas(), address(), 0, add(data, 32), mload(data), 0, 0)
        }
        require(success, "CALLCODE_FAILED");

        // CALL
        data = abi.encodeWithSelector(
            this.checkIsTopLevelOrWasAliased.selector,
            useTopLevel,
            callCase
        );
        // solhint-disable-next-line avoid-low-level-calls
        (success, ) = address(this).call(data);
        require(success, "CALL_FAILED");
    }

    function checkGasUsed(address to, bytes calldata input) external view returns (uint256) {
        uint256 before = gasleft();
        // The inner call may revert, but we still want to return the amount of gas used,
        // so we ignore the result of this call.
        // solhint-disable-next-line avoid-low-level-calls
        // solc-ignore-next-line unused-call-retval
        to.staticcall{gas: before - 10000}(input); // forgefmt: disable-line
        return before - gasleft();
    }

    function postManyBatches(
        ISequencerInbox sequencerInbox,
        bytes memory batchData,
        uint256 numberToPost
    ) external {
        uint256 sequenceNumber = sequencerInbox.batchCount();
        uint256 delayedMessagesRead = sequencerInbox.totalDelayedMessagesRead();
        for (uint256 i = 0; i < numberToPost; i++) {
            sequencerInbox.addSequencerL2Batch(
                sequenceNumber,
                batchData,
                delayedMessagesRead,
                IGasRefunder(address(0)),
                0,
                0
            );
            sequenceNumber++;
        }
    }
}
