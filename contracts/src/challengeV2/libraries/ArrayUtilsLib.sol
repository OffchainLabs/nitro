// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

// CHRIS: TODO: test and documentation below
library ArrayUtilsLib {
    function append(bytes32[] memory arr, bytes32 newItem) internal pure returns (bytes32[] memory) {
        bytes32[] memory clone = new bytes32[](arr.length + 1);
        for (uint256 i = 0; i < arr.length; i++) {
            clone[i] = arr[i];
        }
        clone[clone.length - 1] = newItem;
        return clone;
    }

    // start index inclusive, end index not
    function slice(bytes32[] memory arr, uint256 startIndex, uint256 endIndex)
        internal
        pure
        returns (bytes32[] memory)
    {
        bytes32[] memory newArr = new bytes32[](endIndex - startIndex);
        for (uint256 i = startIndex; i < endIndex; i++) {
            newArr[i - startIndex] = arr[i];
        }
        return newArr;
    }

    function concat(bytes32[] memory arr1, bytes32[] memory arr2) internal pure returns (bytes32[] memory) {
        bytes32[] memory full = new bytes32[](arr1.length + arr2.length);
        for (uint256 i = 0; i < arr1.length; i++) {
            full[i] = arr1[i];
        }
        for (uint256 i = 0; i < arr2.length; i++) {
            full[arr1.length + i] = arr2[i];
        }
        return full;
    }
}