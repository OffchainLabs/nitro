// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/IChallengeManager.sol";
import "../libraries/DelegateCallAware.sol";
import "../libraries/IGasRefunder.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";


/// @dev a user can append extra useless data when calling functions
/// this error is thrown if a user attempts to get
error BadCalldataLength(uint256 max, uint256 actual);

uint256 constant FUNC_SELECTOR_CALLDATA_OVERHEAD = 4;
uint256 constant DYNAMIC_ARRAY_CALLDATA_OVERHEAD = 32;
uint256 constant ADDRESS_CALLDATA_LENGTH = 32;
uint256 constant UINT256_CALLDATA_LENGTH = 32;

contract ValidatorWallet is OwnableUpgradeable, DelegateCallAware, GasRefundEnabled {
    using Address for address;

    function initialize() external initializer onlyDelegated {
        __Ownable_init();
    }

    function executeTransactions(
        bytes[] calldata data,
        address[] calldata destination,
        uint256[] calldata amount
    ) external payable {
        executeTransactionsWithGasRefunder(IGasRefunder(address(0)), data, destination, amount);
    }

    function executeTransactionsWithGasRefunder(
        IGasRefunder gasRefunder,
        bytes[] calldata data,
        address[] calldata destination,
        uint256[] calldata amount
    ) public payable onlyOwner refundsGasWithCalldata(gasRefunder, payable(msg.sender)) {
        if(gasRefunder != IGasRefunder(address(0))) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            uint256 expectedCalldataSize =
                FUNC_SELECTOR_CALLDATA_OVERHEAD +
                ADDRESS_CALLDATA_LENGTH +
                DYNAMIC_ARRAY_CALLDATA_OVERHEAD + data.length +
                DYNAMIC_ARRAY_CALLDATA_OVERHEAD + (destination.length * ADDRESS_CALLDATA_LENGTH) +
                DYNAMIC_ARRAY_CALLDATA_OVERHEAD + (amount.length * UINT256_CALLDATA_LENGTH);
            
            if(calldataSize > expectedCalldataSize) revert BadCalldataLength(expectedCalldataSize, calldataSize);
        }

        uint256 numTxes = data.length;
        for (uint256 i = 0; i < numTxes; i++) {
            if (data[i].length > 0) require(destination[i].isContract(), "NO_CODE_AT_ADDR");
            // We use a low level call here to allow for contract and non-contract calls
            // solhint-disable-next-line avoid-low-level-calls
            (bool success, ) = address(destination[i]).call{value: amount[i]}(data[i]);
            if (!success) {
                assembly {
                    let ptr := mload(0x40)
                    let size := returndatasize()
                    returndatacopy(ptr, 0, size)
                    revert(ptr, size)
                }
            }
        }
    }

    function executeTransaction(
        bytes calldata data,
        address destination,
        uint256 amount
    ) external payable {
        executeTransactionWithGasRefunder(IGasRefunder(address(0)), data, destination, amount);
    }

    function executeTransactionWithGasRefunder(
        IGasRefunder gasRefunder,
        bytes calldata data,
        address destination,
        uint256 amount
    ) public payable onlyOwner refundsGasWithCalldata(gasRefunder, payable(msg.sender)) {
        if(gasRefunder != IGasRefunder(address(0))) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            uint256 expectedCalldataSize =
                FUNC_SELECTOR_CALLDATA_OVERHEAD +
                ADDRESS_CALLDATA_LENGTH +
                data.length +
                ADDRESS_CALLDATA_LENGTH +
                UINT256_CALLDATA_LENGTH;
            
            if(calldataSize > expectedCalldataSize) revert BadCalldataLength(expectedCalldataSize, calldataSize);
        }

        if (data.length > 0) require(destination.isContract(), "NO_CODE_AT_ADDR");
        // We use a low level call here to allow for contract and non-contract calls
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = destination.call{value: amount}(data);
        if (!success) {
            assembly {
                let ptr := mload(0x40)
                let size := returndatasize()
                returndatacopy(ptr, 0, size)
                revert(ptr, size)
            }
        }
    }

    function timeoutChallenges(IChallengeManager manager, uint64[] calldata challenges)
        external
    {
        timeoutChallengesWithGasRefunder(IGasRefunder(address(0)), manager, challenges);
    }

    function timeoutChallengesWithGasRefunder(
        IGasRefunder gasRefunder,
        IChallengeManager manager,
        uint64[] calldata challenges
    )
        public
        onlyOwner
    {
        if(gasRefunder != IGasRefunder(address(0))) {
            uint256 calldataSize;
            assembly {
                calldataSize := calldatasize()
            }
            uint256 expectedCalldataSize =
                FUNC_SELECTOR_CALLDATA_OVERHEAD +
                ADDRESS_CALLDATA_LENGTH +
                ADDRESS_CALLDATA_LENGTH +
                DYNAMIC_ARRAY_CALLDATA_OVERHEAD + (challenges.length * UINT256_CALLDATA_LENGTH);
            
            if(calldataSize > expectedCalldataSize) revert BadCalldataLength(expectedCalldataSize, calldataSize);
        }
        uint256 challengesCount = challenges.length;
        for (uint256 i = 0; i < challengesCount; i++) {
            try manager.timeout(challenges[i]) {} catch (bytes memory error) {
                if (error.length == 0) {
                    // Assume out of gas
                    // We need to revert here so gas estimation works
                    require(false, "GAS");
                }
            }
        }
    }
}
