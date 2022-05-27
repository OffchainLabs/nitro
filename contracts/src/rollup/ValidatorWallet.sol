// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/IChallengeManager.sol";
import "../libraries/DelegateCallAware.sol";
import "../libraries/IGasRefunder.sol";
import {IRollupUserAbs} from "./IRollupLogic.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";


error BadArrayLength(uint256 expected, uint256 actual);
error NotCallerOrOwner(address actual);
error OnlyOwnerFunctionSig(address expected, address actual);

contract ValidatorWallet is OwnableUpgradeable, DelegateCallAware, GasRefundEnabled {
    using Address for address;

    /// @dev the executor is allowed to call non-admin functions and get refunded
    address public executor;

    /// @dev function signatures that can only be called by the owner
    mapping(bytes4 => bool) public onlyOwnerFuncSigs;

    modifier onlyExecutorOrOwner {
        if(executor != _msgSender() && owner() != _msgSender())
            revert NotCallerOrOwner(_msgSender());
        _;
    }

    event ExecutorTransferred(address indexed previousExecutor, address indexed newExecutor);

    function setExecutor(address newExecutor) external onlyExecutorOrOwner {
        address prevExecutor = executor;
        executor = newExecutor;
        emit ExecutorTransferred(prevExecutor, newExecutor);
    }

    function initialize(address _executor, address _owner) external initializer onlyDelegated {
        __Ownable_init();
        executor = _executor;
        emit ExecutorTransferred(address(0), _executor);
        transferOwnership(_owner);
        
        onlyOwnerFuncSigs[IRollupUserAbs.withdrawStakerFunds.selector] = true;
        emit OnlyOwnerFuncSigUpdated(IRollupUserAbs.withdrawStakerFunds.selector, true);

        onlyOwnerFuncSigs[IRollupUserAbs.createChallenge.selector] = true;
        emit OnlyOwnerFuncSigUpdated(IRollupUserAbs.createChallenge.selector, true);
    }

    event OnlyOwnerFuncSigUpdated(bytes4 indexed sig, bool val);

    /// @notice updates the function signatures which only the owner is allowed to call
    function setOnlyOwnerFunctionSigs(bytes4[] calldata funcSigs, bool[] calldata isSet) external onlyOwner {
        if(funcSigs.length != isSet.length) revert BadArrayLength(funcSigs.length, isSet.length);
        unchecked {
            for(uint256 i = 0; i< funcSigs.length; ++i) {
                onlyOwnerFuncSigs[funcSigs[i]] = isSet[i];
                emit OnlyOwnerFuncSigUpdated(funcSigs[i], isSet[i]);
            }
        }
    }

    /// @dev reverts if the current function can't be called
    function validateExecuteTransaction(bytes calldata data) public view {
        bytes4 funcSig = data.length < 4
            ? bytes4(data[:data.length])
            : bytes4(data[:4]);

        if(onlyOwnerFuncSigs[funcSig] && owner() != _msgSender())
            revert OnlyOwnerFunctionSig(owner(), _msgSender());
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
    ) public payable onlyExecutorOrOwner refundsGas(gasRefunder) {
        uint256 numTxes = data.length;
        for (uint256 i = 0; i < numTxes; i++) {
            if (data[i].length > 0) require(destination[i].isContract(), "NO_CODE_AT_ADDR");
            validateExecuteTransaction(data[i]);
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
    ) public payable onlyExecutorOrOwner refundsGas(gasRefunder) {
        if (data.length > 0) require(destination.isContract(), "NO_CODE_AT_ADDR");
        validateExecuteTransaction(data);
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

    function timeoutChallenges(IChallengeManager manager, uint64[] calldata challenges) external {
        timeoutChallengesWithGasRefunder(IGasRefunder(address(0)), manager, challenges);
    }

    function timeoutChallengesWithGasRefunder(
        IGasRefunder gasRefunder,
        IChallengeManager manager,
        uint64[] calldata challenges
    ) public onlyExecutorOrOwner refundsGas(gasRefunder) {
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

    receive() payable external {

    }
}
