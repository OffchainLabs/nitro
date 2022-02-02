//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

interface IChallenge {
    function asserter() external view returns (address);
    function challenger() external view returns (address);
    function lastMoveTimestamp() external view returns (uint256);
    function currentResponderTimeLeft() external view returns (uint256);

    function clearChallenge() external;
    function timeout() external;
}
