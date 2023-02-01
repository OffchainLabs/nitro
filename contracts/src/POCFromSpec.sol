// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

interface IAssertionChain {
    function getPredecessorId(bytes32 assertionId) external view returns (bytes32);
    function getHeight(bytes32 assertionId) external view returns (uint256);
    function getInboxMsgCountSeen(bytes32 assertionId) external view returns (uint256);
    function getStateHash(bytes32 assertionId) external view returns (bytes32);
    function getSuccessionChallenge(bytes32 assertionId) external view returns (bytes32);
    function isFirstChild(bytes32 assertionId) external view returns (bool);
}

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

enum Status {
    Pending,
    Confirmed,
    Rejected
}

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
    ChallengeManagers challengeManagers;
    mapping(bytes32 => Assertion) public assertions;
    uint256 public immutable stakeAmount = 100 ether; // CHRIS: TODO: update
    uint256 public immutable challengePeriod = 1000; // CHRIS: TODO: update in constructor
    IInbox inbox;

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

    function createNewAssertion(bytes32 stateHash, uint256 height, bytes32 predecessorId) external {
        // CHRIS: TODO: library on the assertion
        // CHRIS: TODO: consider if we should include the prev here? we need to right? but the reference below should be to the state hash
        bytes32 assertionId = keccak256(abi.encodePacked(stateHash, height, predecessorId));

        // assertions are always unique as they consume a deterministic number of inbox messages
        // so two different correct assertions do not exist.
        require(!assertionExists(assertionId), "Assertion already exists");

        // CHRIS: TODO: staker checks here - msg.sender has put down stake and is not staked elsewhere, then update the staker location

        require(assertionExists(predecessorId), "Previous assertion does not exist");
        require(previousAssertion(assertionId).status != Status.Rejected, "Previous assertion rejected");
        require(previousAssertion(assertionId).height < height, "Height not greater than predecessor");

        bool isFirstChild = assertions[predecessorId].firstChildCreationTime == 0;
        if (isFirstChild) {
            // if this is the first child then we update the prev
            assertions[predecessorId].firstChildCreationTime = block.timestamp;
        } else {
            require(
                block.timestamp < previousAssertion(assertionId).firstChildCreationTime + challengePeriod,
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
            isFirstChild: isFirstChild,
            firstChildCreationTime: 0,
            secondChildCreationTime: 0,
            stateHash: stateHash,
            height: height,
            status: Status.Pending,
            inboxMsgCountSeen: inbox.msgCount()
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
            bytes32 winningClaim = challengeManagers.blockChallengeManager().winningClaim(successionChallenge);
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

        require(previousAssertion(assertionId).status == Status.Confirmed, "Previous assertion not confirmed");

        // CHRIS: TODO: add a test for this:
        // bad pattern here - create a test case for it, shouldnt be possible now
        // 1. create child
        // 2. confirm child by waiting for timeout
        // 3. create second child
        // 4. create challenge

        // CHRIS: TODO: this pattern and above in reject isnt nice
        if (
            previousAssertion(assertionId).secondChildCreationTime == 0
                && block.timestamp > previousAssertion(assertionId).firstChildCreationTime + challengePeriod
        ) {
            assertions[assertionId].status = Status.Confirmed;
        } else {
            bytes32 successionChallenge = previousAssertion(assertionId).successionChallenge;
            if (successionChallenge == 0) {
                revert NotConfirmable(assertionId);
            }

            // CHRIS: TODO: external call, careful!
            bytes32 winner = challengeManagers.blockChallengeManager().winningClaim(successionChallenge);
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

        require(assertions[assertionId].secondChildCreationTime != 0, "At least two children not created");

        // CHRIS: TODO: I think this should be secondChildTime + 1 challenge period, and in the endTime of BlockChallenge below
        // CHRIS: TODO: do we have this requirement in the new paper?
        require(
            block.timestamp < assertions[assertionId].firstChildCreationTime + (2 * challengePeriod),
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

        assertions[assertionId].successionChallenge =
            challengeManagers.blockChallengeManager().createChallenge(assertionId);
    }
}

struct ChallengeVertex {
    bytes32 predecessorId;
    bytes32 successionChallenge;
    bytes32 historyCommitment;
    uint256 height; // CHRIS: TODO: are heights zero indexed or from 1?
    bytes32 claimId; // CHRIS: TODO: aka tag; only on a leaf
    address staker; // CHRIS: TODO: only on a leaf
    // CHRIS: TODO: use a different status for the vertices since they never transition to rejected?
    Status status;
    // the presumptive successor to this vertex
    bytes32 presumptiveSuccessorId;
    // CHRIS: TODO: we should have a staker in here to decide what do in the event of a win/loss?
    // the last time the presumptive successor to this vertex changed
    uint256 presumptiveSuccessorLastUpdated;
    // the amount of time this vertex has spent as the presumptive successor
    /// @notice DO NOT USE TO GET PS TIME! Instead use a getter function which takes into account unflushed ps time as well.
    ///         This is the amount of time that this vertex is recorded to have been the presumptive successor
    ///         However this may not be the total amount of time being the presumptive successor, as this vertex may currently
    ///         be the ps, and so may have some time currently being record on the predecessor.
    uint256 flushedPsTime;
    // the id of the successor with the lowest height. Zero if this vertex has no successors.
    bytes32 lowestHeightSucessorId;
}
// CHRIS: TODO:
// 1. what do we do at the end if no-one has the necessary stuff? we just dont have an end time
// 2. also, what about the exclusion stuff? is that necessary any more?

// 3. merge timer changed,
// ps logic changed,
// confirmation logic changed - we just leave the challenge open - it doesnt have an end time
// how do confirmation times improve here? - I dont think they do
// The start point is now recorded (we already had this).
// The end point is always the same height in the new protocol.
// 4.

library ChallengeVertexLib {
    function rootId() internal pure returns (bytes32) {
        return id(0, 0);
    }

    function newRoot() internal pure returns (ChallengeVertex memory) {
        // CHRIS: TODO: the root should have a height 1 and should inherit the state commitment from above right?
        return ChallengeVertex({
            predecessorId: 0,
            successionChallenge: 0,
            historyCommitment: 0, // CHRIS: TODO: this isnt correct - we should compute this from the claim apparently
            height: 0, // CHRIS: TODO: this should be 1 from the spec/paper - DIFF to paper - also in the id
            claimId: 0, // CHRIS: TODO: should this be a reference to the assertion on which this challenge is based? 2-way link?
            status: Status.Confirmed,
            staker: address(0),
            presumptiveSuccessorId: 0, // we dont know who the presumptive successor was
            presumptiveSuccessorLastUpdated: 0, // CHRIS: TODO: maybe we wanna update this?
            // but when adding a new leaf if the presumptive successor is still 0, then we say that the
            // CHRIS: TODO: although this migh violate an invariant about lowest height
            flushedPsTime: 0, // always zero for the root
            lowestHeightSucessorId: 0
        });
    }

    function id(bytes32 historyCommitment, uint256 height) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(historyCommitment, height));
    }

    // CHRIS: TODO: duplication for storage/mem - we also dont need `has` AND vertexExists
    function exists(ChallengeVertex storage vertex) internal view returns (bool) {
        return vertex.historyCommitment != 0;
    }

    function existsMem(ChallengeVertex memory vertex) internal pure returns (bool) {
        return vertex.historyCommitment != 0;
    }

    function isLeaf(ChallengeVertex storage vertex) internal view returns (bool) {
        return exists(vertex) && vertex.staker != address(0);
    }

    function isLeafMem(ChallengeVertex memory vertex) internal pure returns (bool) {
        return existsMem(vertex) && vertex.staker != address(0);
    }
}

library ChallengeVertexMappingLib {
    using ChallengeVertexLib for ChallengeVertex;

    function has(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) public view returns (bool) {
        // CHRIS: TODO: this doesnt work for root atm
        return vertices[vId].historyCommitment != 0;
    }

    function hasConfirmablePsAt(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) public view returns (bool) {
        require(has(vertices, vId), "Predecessor vertex does not exist");

        // CHRIS: TODO: rework this to question if we are confirmable
        return getCurrentPsTimer(vertices, vertices[vId].presumptiveSuccessorId) > challengePeriod;
    }

    function getCurrentPsTimer(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        // CHRIS: TODO: is it necessary to check exists everywhere? shoudlnt we just do that in the base? ideally we'd do it here, but it's expensive
        require(has(vertices, vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(has(vertices, predecessorId), "Predecessor vertex does not exist");

        bytes32 presumptiveSuccessorId = vertices[predecessorId].presumptiveSuccessorId;
        uint256 flushedPsTimer = vertices[vId].flushedPsTime;
        if (presumptiveSuccessorId == vId) {
            return (block.timestamp - vertices[predecessorId].presumptiveSuccessorLastUpdated) + flushedPsTimer;
        } else {
            return flushedPsTimer;
        }
    }

    struct NewChallengeVertex {
        bytes32 historyCommitment;
        uint256 height;
        bytes32 claimId;
        address staker;
        uint256 initialPsTime;
    }

    // // a. bisect
    // // 1. is there a current presumptive successor - if so then it is the only vertex at this height. Update it and set the ps to 0. Set our own timer to be inherited from above
    // // 2. there is no current ps - if there exists a sibling to us then we do nothing with ps. If there does not we set ourselves as the PS. We do this by setting.
    // // 3. ok, include a lowest successor height. And if we are less than this then set ourselves to the ps, if we are = then update the pstimer of the ps if ps id is non zero, otherwise do nothing
    // // b. merge
    // // 1. Set the ps timer to be the max of the two - if we're merging we change nothing on the ps of the former
    // // 2. for the second child a rival must exist right? Yes - so we update the latter if it has a ps id

    function addNewSuccessor(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 predecessorId,
        NewChallengeVertex memory successor,
        uint256 challengePeriod
    ) public {
        bytes32 vId = ChallengeVertexLib.id(successor.historyCommitment, successor.height);
        require(!has(vertices, vId), "Successor already exists exist");

        vertices[vId] = ChallengeVertex({
            predecessorId: ChallengeVertexLib.rootId(),
            successionChallenge: 0,
            historyCommitment: successor.historyCommitment,
            height: successor.height,
            claimId: successor.claimId,
            staker: successor.staker,
            status: Status.Pending,
            presumptiveSuccessorId: 0,
            presumptiveSuccessorLastUpdated: 0,
            flushedPsTime: successor.initialPsTime,
            lowestHeightSucessorId: 0
        });

        updateSuccessor(vertices, predecessorId, vId, challengePeriod);
    }

    // CHRIS: TODO: is it always safe to call this?
    function updateSuccessor(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 newPredecessorId,
        bytes32 vId,
        uint256 challengePeriod
    ) public {
        // CHRIS: TODO: this is a beast of a function - we should break it down

        // dont add a node if the challenge has declared a winner
        // CHRIS: TODO: require winning claim == 0

        require(has(vertices, newPredecessorId), "Predecessor vertex does not exist");
        require(has(vertices, vId), "Successor already exists exist");

        vertices[vId].predecessorId = newPredecessorId;

        uint256 height = vertices[vId].height;
        // is there a current lowest height - this is always the case unless we're adding a new node
        if (vertices[newPredecessorId].lowestHeightSucessorId == 0) {
            // no lowest height successor, means no successors at all, so we can set this vertex as the presumptive successor
            vertices[newPredecessorId].presumptiveSuccessorLastUpdated = block.timestamp;

            vertices[newPredecessorId].presumptiveSuccessorId = vId;
            vertices[newPredecessorId].lowestHeightSucessorId = vId;
        } else if (vertices[newPredecessorId].presumptiveSuccessorId == 0) {
            if (height < vertices[vertices[newPredecessorId].lowestHeightSucessorId].height) {
                // if we are lower than the lowest height then we set ourselves
                vertices[newPredecessorId].presumptiveSuccessorLastUpdated = block.timestamp;

                vertices[newPredecessorId].presumptiveSuccessorId = vId;
                vertices[newPredecessorId].lowestHeightSucessorId = vId;
            } else {
                // if we are at the same height or above, then there's nothing to set
            }
        } else {
            // there is a lowest height, but there is not a ps
            // this means there are multiple vertices at the same lowest height, so none are the ps

            // never set the ps if one is already confirmable
            require(
                !hasConfirmablePsAt(vertices, newPredecessorId, challengePeriod),
                "Presumptive successor already confirmable"
            );

            if (height < vertices[vertices[newPredecessorId].lowestHeightSucessorId].height) {
                // if we are lower than the lowest height, then flush the old ps and set ourselves
                uint256 timeToAdd = block.timestamp - vertices[newPredecessorId].presumptiveSuccessorLastUpdated;
                vertices[vertices[newPredecessorId].presumptiveSuccessorId].flushedPsTime += timeToAdd;
                vertices[newPredecessorId].presumptiveSuccessorLastUpdated = block.timestamp;

                vertices[newPredecessorId].presumptiveSuccessorId = vId;
                vertices[newPredecessorId].lowestHeightSucessorId = vId;
                // CHRIS: TODO: this doesnt take into account if we are the ps that we're updating - yes it does now since we check lowest height successor = vId
            } else if (height == vertices[vertices[newPredecessorId].lowestHeightSucessorId].height) {
                // if we are at the same height as the ps, then flush the ps and 0 the ps
                uint256 timeToAdd = block.timestamp - vertices[newPredecessorId].presumptiveSuccessorLastUpdated;
                vertices[vertices[newPredecessorId].presumptiveSuccessorId].flushedPsTime += timeToAdd;
                vertices[newPredecessorId].presumptiveSuccessorLastUpdated = block.timestamp;

                if (vertices[newPredecessorId].lowestHeightSucessorId == vId) {
                    // in this case presumptive successor is equal to the lowest height successor // CHRIS: TODO: add asserts for things like that
                    // so we just need to update the flush
                } else {
                    // CHRIS: TODO: no need to update twice here
                    vertices[newPredecessorId].presumptiveSuccessorLastUpdated = 0;
                    vertices[newPredecessorId].presumptiveSuccessorId = 0;
                }
            } else {
                // otherwise we are higher than the lowest height, so nothing to set
            }
        }
    }

    function bisectionHeight(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId)
        internal
        view
        returns (uint256)
    {
        require(has(vertices, vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(has(vertices, predecessorId), "Predecessor vertex does not exist");

        require(vertices[vId].height - vertices[predecessorId].height >= 2, "Height different not two or more");
        // CHRIS: TODO: look at the boundary conditions here
        // CHRIS: TODO: update this to use the correct formula from the paper
        return 10; // placeholder
    }
}

library HistoryCommitmentLib {
    function hasState(bytes32 historyCommitment, bytes32 state, uint256 stateHeight, bytes memory proof)
        internal
        returns (bool)
    {
        // CHRIS: TODO: do a merkle proof check
        return true;
    }

    function hasPrefix(
        bytes32 historyCommitment,
        bytes32 prefixHistoryCommitment,
        uint256 prefixHistoryHeight,
        bytes memory proof
    ) internal returns (bool) {
        // CHRIS: TODO:
        // prove that the sequence of states commited to by prefixHistoryCommitment is a prefix
        // of the sequence of state commited to by the historyCommitment
        return true;
    }
}

library ChallengeLib {
    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);

    function confirmationPreChecks(mapping(bytes32 => ChallengeVertex) storage vertices, bytes32 vId) internal view {
        // basic checks
        require(vertices.has(vId), "Vertex does not exist");
        require(vertices[vId].status == Status.Pending, "Vertex is not pending");
        bytes32 predecessorId = vertices[vId].predecessorId;
        require(vertices.has(predecessorId), "Predecessor vertex does not exist");

        // for a vertex to be confirmed its predecessor must be confirmed
        // this ensures an unbroken chain of confirmation from the root eventually up to one the leaves
        require(vertices[predecessorId].status == Status.Confirmed, "Predecessor not confirmed");
    }

    /// @notice Checks if the vertex is eligible to be confirmed because it has a high enought ps timer
    /// @param vertices The tree of vertices
    /// @param vId The vertex to be confirmed
    /// @param challengePeriod One challenge period in seconds
    function checkConfirmForPsTimer(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        uint256 challengePeriod
    ) internal view {
        confirmationPreChecks(vertices, vId);

        // ensure only one type of confirmation is valid on this node and all it's siblings
        require(vertices[vertices[vId].predecessorId].successionChallenge == 0, "Succession challenge already opened");

        // now ensure that only one of the siblings is valid for this time of confirmation
        // here we ensure that because only one vertex can ever have a ps timer greater than the challenge period, before the end time
        require(vertices.getCurrentPsTimer(vId) > challengePeriod, "PsTimer not greater than challenge period");
    }

    /// @notice Checks if the vertex is eligible to be confirmed because it has been declared a winner in a succession challenge
    function checkConfirmForSucessionChallengeWin(
        mapping(bytes32 => ChallengeVertex) storage vertices,
        bytes32 vId,
        ChallengeManager challengeManager
    ) internal view {
        confirmationPreChecks(vertices, vId);

        // ensure only one type of confirmation is valid on this node and all it's siblings
        bytes32 successionChallenge = vertices[vertices[vId].predecessorId].successionChallenge;
        require(successionChallenge != 0, "Succession challenge does not exist");

        // now ensure that only one of the siblings is valid for this time of confirmation
        // here we ensure that since a succession challenge only declares one winner
        require(
            // CHRIS: TODO: handle this "sub" challenge manager thing differently
            challengeManager.subWinningClaim(successionChallenge) == vId,
            "Succession challenge did not declare this vertex the winner"
        );
    }
}

contract ChallengeManagers {
    BlockChallengeManager public blockChallengeManager;
    BigStepChallengeManager public bigStepChallengeManager;
    SmallStepChallengeManager public smallStepChallengeManager;
    IAssertionChain public assertionChain;
    OneStepProofManager public oneStepProofManager;
}

struct Challenge {
    bytes32 winningClaim;
    uint256 startTime; // CHRIS: TODO: why do we need this?
}

abstract contract ChallengeManager {
    // CHRIS: TODO: do this in a different way
    ChallengeManagers challengeManagers;

    using ChallengeVertexMappingLib for mapping(bytes32 => ChallengeVertex);
    using ChallengeVertexLib for ChallengeVertex;

    mapping(bytes32 => mapping(bytes32 => ChallengeVertex)) public vertices;
    mapping(bytes32 => Challenge) public challenges;

    uint256 public immutable startTime = block.timestamp;
    uint256 immutable miniStake = 1 ether; // CHRIS: TODO: fill with value
    uint256 immutable challengePeriod = 10; // CHRIS: TODO: how to set this, and compare to end time?

    // CHRIS: TODO: better name for that start
    // CHRIS: TODO: any access management here? we shouldnt allow the challenge to be created by anyone as this affects the start timer - so we should has the id with teh creating address?
    function createChallenge(bytes32 startId) public returns (bytes32) {
        challenges[startId] = Challenge({winningClaim: 0, startTime: block.timestamp});
        // CHRIS: TODO: pass the startId into the newroot and also use it as the root id
        vertices[startId][ChallengeVertexLib.rootId()] = ChallengeVertexLib.newRoot();
        return startId;
    }

    function winningClaim(bytes32 challengeId) public view returns (bytes32) {
        // CHRIS: TODO: check exists? or return the full struct?
        return challenges[challengeId].winningClaim;
    }

    function vertexExists(bytes32 challengeId, bytes32 vId) public view returns (bool) {
        return vertices[challengeId].has(vId);
    }

    function getVertex(bytes32 challengeId, bytes32 vId) public view returns (ChallengeVertex memory) {
        require(vertices[challengeId][vId].exists(), "Vertex does not exist");

        return vertices[challengeId][vId];
    }

    function initialPSTime(bytes32 challengeId, bytes32 claimId) internal virtual returns (uint256);
    function instantiateSubChallenge(bytes32 predecessorId) internal virtual returns (bytes32);

    function getCurrentPsTimer(bytes32 challengeId, bytes32 vId) public view returns (uint256) {
        return vertices[challengeId].getCurrentPsTimer(vId);
    }

    function subWinningClaim(bytes32 challengeId) public view virtual returns (bytes32);

    // CHRIS: TODO: re-arrange the order of args on all these functions - we should use something consistent
    function addLeafImpl(
        bytes32 challengeId,
        bytes32 claimId,
        uint256 height,
        bytes32 historyCommitment,
        bytes32 lastState,
        bytes memory lastStatehistoryProof
    ) internal {
        require(claimId != 0, "Empty claimId");
        require(historyCommitment != 0, "Empty historyCommitment");
        // CHRIS: TODO: we should also prove that the height is greater than 1 if we set the root heigt to 1
        require(height != 0, "Empty height");

        // could we have lib functions that operate on the same contract?
        //

        // CHRIS: TODO: comment on why we need the mini stake
        // CHRIS: TODO: also are we using this to refund moves in real-time? would be more expensive if so, but could be necessary?
        // CHRIS: TODO: this can apparently be moved directly to the public goods fund
        // CHRIS: TODO: we need to record who was on the winning leaf
        require(msg.value == miniStake, "Incorrect mini-stake amount");

        // CHRIS: TODO: require that this challenge hasnt declared a winner
        require(winningClaim(challengeId) == 0, "Winner already declared");

        require(
            HistoryCommitmentLib.hasState(historyCommitment, lastState, height, lastStatehistoryProof),
            "Last state not in history"
        );
        // CHRIS: TODO: also check the root is in the history at height 0/1?

        vertices[challengeId].addNewSuccessor(
            ChallengeVertexLib.rootId(),
            // CHRIS: TODO: move this struct out
            ChallengeVertexMappingLib.NewChallengeVertex({
                historyCommitment: historyCommitment,
                height: height,
                claimId: claimId,
                staker: msg.sender,
                // CHRIS: TODO: the naming is bad here
                initialPsTime: initialPSTime(challengeId, claimId)
            }),
            challengePeriod
        );
    }

    /// @dev Confirms the vertex without doing any checks. Also sets the winning claim if the vertex
    ///      is a leaf.
    function setConfirmed(bytes32 challengeId, bytes32 vId) internal {
        vertices[challengeId][vId].status = Status.Confirmed;
        if (vertices[challengeId][vId].isLeaf()) {
            challenges[challengeId].winningClaim = vertices[challengeId][vId].claimId;
        }
    }

    /// @notice Confirm a vertex because it has been the presumptive successor for long enough
    /// @param challengeId The challenge to confirm the vertex in
    /// @param vId The vertex id
    function confirmForPsTimer(bytes32 challengeId, bytes32 vId) public {
        ChallengeLib.checkConfirmForPsTimer(vertices[challengeId], vId, challengePeriod);
        setConfirmed(challengeId, vId);
    }

    /// Confirm a vertex because it has won a succession challenge
    /// @param challengeId The challenge to confirm the vertex in
    /// @param vId The vertex id
    function confirmForSucessionChallengeWin(bytes32 challengeId, bytes32 vId) public {
        ChallengeLib.checkConfirmForSucessionChallengeWin(vertices[challengeId], vId, this);
        setConfirmed(challengeId, vId);
    }

    function createSubChallenge(bytes32 challengeId, bytes32 child1Id, bytes32 child2Id) public virtual {
        require(child1Id != child2Id, "Children are not different");
        require(vertices[challengeId].has(child1Id), "Child 1 does not exist");
        require(vertices[challengeId].has(child2Id), "Child 2 does not exist");
        bytes32 predecessorId = vertices[challengeId][child1Id].predecessorId;
        require(
            predecessorId == vertices[challengeId][child2Id].predecessorId, "Children do not have the same predecessor"
        );

        uint256 predecessorHeight = vertices[challengeId][predecessorId].height;
        require(
            vertices[challengeId][child1Id].height - predecessorHeight == 1,
            "Child 1 is not one step from it's predecessor"
        );
        require(
            vertices[challengeId][child2Id].height - predecessorHeight == 1,
            "Child 2 is not one step from it's predecessor"
        );

        require(winningClaim(challengeId) == 0, "Winner already declared");

        require(
            !vertices[challengeId].hasConfirmablePsAt(predecessorId, challengePeriod),
            "Presumptive successor confirmable"
        );

        require(vertices[challengeId][predecessorId].successionChallenge == 0, "Challenge already exists");

        bytes32 subChallenge = instantiateSubChallenge(vertices[challengeId][child1Id].predecessorId);
        vertices[challengeId][predecessorId].successionChallenge = subChallenge;

        // CHRIS: TODO: opening a challenge and confirming a winner vertex should have mutually exlusive checks
    }

    function getBisectionVertex(
        bytes32 challengeId,
        bytes32 vId,
        bytes32 prefixHistoryCommitment,
        bytes memory prefixProof
    ) internal returns (uint256 height) {
        require(winningClaim(challengeId) == 0, "Winner already declared");

        require(vertices[challengeId].has(vId), "Vertex does not exist");
        bytes32 predecessorId = vertices[challengeId][vId].predecessorId;
        require(vertices[challengeId].has(predecessorId), "Predecessor vertex does not exist");
        require(
            vertices[challengeId][predecessorId].presumptiveSuccessorId != vId, "Cannot bisect presumptive successor"
        );
        require(
            !vertices[challengeId].hasConfirmablePsAt(predecessorId, challengePeriod),
            "Presumptive successor already confirmable"
        );

        uint256 bHeight = vertices[challengeId].bisectionHeight(vId);

        require(
            HistoryCommitmentLib.hasPrefix(
                vertices[challengeId][vId].historyCommitment, prefixHistoryCommitment, bHeight, prefixProof
            ),
            "Invalid prefix history"
        );

        return bHeight;
    }

    function bisect(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        public
        virtual
    {
        uint256 bHeight = getBisectionVertex(challengeId, vId, prefixHistoryCommitment, prefixProof);
        bytes32 bVId = ChallengeVertexLib.id(prefixHistoryCommitment, bHeight);

        // CHRIS: TODO: could use this=>
        // mapping(bytes32 => ChallengeVertex) storage v = vertices[challengeId];
        // CHRIS: redundant check?
        require(!vertices[challengeId].has(bVId), "Bisection vertex already exists");

        // CHRIS: TODO: the spec says we should stop the presumptive successor timer of the vId, but why?
        // CHRIS: TODO: is that because we only care about presumptive successors further down the chain?

        bytes32 predecessorId = vertices[challengeId][vId].predecessorId;
        vertices[challengeId].addNewSuccessor(
            predecessorId,
            ChallengeVertexMappingLib.NewChallengeVertex({
                historyCommitment: prefixHistoryCommitment,
                height: bHeight,
                claimId: 0,
                staker: address(0),
                // CHRIS: TODO: double check the timer updates in here and merge - they're a bit tricky to reason about
                initialPsTime: vertices[challengeId].getCurrentPsTimer(vId)
            }),
            challengePeriod
        );
        // CHRIS: TODO: check these two successor updates really do conform to the spec
        vertices[challengeId].updateSuccessor(bVId, vId, challengePeriod);
    }

    function merge(bytes32 challengeId, bytes32 vId, bytes32 prefixHistoryCommitment, bytes memory prefixProof)
        public
        virtual
    {
        uint256 bHeight = getBisectionVertex(challengeId, vId, prefixHistoryCommitment, prefixProof);
        bytes32 bVId = ChallengeVertexLib.id(prefixHistoryCommitment, bHeight);
        // CHRIS: redundant check?
        require(vertices[challengeId].has(bVId), "Bisection vertex does not already exist");
        // CHRIS: TODO: include a long comment about this
        require(!vertices[challengeId][bVId].isLeaf(), "Cannot merge to a leaf");

        vertices[challengeId].updateSuccessor(bVId, vId, challengePeriod);
        // update the merged vertex so that it has an up to date timer
        vertices[challengeId].updateSuccessor(vertices[challengeId][bVId].predecessorId, bVId, challengePeriod);
        // update the merge vertex if we have a higher ps time
        if (vertices[challengeId][bVId].flushedPsTime < vertices[challengeId][vId].flushedPsTime) {
            vertices[challengeId][bVId].flushedPsTime = vertices[challengeId][vId].flushedPsTime;
        }
    }
}

contract BlockChallengeManager is ChallengeManager {
    constructor() ChallengeManager() {}

    function getBlockHash(bytes32 assertionStateHash, bytes memory proof) internal returns (bytes32) {
        // CHRIS: TODO:
        // 1. The assertion state hash contains all the info being asserted - including the block hash
        // 2. Extract the block hash from the assertion state hash using the claim proof and return it
    }

    function getInboxMsgProcessedCount(bytes32 assertionStateHash, bytes memory proof) internal returns (uint256) {
        // CHRIS: TODO:
        // 1. Unwrap the assertion state hash to find the number of inbox messages it processed
    }

    function addLeaf(
        bytes32 challengeId,
        bytes32 claimId,
        uint256 height,
        bytes32 historyCommitment,
        bytes32 lastState,
        bytes memory lastStatehistoryProof,
        bytes memory blockHashProof,
        bytes memory inboxMsgProcessedProof
    ) public {
        // check that the predecessor of this claim has registered this contract as it's succession challenge
        bytes32 predecessorId = challengeManagers.assertionChain().getPredecessorId(claimId);
        require(
            challengeManagers.assertionChain().getSuccessionChallenge(predecessorId) == challengeId,
            "Claim predecessor not linked to this challenge"
        );

        uint256 assertionHeight = challengeManagers.assertionChain().getHeight(claimId);
        uint256 predecessorAssertionHeight = challengeManagers.assertionChain().getHeight(predecessorId);

        uint256 leafHeight = assertionHeight - predecessorAssertionHeight;
        require(leafHeight == height, "Invalid height");

        bytes32 claimStateHash = challengeManagers.assertionChain().getStateHash(claimId);
        require(
            getInboxMsgProcessedCount(claimStateHash, inboxMsgProcessedProof)
                == challengeManagers.assertionChain().getInboxMsgCountSeen(predecessorId),
            "Invalid inbox messages processed"
        );

        require(
            getBlockHash(claimStateHash, blockHashProof) == lastState,
            "Last state is not the assertion claim block hash"
        );

        addLeafImpl(challengeId, claimId, height, historyCommitment, lastState, lastStatehistoryProof);
    }

    // CHRIS: TODO: rethink this - it isnt so nice
    function subWinningClaim(bytes32 challengeId) public view override returns (bytes32) {
        return challengeManagers.bigStepChallengeManager().winningClaim(challengeId);
    }

    function initialPSTime(bytes32 challengeId, bytes32 claimId) internal view override returns (uint256) {
        bool isFirstChild = challengeManagers.assertionChain().isFirstChild(claimId);
        if (isFirstChild) {
            // CHRIS: TODO: look into this more, it seems not right to use start time - we should use assertion creation times
            return block.timestamp - startTime;
        } else {
            return 0;
        }
    }

    function instantiateSubChallenge(bytes32 predecessorId) internal override returns (bytes32) {
        return challengeManagers.smallStepChallengeManager().createChallenge(predecessorId);
    }
}

contract BigStepChallengeManager is ChallengeManager {
    constructor() ChallengeManager() {}

    // CHRIS: TODO: should we also check that the root is the first in the merkle history? we do that for the ends, why not for the start, would be nice for that invariant to hold

    function getBlockHashFromClaim(bytes32 claimId, bytes memory claimProof) internal returns (bytes32) {
        // CHRIS: TODO:
        // 1. Get the history commitment for this claim
        // 2. Unwrap the last state of the claim using the proof
        // 3. Also use the proof to extract the block hash from the last state
        // 4. Return the block hash
    }

    function getBlockHashProducedByTerminalState(bytes32 state, bytes memory stateProof) internal returns (bytes32) {
        // 1. Hydrate the state using the state proof
        // 2. Show that the state is terminal
        // 3. Extract the block hash that is being produced by this terminal state
    }

    function addLeaf(
        bytes32 challengeId,
        bytes32 claimId,
        uint256 height,
        bytes32 historyCommitment,
        bytes32 lastState,
        bytes memory lastStatehistoryProof,
        bytes memory claimBlockHashProof,
        bytes memory stateBlockHashProof
    ) public {
        // CHRIS: TODO: rename challenge to challenge manager
        require(
            challengeManagers.blockChallengeManager().vertexExists(challengeId, claimId),
            "Claim does not exist in parent"
        );
        ChallengeVertex memory claimVertex = challengeManagers.blockChallengeManager().getVertex(challengeId, claimId);
        require(
            challengeManagers.blockChallengeManager().vertexExists(challengeId, claimVertex.predecessorId),
            "Claim predecessor does not exist in parent"
        );
        ChallengeVertex memory claimPrevVertex =
            challengeManagers.blockChallengeManager().getVertex(challengeId, claimVertex.predecessorId);

        require(claimVertex.height - claimPrevVertex.height == 1, "Claim is not one step above it's predecessor");

        require(claimPrevVertex.successionChallenge == challengeId, "Claim predecessor challenge is not this challenge");

        // CHRIS: TODO: also check that the claim is a block hash?

        // in a bigstep challenge the states are wasm states, and the claims are block challenge vertices
        // check that the wasm state is a terminal state, and that it produces the blockhash that's in the claim
        bytes32 lastStateBlockHash = getBlockHashProducedByTerminalState(lastState, stateBlockHashProof);
        bytes32 claimBlockHash = getBlockHashFromClaim(claimId, claimBlockHashProof);

        require(claimBlockHash == lastStateBlockHash, "Claim inconsistent with state");

        addLeafImpl(challengeId, claimId, height, historyCommitment, lastState, lastStatehistoryProof);
    }

    // CHRIS: TODO: rethink this - it isnt so nice
    function subWinningClaim(bytes32 challengeId) public view override returns (bytes32) {
        return challengeManagers.smallStepChallengeManager().winningClaim(challengeId);
    }

    function initialPSTime(bytes32 challengeId, bytes32 claimId) internal view override returns (uint256) {
        return challengeManagers.blockChallengeManager().getCurrentPsTimer(challengeId, claimId);
    }

    // CHRIS: TODO: better naming on this and createchallenge
    function instantiateSubChallenge(bytes32 predecessorId) internal override returns (bytes32) {
        return challengeManagers.smallStepChallengeManager().createChallenge(predecessorId);
    }
}

contract SmallStepChallengeManager is ChallengeManager {
    uint256 public constant MAX_STEPS = 2 << 19;

    constructor() ChallengeManager() {}

    function getProgramCounter(bytes32 state, bytes memory proof) public returns (uint256) {
        // CHRIS: TODO:
        // 1. hydrate the wavm state with the proof
        // 2. find the program counter and return it
    }

    function addLeaf(
        bytes32 challengeId,
        bytes32 claimId,
        uint256 height,
        bytes32 historyCommitment,
        bytes32 lastState,
        bytes memory lastStatehistoryProof,
        bytes memory claimHistoryProof,
        bytes memory programCounterProof
    ) public {
        require(
            challengeManagers.bigStepChallengeManager().vertexExists(challengeId, claimId),
            "Claim does not exist in parent"
        );
        ChallengeVertex memory claimVertex = challengeManagers.bigStepChallengeManager().getVertex(challengeId, claimId);
        require(
            challengeManagers.bigStepChallengeManager().vertexExists(challengeId, claimVertex.predecessorId),
            "Claim predecessor does not exist in parent"
        );
        ChallengeVertex memory claimPrevVertex =
            challengeManagers.bigStepChallengeManager().getVertex(challengeId, claimVertex.predecessorId);

        require(claimVertex.height - claimPrevVertex.height == 1, "Claim is not one step above it's predecessor");

        require(claimPrevVertex.successionChallenge == challengeId, "Claim predecessor challenge is not this challenge");

        // the wavm state of the last state should always be exactly the same as the wavm state of the claim
        // regardless of the height
        require(
            HistoryCommitmentLib.hasState(claimVertex.historyCommitment, lastState, 1, claimHistoryProof),
            "Invalid claim state"
        );

        uint256 lastStateProgramCounter = getProgramCounter(lastState, programCounterProof);
        uint256 predecessorSteps = claimPrevVertex.height * MAX_STEPS;

        require(predecessorSteps + height == lastStateProgramCounter, "Inconsistent program counter");

        if (ChallengeVertexLib.isLeafMem(claimVertex)) {
            require(height == MAX_STEPS, "Invalid non-leaf steps");
        } else {
            require(height <= MAX_STEPS, "Invalid leaf steps");
        }

        addLeafImpl(challengeId, claimId, height, historyCommitment, lastState, lastStatehistoryProof);
    }

    // CHRIS: TODO: rethink this - it isnt so nice
    function subWinningClaim(bytes32 challengeId) public view override returns (bytes32) {
        return challengeManagers.oneStepProofManager().winningClaim(challengeId);
    }

    function initialPSTime(bytes32 challengeId, bytes32 claimId) internal view override returns (uint256) {
        return challengeManagers.bigStepChallengeManager().getCurrentPsTimer(challengeId, claimId);
    }

    function instantiateSubChallenge(bytes32 predecessorId) internal override returns (bytes32) {
        return challengeManagers.oneStepProofManager().createOneStepProof(predecessorId);
    }
}

// CHRIS: TODO: one step proof test just here for structure test
contract OneStepProofManager {
    mapping(bytes32 => Challenge) public challenges;

    function winningClaim(bytes32 challengeId) public view returns (bytes32) {
        return challenges[challengeId].winningClaim;
    }

    function createOneStepProof(bytes32 startState) public returns (bytes32) {
        revert("NOT_IMPLEMENTED");
    }

    function setWinningClaim(bytes32 startState, bytes32 _winner) public {
        challenges[startState] = Challenge({winningClaim: _winner, startTime: 0});
    }
}
