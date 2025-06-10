// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.24;

/*
 * HostioTest is a test contract used to compare EVM with Stylus.
 */
contract HostioTest {
    function exitEarly() external pure {
        assembly {
            stop()
        }
    }

    function transientLoadBytes32(
        bytes32 key
    ) external view returns (bytes32) {
        bytes32 data;
        assembly {
            data := tload(key)
        }
        return data;
    }

    function transientStoreBytes32(bytes32 key, bytes32 value) external {
        assembly {
            // solc-ignore-next-line transient-storage
            tstore(key, value)
        }
    }

    function returnDataSize() external pure returns (uint256) {
        uint256 size;
        assembly {
            size := returndatasize()
        }
        return size;
    }

    function emitLog(
        bytes calldata _data,
        int8 n,
        bytes32 t1,
        bytes32 t2,
        bytes32 t3,
        bytes32 t4
    ) external {
        bytes memory data = _data;
        if (n == 0) {
            assembly {
                log0(add(data, 32), mload(data))
            }
        } else if (n == 1) {
            assembly {
                log1(add(data, 32), mload(data), t1)
            }
        } else if (n == 2) {
            assembly {
                log2(add(data, 32), mload(data), t1, t2)
            }
        } else if (n == 3) {
            assembly {
                log3(add(data, 32), mload(data), t1, t2, t3)
            }
        } else if (n == 4) {
            assembly {
                log4(add(data, 32), mload(data), t1, t2, t3, t4)
            }
        } else {
            revert("invalid n for emit log");
        }
    }

    function accountBalance(
        address account
    ) external view returns (uint256) {
        return account.balance;
    }

    function accountCode(
        address account
    ) external view returns (bytes memory) {
        uint256 size = 10000;
        bytes memory code = new bytes(size);
        assembly {
            extcodecopy(account, add(code, 32), 0, size)
            size := extcodesize(account)
            mstore(code, size)
        }
        return code;
    }

    function accountCodeSize(
        address account
    ) external view returns (uint256) {
        uint256 size;
        assembly {
            size := extcodesize(account)
        }
        return size;
    }

    function accountCodehash(
        address account
    ) external view returns (bytes32) {
        bytes32 hash;
        assembly {
            hash := extcodehash(account)
        }
        return hash;
    }

    function evmGasLeft() external view returns (uint256) {
        return gasleft();
    }

    function evmInkLeft() external view returns (uint256) {
        return gasleft();
    }

    function blockBasefee() external view returns (uint256) {
        return block.basefee;
    }

    function chainid() external view returns (uint256) {
        return block.chainid;
    }

    function blockCoinbase() external view returns (address) {
        return block.coinbase;
    }

    function blockGasLimit() external view returns (uint256) {
        return block.gaslimit;
    }

    function blockNumber() external view returns (uint256) {
        return block.number;
    }

    function blockTimestamp() external view returns (uint256) {
        return block.timestamp;
    }

    function contractAddress() external view returns (address) {
        return address(this);
    }

    function mathDiv(uint256 a, uint256 b) external pure returns (uint256) {
        return a / b;
    }

    function mathMod(uint256 a, uint256 b) external pure returns (uint256) {
        return a % b;
    }

    function mathPow(uint256 a, uint256 b) external pure returns (uint256) {
        uint256 result;
        assembly {
            result := exp(a, b)
        }
        return result;
    }

    function mathAddMod(uint256 a, uint256 b, uint256 c) external pure returns (uint256) {
        return addmod(a, b, c);
    }

    function mathMulMod(uint256 a, uint256 b, uint256 c) external pure returns (uint256) {
        return mulmod(a, b, c);
    }

    function msgSender() external view returns (address) {
        return msg.sender;
    }

    function msgValue() external payable returns (uint256) {
        return msg.value;
    }

    function keccak(
        bytes calldata preimage
    ) external pure returns (bytes32) {
        return keccak256(preimage);
    }

    function txGasPrice() external view returns (uint256) {
        return tx.gasprice;
    }

    function txInkPrice() external view returns (uint256) {
        return tx.gasprice;
    }

    function txOrigin() external view returns (address) {
        return tx.origin;
    }
}
