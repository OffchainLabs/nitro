// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "../rollup/IRollupLogic.sol";
import "../rollup/IRollupCore.sol";
import "./StakingPoolErrors.sol";

/// @notice Staking pool contract for target assertion.
/// Allows users to deposit stake, create assertion once required stake amount is reached,
/// and reclaim their stake when and if the assertion is confirmed.
contract AssertionStakingPool {
    using SafeERC20 for IERC20;
    address public immutable rollup;
    bytes32 public immutable assertionHash;
    AssertionInputs public assertionInputs;
    IERC20 public immutable stakeToken;
    mapping(address => uint256) public depositedTokenBalances;

    event StakeDeposited(address indexed sender, uint256 amount);
    event StakeWithdrawn(address indexed sender, uint256 amount);

    /// @param _rollup Rollup contract of target chain
    /// @param _assertionInputs Inputs to be passed into Rollup.stakeOnNewAssertion
    /// @param _assertionHash Assertion hash to be passed into Rollup.stakeOnNewAssertion
    constructor(
        address _rollup,
        AssertionInputs memory _assertionInputs,
        bytes32 _assertionHash
    ) {
        rollup = _rollup;
        assertionHash = _assertionHash;
        assertionInputs = _assertionInputs;
        stakeToken = IERC20(IRollupCore(rollup).stakeToken());
    }

    /// @notice Deposit stake into pool contract.
    /// @param _amount amount of stake token to deposit
    function depositIntoPool(uint256 _amount) external {
        depositedTokenBalances[msg.sender] += _amount;
        stakeToken.safeTransferFrom(msg.sender, address(this), _amount);
        emit StakeDeposited(msg.sender, _amount);
    }

    /// @notice Create assertion. Callable only if required stake has been reached and assertion has not been asserted yet.
    function createAssertion() external {
        uint256 requiredStake = getRequiredStake();
        // approve spending from rollup for newStakeOnNewAssertion call
        stakeToken.safeIncreaseAllowance(rollup, requiredStake);
        // reverts if pool doesn't have enough stake and if assertion has already been asserted
        IRollupUser(rollup).newStakeOnNewAssertion(requiredStake, assertionInputs, assertionHash);
    }

    /// @notice Make stake withdrawable.
    /// @dev Separate call from withdrawStakeBackIntoPool since returnOldDeposit reverts with 0 balance (in e.g., case of admin forceRefundStaker)
    function makeStakeWithdrawable() public {
        // this checks for active staker
        IRollupUser(rollup).returnOldDeposit();
    }

    /// @notice Move stake back from rollup contract to this contract.
    /// Callable only if this contract has already created an assertion and it's now inactive.
    /// @dev Separate call from makeStakeWithdrawable since returnOldDeposit reverts with 0 balance (in e.g., case of admin forceRefundStaker)
    function withdrawStakeBackIntoPool() public {
        IRollupUser(rollup).withdrawStakerFunds();
    }

    /// @notice Combines makeStakeWithdrawable and withdrawStakeBackIntoPool into single call
    function makeStakeWithdrawableAndWithdrawBackIntoPool() external {
        makeStakeWithdrawable();
        withdrawStakeBackIntoPool();
    }

    /// @notice Send supplied amount of stake from this contract back to its depositor.
    /// @param _amount stake amount to withdraw
    function withdrawFromPool(uint256 _amount) public {
        uint256 balance = depositedTokenBalances[msg.sender];
        if (balance == 0) {
            revert NoBalanceToWithdraw(msg.sender);
        }
        if (_amount > balance) {
            revert AmountExceedsBalance(msg.sender, _amount, balance);
        }
        depositedTokenBalances[msg.sender] = balance - _amount;
        stakeToken.safeTransfer(msg.sender, _amount);
        emit StakeWithdrawn(msg.sender, _amount);
    }

    /// @notice Send full balance of stake from this contract back to its depositor.
    function withdrawFromPool() external {
        withdrawFromPool(depositedTokenBalances[msg.sender]);
    }

    /// @notice Get required stake for pool's assertion.
    /// Requried stake for a given assertion is set in the previous assertion's config data
    function getRequiredStake() public view returns (uint256) {
        return assertionInputs.beforeStateData.configData.requiredStake;
    }
}
