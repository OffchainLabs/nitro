// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "../ERC20Mock.sol";
import "../../src/assertionStakingPool/EdgeStakingPoolCreator.sol";
import "../../src/challengeV2/EdgeChallengeManager.sol";

contract MockChallengeManager {
    IERC20 public immutable stakeToken;

    event EdgeCreated(CreateEdgeArgs args);

    constructor(IERC20 _token) {
        stakeToken = _token;
    }

    function createLayerZeroEdge(CreateEdgeArgs calldata args) external returns (bytes32) {
        stakeToken.transferFrom(msg.sender, address(this), stakeAmounts(args.level));

        emit EdgeCreated(args);

        return keccak256(abi.encode(args));
    }

    function stakeAmounts(uint256 lvl) public pure returns (uint256) {
        return 100 * (lvl + 1);
    }
}

contract EdgeStakingPoolTest is Test {
    IERC20 token;
    MockChallengeManager challengeManager;
    EdgeStakingPoolCreator stakingPoolCreator;

    event EdgeCreated(CreateEdgeArgs args);

    function setUp() public {
        token = new ERC20Mock("TEST", "TST", address(this), 100 ether);
        challengeManager = new MockChallengeManager(token);
        stakingPoolCreator = new EdgeStakingPoolCreator();
    }

    function testProperInitialization(bytes32 edgeId) public {
        IEdgeStakingPool stakingPool = stakingPoolCreator.createPool(address(challengeManager), edgeId);

        assertEq(address(stakingPoolCreator.getPool(address(challengeManager), edgeId)), address(stakingPool));

        assertEq(address(stakingPool.challengeManager()), address(challengeManager));
        assertEq(stakingPool.edgeId(), edgeId);
        assertEq(address(stakingPool.stakeToken()), address(token));
    }

    function testCreateEdge(CreateEdgeArgs memory args) public {
        uint256 requiredStake = challengeManager.stakeAmounts(args.level);
        bytes32 realEdgeId = keccak256(abi.encode(args));
        IEdgeStakingPool stakingPool = stakingPoolCreator.createPool(address(challengeManager), realEdgeId);

        // simulate deposits
        // we don't need to deposit using the staking pool's deposit function because we're not testing that here
        token.transfer(address(stakingPool), requiredStake - 1);
        vm.expectRevert("ERC20: transfer amount exceeds balance");
        stakingPool.createEdge(args);
        token.transfer(address(stakingPool), 1);

        // simulate an incorrect edge id
        args.claimId = ~args.claimId;
        vm.expectRevert(abi.encodeWithSelector(IEdgeStakingPool.IncorrectEdgeId.selector, keccak256(abi.encode(args)), realEdgeId));
        stakingPool.createEdge(args);
        args.claimId = ~args.claimId;

        vm.expectEmit(false, false, false, true);
        emit EdgeCreated(args);
        stakingPool.createEdge(args);

        assertEq(token.balanceOf(address(stakingPool)), 0);
        assertEq(token.balanceOf(address(challengeManager)), requiredStake);
    }
}
