// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

/// @notice Test contract for exercising different call opcodes for address filtering tests
contract AddressFilterTest {
    uint256 public dummy;
    event CallResult(bool success);

    /// @notice Makes a CALL to the target address
    function callTarget(address target) external returns (bool success) {
        (success,) = target.call("");
        emit CallResult(success);
    }

    /// @notice Makes a STATICCALL to the target address (view function for eth_call)
    function staticcallTarget(address target) external view returns (bool success) {
        (success,) = target.staticcall("");
    }

    /// @notice Makes a STATICCALL to the target address within a transaction
    function staticcallTargetInTx(address target) external returns (bool success) {
        dummy++;
        (success,) = target.staticcall("");
        emit CallResult(success);
    }

    /// @notice Simple function that can be called by other contracts
    function noop() external pure {}

    /// @notice Deploys a contract using CREATE and returns the address
    function createContract() external returns (address created) {
        // Deploy minimal contract (just returns)
        bytes memory bytecode = hex"6080604052348015600f57600080fd5b50603580601d6000396000f3fe6080604052600080fdfea164736f6c6343000811000a";
        assembly {
            created := create(0, add(bytecode, 0x20), mload(bytecode))
        }
    }

    /// @notice Deploys a contract using CREATE2 with a salt and returns the address
    function create2Contract(bytes32 salt) external returns (address created) {
        bytes memory bytecode = hex"6080604052348015600f57600080fd5b50603580601d6000396000f3fe6080604052600080fdfea164736f6c6343000811000a";
        assembly {
            created := create2(0, add(bytecode, 0x20), mload(bytecode), salt)
        }
    }

    /// @notice Computes the CREATE2 address for a given salt (for pre-computing the address to filter)
    function computeCreate2Address(bytes32 salt) external view returns (address) {
        bytes memory bytecode = hex"6080604052348015600f57600080fd5b50603580601d6000396000f3fe6080604052600080fdfea164736f6c6343000811000a";
        bytes32 hash = keccak256(abi.encodePacked(bytes1(0xff), address(this), salt, keccak256(bytecode)));
        return address(uint160(uint256(hash)));
    }

    /// @notice Selfdestructs this contract and sends balance to beneficiary
    function selfDestructTo(address payable beneficiary) external {
        selfdestruct(beneficiary);
    }

    /// @notice Allow contract to receive ETH
    receive() external payable {}
}
