// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract Invalid {
    uint256 public n;

    function invalid() public {
        assembly{ 
            invalid()
        }
    }
    
    function revertNoMessage() public {
        assembly {
            revert(0, 0)
        }
    }

    function revertWithMessage() public {
        bytes memory message = "ABCD";
        uint256 msgSize = message.length;
        assembly {
            revert(add(message,0x20), msgSize)
        }
    }

    function revertWithMemoryExpansion() public {
        assembly{
            let x := msize()
            let y := add(x, 0x20)
            revert(x, y)
        }
    }

    function revertInTryCatch() public {
        try this.revertWithMessage() {
            n = 3;
        } catch {
            n = 1;
        }
    }

    function revertInTryCatchWithMemoryExpansion() public {
        try this.revertWithMemoryExpansion() {
            n = 3;
        } catch {
            n = 1;
        }
    }

}
