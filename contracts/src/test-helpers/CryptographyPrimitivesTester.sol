// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "../libraries/CryptographyPrimitives.sol";

library CryptographyPrimitivesTester {
    function keccakF(uint256[25] memory input) public pure returns (uint256[25] memory) {
        return CryptographyPrimitives.keccakF(input);
    }

    function sha256Block(bytes32[2] memory inputChunk, bytes32 hashState)
        public
        pure
        returns (bytes32)
    {
        return
            bytes32(
                CryptographyPrimitives.sha256Block(
                    [uint256(inputChunk[0]), uint256(inputChunk[1])],
                    uint256(hashState)
                )
            );
    }
}
