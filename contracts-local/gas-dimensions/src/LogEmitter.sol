// Copyright 2025, Offchain Labs, Inc.
// For license information, see:
// https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.13;

contract LogEmitter {
    uint256 public number;

    event LogOneTopic();
    event LogOneTopicExtraData(bytes);
    event LogTwoTopics(uint256 indexed number);
    event LogTwoTopicsExtraData(uint256 indexed number, address);
    event LogThreeTopics(uint256 indexed number, address indexed addy);
    event LogThreeTopicsExtraData(uint256 indexed number, address indexed addy, bytes32);
    event LogFourTopics(uint256 indexed number, address indexed addy, bytes32 indexed keccakHash);
    event LogFourTopicsExtraData(uint256 indexed number, address indexed addy, bytes32 indexed keccakHash, bytes32);

    function logZeroTopics(bytes memory data) internal {
        assembly {
            log0(add(data, 0x20), mload(data))
        }
    }

    function emitZeroTopicEmptyData() public {
        logZeroTopics("");
        number++;
    }

    function emitZeroTopicNonEmptyData() public {
        logZeroTopics(abi.encodePacked("abcdefg"));
        number++;
    }

    function emitZeroTopicNonEmptyDataAndMemExpansion() public {
        assembly {
            let memPtr := msize()
            log0(memPtr, 0x40)
        }
        number++;
    }

    function emitOneTopicEmptyData() public {
        emit LogOneTopic();
        number++;
    }

    function emitOneTopicNonEmptyData() public {
        bytes memory data = abi.encodePacked("hijklmnop");
        bytes32 topic = bytes32(uint256(0x1337));
        assembly {
            log1(add(data, 0x20), mload(data), topic)
        }
        number++;
    }

    function emitOneTopicNonEmptyDataAndMemExpansion() public {
        bytes32 topic = bytes32(uint256(0x1337));
        assembly {
            let memPtr := msize()
            log1(memPtr, 0x40, topic)
        }
        number++;
    }

    function emitTwoTopics() public {
        emit LogTwoTopics(0xd00d00);
        number++;
    }

    function emitTwoTopicsExtraData() public {
        emit LogTwoTopicsExtraData(0xcaca, address(0xdeadcafe));
        number++;
    }

    function emitTwoTopicsExtraDataAndMemExpansion() public {
        bytes32 topic = bytes32(uint256(0x1337));
        bytes32 topic2 = bytes32(uint256(0x1338));
        assembly {
            let memPtr := msize()
            log2(memPtr, 0x40, topic, topic2)
        }
        number++;
    }

    function emitThreeTopics() public {
        emit LogThreeTopics(0xb1337b, address(0xbabecafe));
        number++;
    }

    function emitThreeTopicsExtraData() public {
        emit LogThreeTopicsExtraData(0xb00bb00c, address(0xfeedface), bytes32(abi.encodePacked("HIJKLMNOP")));
        number++;
    }

    function emitThreeTopicsExtraDataAndMemExpansion() public {
        bytes32 topic = bytes32(uint256(0x1337));
        bytes32 topic2 = bytes32(uint256(0x1338));
        bytes32 topic3 = bytes32(uint256(0x1339));
        assembly {
            let memPtr := msize()
            log3(memPtr, 0x40, topic, topic2, topic3)
        }
        number++;
    }

    function emitFourTopics() public {
        emit LogFourTopics(0xfadedb00b, address(0xbeeffeed), keccak256(abi.encodePacked("QRSTUVWXYZ")));
        number++;
    }

    function emitFourTopicsExtraData() public {
        emit LogFourTopicsExtraData(
            0xdaff0d177, address(0xfeedcafe), keccak256(abi.encodePacked("ABCD")), bytes32(abi.encodePacked("ABCDEFG"))
        );
        number++;
    }

    function emitFourTopicsExtraDataAndMemExpansion() public {
        bytes32 topic = bytes32(uint256(0x1337));
        bytes32 topic2 = bytes32(uint256(0x1338));
        bytes32 topic3 = bytes32(uint256(0x1339));
        bytes32 topic4 = bytes32(uint256(0x133a));
        assembly {
            let memPtr := msize()
            log4(memPtr, 0x40, topic, topic2, topic3, topic4)
        }
        number++;
    }
}
