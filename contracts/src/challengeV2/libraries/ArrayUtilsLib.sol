// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

/// @title  Array utils library
/// @notice Utils for working with bytes32 arrays
library ArrayUtilsLib {
    /// @notice Append an item to the end of an array
    /// @param arr      The array to append to
    /// @param newItem  The item to append
    function append(bytes32[] memory arr, bytes32 newItem) internal pure returns (bytes32[] memory) {
        bytes32[] memory clone = new bytes32[](arr.length + 1);
        for (uint256 i = 0; i < arr.length; i++) {
            clone[i] = arr[i];
        }
        clone[clone.length - 1] = newItem;
        return clone;
    }

    /// @notice Get a slice of an existing array
    /// @dev    End index exlusive so slice(arr, 0, 5) gets the first 5 elements of arr
    /// @param arr          Array to slice
    /// @param startIndex   The start index of the slice in the original array - inclusive
    /// @param endIndex     The end index of the slice in the original array - exlusive
    function slice(bytes32[] memory arr, uint256 startIndex, uint256 endIndex)
        internal
        pure
        returns (bytes32[] memory)
    {
        require(startIndex < endIndex, "Start not less than end");
        require(endIndex <= arr.length, "End not less than length");

        bytes32[] memory newArr = new bytes32[](endIndex - startIndex);
        for (uint256 i = startIndex; i < endIndex; i++) {
            newArr[i - startIndex] = arr[i];
        }
        return newArr;
    }

    /// @notice Concatenated to arrays
    /// @param arr1 First array
    /// @param arr1 Second array
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