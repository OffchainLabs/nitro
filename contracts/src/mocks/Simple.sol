// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../precompiles/ArbRetryableTx.sol";
import "../precompiles/ArbSys.sol";

contract Simple {
    uint64 public counter;

    event CounterEvent(uint64 count);
    event RedeemedEvent(address caller, address redeemer);
    event NullEvent();

    function increment() external {
        counter++;
    }

    function incrementEmit() external {
        counter++;
        emit CounterEvent(counter);
    }

    function incrementRedeem() external {
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

    function getBlockDifficulty() external view returns (uint256) {
        return block.difficulty;
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
        (success, ) = address(this).call(data);
        require(success, "CALL_FAILED");
    }
}
