// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import {Status, IAssertionChain, IChallengeManager} from "./challengeV2/DataEntities.sol";

// Questions
// 2. I have a different idea of when the challenge endtime should be. I think it should be 1 challenge period after the second child creation
//    not 2 challenge periods after the first child creation.
// 3. Should we restructure the challenges into a single contract?
// 4. use timestamps or block numbers? always timestamps atm
// 5. We dont allow a challenge to be created if the ps has a pstimer > challengeperiod, but it may not be
// on a confirmed branch so this would be meaningless?

// CHRIS: TODO: check non zeros in all  functions and constructors
// CHRIS: TODO: draw timings somehow, at least try it
// CHRIS: TODO: wherever we check that an assertion of vertex exists, should we also be checking the status?
// CHRIS: TODO: timings shouldnt be checked all over the place - it's too weird to reason about
// CHRIS: TODO: replace all requires with custom errors;
// CHRIS: TODO: For all arguments implicit and explicit to every function, consider if there's a restriction

// CHRIS: TODO: when winning a challenge you can claim back your ministake. Leaf stake
// CHRIS: TODO: when your assertion is reject you lose you major stake. Assertion stake.

// INVARIANTS
// If an assertion exists, the previous assertion also exists

struct Assertion {
    bytes32 predecessorId;
    bytes32 successionChallenge;
    bool isFirstChild;
    uint256 secondChildCreationTime;
    uint256 firstChildCreationTime;
    // CHRIS: TODO: where is this in the spec?
    // CHRIS: TODO: we can remove these from the contents? - and the prev+height? -
    // CHRIS: TODO: since we're always using the id to look them up
    bytes32 stateHash; // CHRIS: TODO: this is a general state hash, not the same as the state root in an ethereum block. This hash contains everything, including inboxmessage count, the block hash and the send root
    uint256 height;
    Status status;
    uint256 inboxMsgCountSeen;
}

interface IInbox {
    function msgCount() external returns (uint256);
}

