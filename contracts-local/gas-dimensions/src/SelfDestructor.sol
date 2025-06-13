// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Counter} from "./Counter.sol";

contract SelfDestructor {
    function warmSelfDestructor(address who) public {
        Counter counter = Counter(who);
        counter.setNumber(1);
        selfDestruct(who);
    }

    function warmEmptySelfDestructor(address who) public {
        (bool success,) = who.call("");
        selfDestruct(who);
    }

    function selfDestruct(address who) public {
        selfdestruct(payable(who));
    }

    receive() external payable {}

    fallback() external payable {}
}
