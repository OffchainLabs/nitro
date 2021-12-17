//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../challenge/IChallengeResultReceiver.sol";

contract MockResultReceiver is IChallengeResultReceiver {
	address public winner;
	address public loser;

	event ChallengeCompleted(address indexed challenge, address indexed winner, address indexed loser);

	function completeChallenge(address winner_, address loser_) external override {
		winner = winner_;
		loser = loser_;
		emit ChallengeCompleted(msg.sender, winner_, loser_);
	}
}
