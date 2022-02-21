//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../challenge/IChallengeResultReceiver.sol";

contract MockResultReceiver is IChallengeResultReceiver {
	address public winner;
	address public loser;
	uint256 public challengeIndex;

event ChallengeCompleted(uint256 indexed challengeIndex, address indexed winner, address indexed loser);

	function completeChallenge(uint256 challengeIndex_, address winner_, address loser_) external override {
		winner = winner_;
		loser = loser_;
		challengeIndex = challengeIndex_;
		emit ChallengeCompleted(challengeIndex, winner_, loser_);
	}
}
