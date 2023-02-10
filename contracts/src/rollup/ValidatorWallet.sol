// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../challenge/IOldChallengeManager.sol";
import "../libraries/DelegateCallAware.sol";
import "../libraries/IGasRefunder.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

/// @dev thrown when arrays provided don't have the expected length
error BadArrayLength(uint256 expected, uint256 actual);

/// @dev thrown when a function is called by an address that isn't the owner nor a executor
error NotExecutorOrOwner(address actual);

/// @dev thrown when the particular address can't be called by an executor
error OnlyOwnerDestination(address expected, address actual, address destination);

/// @dev thrown when eth withdrawal tx fails
error WithdrawEthFail(address destination);

contract ValidatorWallet is OwnableUpgradeable, DelegateCallAware, GasRefundEnabled {
    using Address for address;

    /// @dev a executor is allowed to call only certain contracts
    mapping(address => bool) public executors;

    /// @dev allowed addresses which can be called by an executor
    mapping(address => bool) public allowedExecutorDestinations;

    modifier onlyExecutorOrOwner() {
        if (!executors[_msgSender()] && owner() != _msgSender())
            revert NotExecutorOrOwner(_msgSender());
        _;
    }

    event ExecutorUpdated(address indexed executor, bool isExecutor);

    /// @dev updates the executor addresses
    function setExecutor(address[] calldata newExecutors, bool[] calldata isExecutor)
        external
        onlyOwner
    {
        if (newExecutors.length != isExecutor.length)
            revert BadArrayLength(newExecutors.length, isExecutor.length);
        unchecked {
            for (uint64 i = 0; i < newExecutors.length; ++i) {
                executors[newExecutors[i]] = isExecutor[i];
                emit ExecutorUpdated(newExecutors[i], isExecutor[i]);
            }
        }
    }

    function initialize(
        address _executor,
        address _owner,
        address[] calldata initialExecutorAllowedDests
    ) external initializer onlyDelegated {
        __Ownable_init();
        transferOwnership(_owner);

        executors[_executor] = true;
        emit ExecutorUpdated(_executor, true);

        unchecked {
            for (uint64 i = 0; i < initialExecutorAllowedDests.length; ++i) {
                allowedExecutorDestinations[initialExecutorAllowedDests[i]] = true;
                emit AllowedExecutorDestinationsUpdated(initialExecutorAllowedDests[i], true);
            }
        }
    }

    event AllowedExecutorDestinationsUpdated(address indexed destination, bool isSet);

    /// @notice updates the destination addresses which executors are allowed to call
    function setAllowedExecutorDestinations(address[] calldata destinations, bool[] calldata isSet)
        external
        onlyOwner
    {
        if (destinations.length != isSet.length)
            revert BadArrayLength(destinations.length, isSet.length);
        unchecked {
            for (uint256 i = 0; i < destinations.length; ++i) {
                allowedExecutorDestinations[destinations[i]] = isSet[i];
                emit AllowedExecutorDestinationsUpdated(destinations[i], isSet[i]);
            }
        }
    }

    /// @dev reverts if the current function can't be called
    function validateExecuteTransaction(address destination) public view {
        if (!allowedExecutorDestinations[destination] && owner() != _msgSender())
            revert OnlyOwnerDestination(owner(), _msgSender(), destination);
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
        if (numTxes != destination.length) revert BadArrayLength(numTxes, destination.length);
        if (numTxes != amount.length) revert BadArrayLength(numTxes, amount.length);

        for (uint256 i = 0; i < numTxes; i++) {
            if (data[i].length > 0) require(destination[i].isContract(), "NO_CODE_AT_ADDR");
            validateExecuteTransaction(destination[i]);
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
        validateExecuteTransaction(destination);
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

    function timeoutChallenges(IOldChallengeManager manager, uint64[] calldata challenges) external {
        timeoutChallengesWithGasRefunder(IGasRefunder(address(0)), manager, challenges);
    }

    function timeoutChallengesWithGasRefunder(
        IGasRefunder gasRefunder,
        IOldChallengeManager manager,
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

    receive() external payable {}

    /// @dev allows the owner to withdraw eth held by this contract
    function withdrawEth(uint256 amount, address destination) external onlyOwner {
        // solhint-disable-next-line avoid-low-level-calls
        (bool success, ) = destination.call{value: amount}("");
        if (!success) revert WithdrawEthFail(destination);
    }
}
