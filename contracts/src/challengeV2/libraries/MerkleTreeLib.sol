// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../../libraries/MerkleLib.sol";

library ArrayUtils {
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

// CHRIS: TODO: rework this file and add proper unit tests - not just round trips
// CHRIS: TODO: integration test with proof.go?
// CHRIS: TODO: revert everything to pure
// CHRIS: TODO: consolidate with the existing merklelib
// CHRIS: TODO: sort out hasstate
// CHRIS: TODO: copied from challengemanager lib - should remove and reuse
// CHRIS: TODO: uncomment the contents of hasState
// CHRIS: TODO: document and test hasState

library UintUtils {
    // Find the index (from least sig) of the least significant bit
    function leastSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x > 0, "Zero has no significant bits");

        uint256 i = 0;
        while ((x <<= 1) != 0) {
            ++i;
        }
        return 256 - i - 1;
    }

    // take from https://solidity-by-example.org/bitwise/
    // Find the index (from least sig) of the most significant bit using binary search
    function mostSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x != 0, "Zero has no significant bits");

        // x >= 2 ** 128
        if (x >= 0x100000000000000000000000000000000) {
            x >>= 128;
            msb += 128;
        }
        // x >= 2 ** 64
        if (x >= 0x10000000000000000) {
            x >>= 64;
            msb += 64;
        }
        // x >= 2 ** 32
        if (x >= 0x100000000) {
            x >>= 32;
            msb += 32;
        }
        // x >= 2 ** 16
        if (x >= 0x10000) {
            x >>= 16;
            msb += 16;
        }
        // x >= 2 ** 8
        if (x >= 0x100) {
            x >>= 8;
            msb += 8;
        }
        // x >= 2 ** 4
        if (x >= 0x10) {
            x >>= 4;
            msb += 4;
        }
        // x >= 2 ** 2
        if (x >= 0x4) {
            x >>= 2;
            msb += 2;
        }
        // x >= 2 ** 1
        if (x >= 0x2) msb += 1;
    }
}