contract AssertionChain is IAssertionChain {
    IChallengeManager challengeManager;
    mapping(bytes32 => Assertion) public assertions;
    uint256 public immutable stakeAmount = 100 ether; // CHRIS: TODO: update
    uint256 public challengePeriodSeconds;
    IInbox inbox;

    constructor(bytes32 stateHash, uint256 _challengePeriodSeconds) public {
        challengePeriodSeconds = _challengePeriodSeconds;
        bytes32 assertionId = bytes32(0);
        assertions[assertionId] = Assertion({
            predecessorId: assertionId,
            successionChallenge: 0,
            isFirstChild: false,
            firstChildCreationTime: 0,
            secondChildCreationTime: 0,
            stateHash: stateHash,
            height: 0,
            status: Status.Confirmed,
            inboxMsgCountSeen: 0
        });
    }

    function challengeManagerAddr() public view returns (address) {
        return address(challengeManager);
    }

    // CHRIS: TODO: expensive to do from the challenge contract - could just ask for specific properties?
    function getAssertion(bytes32 id) public view returns (Assertion memory) {
        return assertions[id];
    }

    function assertionExists(bytes32 assertionId) public view returns (bool) {
        return assertions[assertionId].stateHash != 0;
    }

    function getPredecessorId(bytes32 assertionId) public view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].predecessorId;
    }

    function getHeight(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].height;
    }

    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].inboxMsgCountSeen;
    }

    function getStateHash(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].stateHash;
    }

    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].successionChallenge;
    }

    function isFirstChild(bytes32 assertionId) external view returns (bool) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].isFirstChild;
    }

    function getFirstChildCreationTime(bytes32 assertionId) external view returns (uint256) {
        require(assertionExists(assertionId), "Assertion does not exist");
        return assertions[assertionId].firstChildCreationTime;
    }

    function createNewAssertion(
        bytes32 stateHash,
        uint256 height,
        bytes32 predecessorId
    ) external {
        // CHRIS: TODO: library on the assertion
        // CHRIS: TODO: consider if we should include the prev here? we need to right? but the reference below should be to the state hash
        bytes32 assertionId = keccak256(abi.encodePacked(stateHash, height, predecessorId));

        // assertions are always unique as they consume a deterministic number of inbox messages
        // so two different correct assertions do not exist.
        require(!assertionExists(assertionId), "Assertion already exists");

        // CHRIS: TODO: staker checks here - msg.sender has put down stake and is not staked elsewhere, then update the staker location

        require(assertionExists(predecessorId), "Previous assertion does not exist");
        require(
            previousAssertion(assertionId).status != Status.Rejected,
            "Previous assertion rejected"
        );
        require(
            previousAssertion(assertionId).height < height,
            "Height not greater than predecessor"
        );

        bool hasFirstChild = assertions[predecessorId].firstChildCreationTime != 0;
        if (!hasFirstChild) {
            // if this is the first child then we update the prev
            assertions[predecessorId].firstChildCreationTime = block.timestamp;
        } else {
            require(
                block.timestamp <
                    previousAssertion(assertionId).firstChildCreationTime + challengePeriodSeconds,
                "Too late to create sibling"
            );

            if (assertions[predecessorId].secondChildCreationTime == 0) {
                // has the first child creation time passed a certain point?
                // do we allow siblings to be created after then since in a challenge they should have no time right

                // if this is the second child then we update the prev
                assertions[predecessorId].secondChildCreationTime = block.timestamp;
            }
        }

        assertions[assertionId] = Assertion({
            predecessorId: predecessorId,
            successionChallenge: 0,
            isFirstChild: !hasFirstChild,
            firstChildCreationTime: 0,
            secondChildCreationTime: 0,
            stateHash: stateHash,
            height: height,
            status: Status.Pending,
            // TODO: Initialize inbox in constructor.
            inboxMsgCountSeen: 0
        });
    }

    function addStake() external payable {
        // CHRIS: TODO: moving stake around is tricky
        // CHRIS: TODO: can you stake on more than one assertion, even they are direct ancestor?
        // CHRIS: TODO: do we really allow "any validator" at the top level to take part in any sub challenge
        require(msg.value == stakeAmount, "Correct stake not provided");
    }

    // CHRIS: TODO: initialisation with an empty assertion - what's the genesis state?

    function previousAssertion(bytes32 assertionId) internal view returns (Assertion storage) {
        return assertions[assertions[assertionId].predecessorId];
    }

    function updateChallengeManager(IChallengeManager _challengeManager) external {
        // CHRIS: TODO: this needs access control
        challengeManager = _challengeManager;
    }

    error NotRejectable(bytes32 assertionId);

    function rejectAssertion(bytes32 assertionId) external {
        require(assertionExists(assertionId), "Assertion does not exist");

        // we can only reject pending assertions
        require(assertions[assertionId].status == Status.Pending, "Assertion is not pending");

        // CHRIS: TODO: what happens to stake when we reject a assertion, or confirm it?

        if (previousAssertion(assertionId).status == Status.Rejected) {
            // the previous assertion was rejected
            assertions[assertionId].status = Status.Rejected;
        } else {
            // CHRIS: TODO: re-arrange this block, it's ugly

            bytes32 successionChallenge = previousAssertion(assertionId).successionChallenge;
            if (successionChallenge == 0) {
                revert NotRejectable(assertionId);
            }

            // CHRIS: TODO: external call, careful!
            bytes32 winningClaim = challengeManager.winningClaim(successionChallenge);
            // does the winner return 0
            if (winningClaim == bytes32(0)) {
                revert NotRejectable(assertionId);
            }

            if (winningClaim == assertionId) {
                revert NotRejectable(assertionId);
            }

            assertions[assertionId].status = Status.Rejected;
        }
    }

    // CHRIS: TODO: create assertion lib

    // CHRIS: TODO: better confirm/rejcet errors
    error NotConfirmable(bytes32 assertionId);

    function confirmAssertion(bytes32 assertionId) external {
        require(assertionExists(assertionId), "Assertion does not exist");

        require(
            previousAssertion(assertionId).status == Status.Confirmed,
            "Previous assertion not confirmed"
        );

        // CHRIS: TODO: add a test for this:
        // bad pattern here - create a test case for it, shouldnt be possible now
        // 1. create child
        // 2. confirm child by waiting for timeout
        // 3. create second child
        // 4. create challenge

        // CHRIS: TODO: this pattern and above in reject isnt nice
        if (
            previousAssertion(assertionId).secondChildCreationTime == 0 &&
            block.timestamp >
            previousAssertion(assertionId).firstChildCreationTime + challengePeriodSeconds
        ) {
            assertions[assertionId].status = Status.Confirmed;
        } else {
            bytes32 successionChallenge = previousAssertion(assertionId).successionChallenge;
            if (successionChallenge == 0) {
                revert NotConfirmable(assertionId);
            }

            // CHRIS: TODO: external call, careful!
            bytes32 winner = challengeManager.winningClaim(successionChallenge);
            if (winner != assertionId) {
                revert NotRejectable(assertionId);
            }

            assertions[assertionId].status = Status.Confirmed;
        }
    }

    function createSuccessionChallenge(bytes32 assertionId) external {
        require(assertionExists(assertionId), "Assertion does not exist");

        // CHRIS: TODO: but what if a much further parent is rejectable
        // we could get rejected later
        // from that point of view, why does it matter then if we start on a rejected branch? it will just be immediately rejectable?
        require(assertions[assertionId].status != Status.Rejected, "Assertion already rejected");

        require(assertions[assertionId].successionChallenge == 0, "Challenge already created");

        require(
            assertions[assertionId].secondChildCreationTime != 0,
            "At least two children not created"
        );

        // CHRIS: TODO: I think this should be secondChildTime + 1 challenge period, and in the endTime of BlockChallenge below
        // CHRIS: TODO: do we have this requirement in the new paper?
        require(
            block.timestamp <
                assertions[assertionId].firstChildCreationTime + (2 * challengePeriodSeconds),
            "Too late to challenge"
        );
        // CHRIS: TODO: answer to the above^^
        // CHRIS: TODO: if we put the challenge end time right at the start, then it will end very quickly after the pstimer
        // CHRIS: TODO: condition has been reached. This is a good reason not to ensure there is always plenty of time
        // CHRIS: TODO: but is there? Ok, so you do the following. You wait - honest person shouldnt wait
        // CHRIS: TODO: so if you're honest then what? dont wait, start the challenge straight away, then
        // CHRIS: TODO: you can be sure that you'll have plenty of time to clean up
        // CHRIS: TODO: basically that's why we always have that extra end on the challenge period!
        // CHRIS: TODO: write a big comment about this

        assertions[assertionId].successionChallenge = challengeManager.createChallenge(assertionId);
    }
}
