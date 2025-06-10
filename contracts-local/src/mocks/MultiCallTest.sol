// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

/*
 * this contract is the solidity equivalent of stylus multicall test contract
 * it should only be used for stylus tests, and it ignores good solidity good practices
 */

contract MultiCallTest {
    event Called(address addr, uint8 count, bool success, bytes returnData);
    event Storage(bytes32 slot, bytes32 data, bool write);

    function getBE(bytes calldata data, uint8 numBytes) internal pure returns (uint256) {
        uint256 res = 0;
        for (uint8 i = 0; i < numBytes; i++) {
            res = res << 8;
            res = res | uint8(data[i]);
        }
        return res;
    }

    // solhint-disable no-complex-fallback
    // solhint-disable reason-string
    // solhint-disable avoid-low-level-calls
    // solhint-disable-next-line prettier/prettier
    fallback(
        bytes calldata input
    ) external payable returns (bytes memory) {
        require(input.length > 0);
        uint8 count = uint8(input[0]);
        input = input[1:];

        // combined output of all calls
        bytes memory output;

        for (uint8 i = 0; i < count; i++) {
            uint32 length = uint32(getBE(input, 4));
            input = input[4:];

            bytes calldata curr = input[:length];
            input = input[length:];

            uint8 kind = uint8(curr[0]);
            curr = curr[1:];

            if (kind & 0xf0 == 0x0) {
                // call
                uint256 value;
                if (kind & 0x3 == 0) {
                    value = getBE(curr, 32);
                    curr = curr[32:];
                }

                address addr = address(bytes20(curr[:20]));
                bytes calldata data = curr[20:];
                bytes memory out;
                bool success;

                if (kind & 0x3 == 0) {
                    (success, out) = addr.call{value: value}(data);
                } else if (kind & 0x3 == 1) {
                    (success, out) = addr.delegatecall(data);
                } else if (kind & 0x3 == 2) {
                    (success, out) = addr.staticcall(data);
                } else {
                    revert("unknown call kind");
                }
                if (!success) {
                    if (kind & 0x4 == 0) {
                        uint256 len = out.length;
                        if (len > 0) {
                            assembly {
                                revert(add(out, 32), len)
                            }
                        } else {
                            revert();
                        }
                    }
                    out = "";
                }
                if (kind & 0x8 != 0) {
                    emit Called(addr, count, success, out);
                }
                output = bytes.concat(output, out);
            } else if (kind & 0xf0 == 0x10) {
                // storage
                bytes32 slot = bytes32(curr[:32]);
                curr = curr[32:];
                bytes32 data;
                bool write;
                if (kind & 0x3 == 0) {
                    data = bytes32(curr[:32]);
                    write = true;
                    assembly {
                        sstore(slot, data)
                    }
                } else if (kind & 0x3 == 1) {
                    write = false;
                    assembly {
                        data := sload(slot)
                    }
                    output = bytes.concat(output, data);
                } else {
                    revert("unknown storage kind");
                }
                if (kind & 0x8 != 0) {
                    emit Storage(slot, data, write);
                }
            } else {
                revert("unknown command");
            }
        }

        return output;
    }
}
