// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

import {Script, VmSafe, console} from "forge-std/Script.sol";
import {Invalid} from "../src/Invalid.sol";

contract InvalidScript is Script {
    function setUp() public {}

    function run() public {
        vm.startBroadcast(address(0x3f1Eae7D46d88F08fc2F8ed27FCb2AB183EB2d0E));
        Invalid invalid = new Invalid();
         try invalid.invalid() {
             console.log("invalid did not revert");
         } catch (bytes memory reason) {
             console.logBytes(reason);
         }
         try invalid.revertInTryCatch() {
             console.log("revertInTryCatch did not revert");
         } catch (bytes memory reason) {
             console.logBytes(reason);
         }
         try invalid.revertInTryCatchWithMemoryExpansion() {
             console.log("revertInTryCatchWithMemoryExpansion did not revert");
         } catch (bytes memory reason) {
             console.logBytes(reason);
         }
         try invalid.revertNoMessage() {
             console.log("revertNoMessage did not revert");
         } catch (bytes memory reason) {
             console.logBytes(reason);
         }
        try invalid.revertWithMessage() {
            console.log("revertWithMessage did not revert");
        } catch (bytes memory reason) {
            console.logBytes(reason);
        }
        vm.stopBroadcast();
    }
}
