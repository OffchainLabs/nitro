// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./util/TestUtil.sol";
import "../../src/rollup/IRollupCreator.sol";
import "../../src/rollup/RollupAdminLogic.sol";
import "../../src/rollup/RollupUserLogic.sol";
import "../../src/rollup/ValidatorUtils.sol";
import "../../src/rollup/ValidatorWalletCreator.sol";
import "../../src/challenge/ChallengeManager.sol";
import "../../src/osp/OneStepProver0.sol";
import "../../src/osp/OneStepProverMemory.sol";
import "../../src/osp/OneStepProverMath.sol";
import "../../src/osp/OneStepProverHostIo.sol";
import "../../src/osp/OneStepProofEntry.sol";

import "@openzeppelin/contracts/access/Ownable.sol";

abstract contract AbsRollupCreatorTest is Test {
    address public rollupOwner = address(4400);
    address public deployer = address(4300);

    function _prepareRollupDeployment(address rollupCreator)
        internal
        returns (
            IOneStepProofEntry ospEntry,
            IChallengeManager challengeManager,
            IRollupAdmin rollupAdminLogic,
            IRollupUser rollupUserLogic,
            ISequencerInbox.MaxTimeVariation memory timeVars,
            address expectedRollupAddr
        )
    {
        //// deploy challenge stuff
        ospEntry = new OneStepProofEntry(
            new OneStepProver0(),
            new OneStepProverMemory(),
            new OneStepProverMath(),
            new OneStepProverHostIo()
        );
        challengeManager = new ChallengeManager();

        //// deploy rollup logic
        rollupAdminLogic = IRollupAdmin(new RollupAdminLogic());
        rollupUserLogic = IRollupUser(new RollupUserLogic());

        timeVars = ISequencerInbox.MaxTimeVariation(
            ((60 * 60 * 24) / 15),
            12,
            60 * 60 * 24,
            60 * 60
        );

        //// calculate expected address for rollup
        expectedRollupAddr = _calculateExpectedAddr(rollupCreator, vm.getNonce(rollupCreator) + 2);

        return (
            ospEntry,
            challengeManager,
            rollupAdminLogic,
            rollupUserLogic,
            timeVars,
            expectedRollupAddr
        );
    }

    function _calculateExpectedAddr(address rollupCreator, uint256 nonce)
        internal
        pure
        returns (address)
    {
        bytes1 nonceBytes1 = bytes1(uint8(nonce));
        address expectedRollupAddr = address(
            uint160(
                uint256(
                    keccak256(
                        abi.encodePacked(
                            bytes1(0xd6),
                            bytes1(0x94),
                            address(rollupCreator),
                            nonceBytes1
                        )
                    )
                )
            )
        );

        return expectedRollupAddr;
    }

    function _checkRollupIsSetUp(
        IRollupCreator rollupCreator,
        address rollupAddress,
        IRollupAdmin rollupAdmin,
        IRollupUser rollupUser
    ) internal {
        /// rollup creator
        assertEq(IOwnable(address(rollupCreator)).owner(), deployer, "Invalid rollupCreator owner");

        /// rollup proxy
        assertEq(IOwnable(rollupAddress).owner(), rollupOwner, "Invalid rollup owner");
        assertEq(_getProxyAdmin(rollupAddress), rollupOwner, "Invalid rollup's proxyAdmin owner");
        assertEq(_getPrimary(rollupAddress), address(rollupAdmin), "Invalid proxy primary impl");
        assertEq(_getSecondary(rollupAddress), address(rollupUser), "Invalid proxy secondary impl");

        /// rollup check
        RollupCore rollup = RollupCore(rollupAddress);
        assertTrue(address(rollup.sequencerInbox()) != address(0), "Invalid seqInbox");
        assertTrue(address(rollup.bridge()) != address(0), "Invalid bridge");
        assertTrue(address(rollup.inbox()) != address(0), "Invalid inbox");
        assertTrue(address(rollup.outbox()) != address(0), "Invalid outbox");
        assertTrue(address(rollup.rollupEventInbox()) != address(0), "Invalid rollupEventInbox");
        assertTrue(address(rollup.challengeManager()) != address(0), "Invalid challengeManager");
    }

    function _expectEvents(
        IRollupCreator rollupCreator,
        IBridgeCreator bridgeCreator,
        address expectedRollupAddr,
        bytes32 wasmModuleRoot,
        uint256 chainId
    ) internal {
        vm.expectEmit(true, true, true, true);
        uint256 bridgeCreatorNonce = vm.getNonce(address(bridgeCreator));
        emit RollupCreated(
            expectedRollupAddr,
            _calculateExpectedAddr(address(bridgeCreator), bridgeCreatorNonce + 2),
            _calculateExpectedAddr(address(rollupCreator), vm.getNonce(address(rollupCreator))),
            _calculateExpectedAddr(address(bridgeCreator), bridgeCreatorNonce + 1),
            _calculateExpectedAddr(address(bridgeCreator), bridgeCreatorNonce)
        );

        emit RollupInitialized(wasmModuleRoot, chainId);
    }

    function _getProxyAdmin(address proxy) internal view returns (address) {
        bytes32 adminSlot = bytes32(uint256(keccak256("eip1967.proxy.admin")) - 1);
        return address(uint160(uint256(vm.load(proxy, adminSlot))));
    }

    function _getPrimary(address proxy) internal view returns (address) {
        bytes32 primarySlot = bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1);
        return address(uint160(uint256(vm.load(proxy, primarySlot))));
    }

    function _getSecondary(address proxy) internal view returns (address) {
        bytes32 secondarySlot = bytes32(
            uint256(keccak256("eip1967.proxy.implementation.secondary")) - 1
        );
        return address(uint160(uint256(vm.load(proxy, secondarySlot))));
    }

    /****
     **** Event declarations
     ***/

    event RollupCreated(
        address indexed rollupAddress,
        address inboxAddress,
        address adminProxy,
        address sequencerInbox,
        address bridge
    );

    event RollupInitialized(bytes32 machineHash, uint256 chainId);
}
