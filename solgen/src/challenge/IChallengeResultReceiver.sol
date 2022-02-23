//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

interface IChallengeResultReceiver {
	function completeChallenge(uint256 challengeIndex, address winner, address loser) external;
}