library MerkleTreeLib {
    uint256 public constant MAX_LEVEL = 256;

    // CHRIS: TODO: update comments for size

    // Binary trees
    // --------------------------------------------------------------------------------------------
    // A complete tree is a balanced binary tree - each node has two children except the leaf
    // Leaves have no children, they are a complete tree of size one
    // Any tree (can be incomplete) can be represented as a collection of complete sub trees.
    // Since the tree is binary only one or zero complete tree at each level is enough to define any size of tree.
    // The root of a tree (incomplete or otherwise) is defined as the cumulative hashing of all of the
    // roots of each of it's complete and empty subtrees.
    // ---------
    // eg. Below a tree of size 3 is represented as the composition of 2 complete subtrees, one of size
    // 2 (AB) and one of size one (C).
    //    AB
    //   /  \
    //  A    B    C

    // Merkle expansions and roots
    // --------------------------------------------------------------------------------------------
    // The minimal amount of information we need to keep in order to compute the root of a tree
    // is the roots of each of it's sub trees, and the levels of each of those trees
    // A "merkle expansion" (ME) is this information - it is a vector of roots of each complete subtree,
    // the level of the tree being the index in the vector, the subtree root being the value.
    // The root is calculated by hashing each of the levels of the subtree together, adding zero hashes
    // where relevant to make a balanced tree.
    // ---------
    // eg. from the example above
    // ME of the AB tree = (0, AB), root=AB
    // ME of the C tree = (C), root=(C, 0)
    // ME of the composed ABC tree = (AB, C), root=hash(AB, hash(C, 0)) - here C is hashed with 0
    // to balance the tree, before then being hashed with AB.

    // Tree operations
    // --------------------------------------------------------------------------------------------
    // Binary trees are modified by adding or subtracting complete subtrees, however this libary
    // supports additive only trees since we dont have a specific use for subtraction at the moment.
    // We call adding a complete subtree to an existing tree "appending", appending has the following
    // rules:
    // 1. Only a complete sub trees can be appended
    // 2. Complete sub trees can only be appended at the level of the lowest complete subtree in the tree, or below
    // 3. If the existing tree is empty a sub tree can be appended at any level
    // When appending a sub tree we may increase the size of the merkle expansion vector, in the same
    // that adding 1 to a binary number may increase the index of its most significant bit
    // ---------
    // eg. A complete subtree can only be appended to the ABC tree at level 0, since the its lowest complete
    // subtree (C) is at level 0. Doing so would create a complete sub tree at level 1, which would in turn
    // cause the creation of new size 4 sub tree
    //
    //                                 ABCD
    //                               /     \
    //    AB                        AB     CD
    //   /  \         +       =    /  \   /  \
    //  A    B    C       D       A    B C    D
    //
    // ME of ABCD = (0, 0, ABCD), root=hash(AB, CD)
    // --------------------------------------------------------------------------------------------

    /// @notice The root of the subtree. A collision free commitment to the contents of the tree.
    /// @dev    The root of a tree is defined as the cumulative hashing of the
    //          roots of all of it's subtrees. Returns 0 for empty tree.
    function root(bytes32[] memory me) internal pure returns (bytes32) {
        bool empty = true;
        bytes32 accum = 0;
        for (uint256 i = 0; i < me.length; i++) {
            bytes32 val = me[i];
            if (empty) {
                if (val != 0) {
                    empty = false;
                    accum = val;

                    // the tree is balanced if the only non zero entry in the merkle extension
                    // us the last entry
                    // otherwise the lowest level entry needs to be combined with a zero to balance the bottom
                    // level, after which zeros in the merkle extension above that will balance the rest
                    if (i != me.length - 1) {
                        accum = keccak256(abi.encodePacked(accum, bytes32(0)));
                    }
                }
            } else if (val != 0) {
                accum = keccak256(abi.encodePacked(val, accum));
            } else {
                // by definition we always complete trees by appending zeros to the right
                accum = keccak256(abi.encodePacked(accum, bytes32(0)));
            }
        }

        return accum;
    }

    /// @notice Append a complete subtree to an existing tree
    /// @dev    See above description of trees for rules on how appending can occur.
    ///         Briefly, appending works like binary addition only that the value being added be an
    ///         exact power of two (complete), and must equal to or less than the least signficant bit
    ///         in the existing tree.
    ///         If the me is empty, will just append directly.
    function appendCompleteSubTree(bytes32[] memory me, uint256 level, bytes32 subtreeRoot)
        internal
        pure
        returns (bytes32[] memory)
    {
        // we use number representations of the levels elsewhere, so we need to ensure we're appending a leve
        // that's too high to use in uint
        require(level < MAX_LEVEL, "Level too high");
        require(subtreeRoot != 0, "Cannot append empty subtree");

        bytes32[] memory empty = new bytes32[](level + 1);
        if (me.length == 0) {
            for (uint256 i = 0; i <= level; i++) {
                if (i == level) {
                    empty[i] = subtreeRoot;
                    return empty;
                } else {
                    empty[i] = 0;
                }
            }
        }

        if (level >= me.length) {
            // This technically isn't necessary since it would be caught by the i < level check
            // on the last loop of the for-loop below, but we add it for a clearer error message
            revert("Level greater than highest level of current expansion");
        }

        bytes32 accumHash = subtreeRoot;
        bytes32[] memory next = new bytes32[](me.length);

        // loop through all the levels in self and try to append the new subtree
        // since each node has two children by appending a subtree we may complete another one
        // in the level above. So we move through the levels updating the result at each level
        for (uint256 i = 0; i < me.length; i++) {
            // we can only append at the level of the smallest complete sub tree or below
            // appending above this level would mean create "holes" in the tree
            // we can find the smallest complete sub tree by looking for the first entry in the merkle expansion
            if (i < level) {
                // we're below the level we want to append - no complete sub trees allowed down here
                // if the level is 0 there are no complete subtrees, and we therefore cannot be too low
                require(me[i] == 0, "Append above least significant bit");
            } else {
                // we're at or above the level
                if (accumHash == 0) {
                    // no more changes to propagate upwards - just fill the tree
                    next[i] = me[i];
                } else {
                    // we have a change to propagate
                    if (me[i] == 0) {
                        // if the level is currently empty we can just add the change
                        next[i] = accumHash;
                        // and then there's nothing more to propagate
                        accumHash = 0;
                    } else {
                        // if the level is not currently empty then we combine it with propagation
                        // change, and propagate that to the level above. This level is now part of a complete subtree
                        // so we zero it out
                        next[i] = 0;
                        accumHash = keccak256(abi.encodePacked(me[i], accumHash));
                    }
                }
            }
        }

        // we had a final change to propagate above the existing highest complete sub tree
        // so we have a new highest complete sub tree in the level above
        if (accumHash != 0) {
            next = ArrayUtils.append(next, accumHash);
        }

        require(next.length < MAX_LEVEL + 1, "Level too high");

        return next;
    }

    /// @notice Append a leaf to a subtree
    /// @dev    Leaves are just complete subtrees at level 0, however we hash the leaf before putting it
    ///         into the tree to avoid root collisions.
    function appendLeaf(bytes32[] memory me, bytes32 leaf) internal pure returns (bytes32[] memory) {
        // it's important that we hash the leaf, this ensures that this leaf cannot be a collision with any other non leaf
        // or root node, since these are always the hash of 64 bytes of data, and we're hashing 32 bytes
        return appendCompleteSubTree(me, 0, keccak256(abi.encodePacked(leaf)));
    }

    /// @notice Find the highest level which can be appended to tree of size startSize without
    ///         creating a tree with size greater than end size (inclusive)
    /// @dev    Subtrees can only be appended according to certain rules, see tree description at top of file
    ///         for details. A subtree can only be appended if it is at the same level, or below, the current lowest
    ///         subtree in the expansion
    function maximumAppendBetween(uint256 startSize, uint256 endSize) internal pure returns (uint256) {
        // Since the tree is binary we can represent it using the binary representation of a number
        // We use size here instead of height since height is zero indexed
        // As described above, subtrees can only be appended to a tree if they are at the same level, or below,
        // the current lowest subtree.
        // In this function we want to find the level of the highest tree that can be appended to the current
        // tree, without the resulting tree surpassing the end point. We do this by looking at the difference
        // between the start and end size, and iteratively reducing it in the maximal way.

        // The start and end size will share some higher order bits, below that they differ, and it is this
        // difference that we need to fill in the minimum number of appends
        // startSize looks like: xxxxxxyyyy
        // endSize looks like:   xxxxxxzzzz
        // where x are the complete sub trees they share, and y and z are the subtrees they dont

        require(startSize < endSize, "Start not less than end");

        // remove the high order bits that are shared
        uint256 msb = UintUtils.mostSignificantBit(startSize ^ endSize);
        uint256 mask = (1 << (msb) + 1) - 1;
        uint256 y = startSize & mask;
        uint256 z = endSize & mask;

        // Since in the verification we will be appending at start size, the highest level at which we
        // can append is the lowest complete subtree - the least significant bit
        if (y != 0) {
            return UintUtils.leastSignificantBit(y);
        }

        // y == 0, therefore we can append at any of levels where start and end differ
        // The highest level that we can append at without surpassing the end, is the most significant
        // bit of the end
        if (z != 0) {
            return UintUtils.mostSignificantBit(z);
        }

        // since we enforce that start < end, we know that y and z cannot both be 0
        revert("Both y and z cannot be zero");
    }

    /// @notice Verify that a pre-root commits to a prefix of the leaves committed by a post-root
    /// @dev    Verifies by appending sub trees to the pre tree until we get to the size of the post tree
    ///         and then checking that the root of the calculated post tree is equal to the supplied one
    function verifyPrefixProof(
        bytes32 preRoot,
        uint256 preSize,
        bytes32 postRoot,
        uint256 postSize,
        bytes32[] memory preExpansion,
        bytes32[] memory proof
    ) internal pure {
        require(preSize > 0, "Pre-size cannot be 0");
        require(root(preExpansion) == preRoot, "Pre expansion root mismatch");
        require(preSize < postSize, "Pre size not less than post size");

        uint256 size = preSize;
        uint256 proofIndex = 0;

        // Iteratively append a tree at the maximum possible level until we get to the post size
        while (size < postSize) {
            uint256 level = maximumAppendBetween(size, postSize);

            preExpansion = appendCompleteSubTree(preExpansion, level, proof[proofIndex]);

            uint256 numLeaves = 1 << level;
            size += numLeaves;
            assert(size <= postSize);
            proofIndex++;
        }

        // Check that the calculated root is equal to the provided post root
        require(root(preExpansion) == postRoot, "Post expansion root not equal post");

        // ensure that we consumed the full proof
        require(proofIndex == proof.length, "Incomplete proof usage");
    }

    /// @notice Using the provided proof verify that the provided leaf is included in the roothash at
    ///         the specified index
    function hasState(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] memory proof)
        internal
        pure
        returns (bool)
    {
        bytes32 calculatedRoot = MerkleLib.calculateRoot(proof, index, keccak256(abi.encodePacked(leaf)));
        return rootHash == calculatedRoot;
    }
}
