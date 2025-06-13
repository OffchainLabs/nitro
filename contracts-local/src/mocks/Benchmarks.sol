// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

contract Benchmarks {
    function fillBlockRecover() external payable {
        bytes32 bridgeToNova = 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
        address cryptoIsCute = 0x361594F5429D23ECE0A88E4fBE529E1c49D524d8;
        uint8 v = 27;
        bytes32 r = 0xc6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f;
        bytes32 s = 0x5fdbcefe2675e96219cdae57a7894280bf80fd40d44ce146a35e169ea6a78fd3;
        while (true) {
            require(ecrecover(bridgeToNova, v, r, s) == cryptoIsCute, "WRONG_ARBINAUT");
        }
    }

    function fillBlockMulMod() external payable {
        uint256 value = 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
        while (true) {
            value = mulmod(
                value,
                0xc6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f,
                0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f
            );
        }
    }

    function fillBlockHash() external payable {
        bytes32 hash = 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
        while (true) {
            hash = keccak256(abi.encodePacked(hash));
        }
    }

    function fillBlockAdd() external payable {
        uint256 value = 0;
        while (true) {
            unchecked {
                value += 0xeddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786;
            }
        }
    }

    function fillBlockQuickStep() external payable {
        uint256 value = 0;
        while (true) {
            value = msg.value;
        }
    }
}
