// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

import "../../libraries/MerkleLib.sol";
import "./ArrayUtilsLib.sol";
import "./UintUtilsLib.sol";

/// @title  Binary merkle tree utilities
/// @notice
///         Binary trees
///         --------------------------------------------------------------------------------------------
///         A complete tree is a balanced binary tree - each node has two children except the leaf
///         Leaves have no children, they are a complete tree of size one
///         Any tree (can be incomplete) can be represented as a collection of complete sub trees.
///         Since the tree is binary only one or zero complete tree at each level is enough to define any size of tree.
///         The root of a tree (incomplete or otherwise) is defined as the cumulative hashing of all of the
///         roots of each of it's complete and empty subtrees.
///         ---------
///         eg. Below a tree of size 3 is represented as the composition of 2 complete subtrees, one of size
///         2 (AB) and one of size one (C).
///            AB
///           /  \
///          A    B    C
///
///         Merkle expansions and roots
///         --------------------------------------------------------------------------------------------
///         The minimal amount of information we need to keep in order to compute the root of a tree
///         is the roots of each of its sub trees, and the levels of each of those trees
///         A "merkle expansion" (ME) is this information - it is a vector of roots of each complete subtree,
///         the level of the tree being the index in the vector, the subtree root being the value.
///         The root is calculated by hashing each of the levels of the subtree together, adding zero hashes
///         where relevant to make a balanced tree.
///         ---------
///
///         ME Example 1
///
///         C => (C)
///
///         ME of the C tree = (C), root=(C)
///         The merkle expansion of a tree consisting of a single leaf is vector of size one with the
///         zeroth index being the leaf C. The zeroth index of the vector represents the presence of a size
///         one complete subtree in the overall tree. So if a tree has a size one complete subtree as part
///         of its composition, the root of that size one tree will be present in the zeroth index.
///
///         ME Example 2
///
///            AB
///           /  \
///          A    B
///
///         ME of the AB tree = (0, AB), root=AB
///         The merkle expansion of a tree consisting of a single size 2 complete subtree is a vector
///         of size 2, with the zeroth index value being 0, and the 1st index value being the root of the size
///         2 subtree. The zero in the zeroth index indicated that there is not a size 1 subtree in the tree's
///         composition. If a tree has a size 2 subtree in its composition its root will be present in the
///         1st index.
///
///         ME Example 3
///
///            AB
///           /  \
///          A    B    C
///
///         ME of the composed ABC tree = (C, AB), root=hash(AB, hash(C, 0)).
///         When a tree is not itself a complete subtree, but rather a composition, zeros are added when
///         calculating the root. To do this hash the first complete sub tree with zero, and from there
///         cumulatively hash the merkle expansion.
///         The merkle expansion of this composed tree is a vector of size two. Since it has a size one tree in
///         its composition the root of that goes in the zeroth index of the expansion - C, and since it has a
///         size two tree in its composition the root of that goes in the 1st index, to give (C, AB).
///
///         Tree operations
///         --------------------------------------------------------------------------------------------
///         Binary trees are modified by adding or subtracting complete subtrees, however this library
///         supports additive only trees since we dont have a specific use for subtraction at the moment.
///         We call adding a complete subtree to an existing tree "appending", appending has the following
///         rules:
///         1. Only a complete sub trees can be appended
///         2. Complete sub trees can only be appended at the level of the lowest complete subtree in the tree, or below
///         3. If the existing tree is empty a sub tree can be appended at any level
///         When appending a sub tree we may increase the size of the merkle expansion vector, in the same
///         that adding 1 to a binary number may increase the index of its most significant bit
///         ---------
///         eg. A complete subtree can only be appended to the ABC tree at level 0, since the its lowest complete
///         subtree (C) is at level 0. Doing so would create a complete sub tree at level 1, which would in turn
///         cause the creation of new size 4 sub tree
///
///                                         ABCD
///                                       /     \
///            AB                        AB     CD
///           /  \         +       =    /  \   /  \
///          A    B    C       D       A    B C    D
///
///         ME of ABCD = (0, AB) + (C) + (D)
///                    = (C, AB) + (D)
///                    = (0, 0, ABCD)
///         root of ABCD =hash(AB, CD)
///         --------------------------------------------------------------------------------------------
library MerkleTreeLib {
    // the go code uses uint64, so we ensure we never go above that here
    uint256 public constant MAX_LEVEL = 64;

    /// @notice The root of the subtree. A collision free commitment to the contents of the tree.
    /// @dev    The root of a tree is defined as the cumulative hashing of the
    ///         roots of all of it's subtrees. Throws error for empty tree
    /// @param me   The merkle expansion to calculate the root of
    function root(bytes32[] memory me) internal pure returns (bytes32) {
        require(me.length > 0, "Empty merkle expansion");
        require(me.length <= MAX_LEVEL, "Merkle expansion too large");

        bytes32 accum = 0;
        for (uint256 i = 0; i < me.length; i++) {
            bytes32 val = me[i];
            if (accum == 0) {
                if (val != 0) {
                    accum = val;

                    // the tree is balanced if the only non zero entry in the merkle extension
                    // is the last entry
                    // otherwise the lowest level entry needs to be combined with a zero to balance the bottom
                    // level, after which zeros in the merkle extension above that will balance the rest
                    if (i != me.length - 1) {
                        accum = keccak256(abi.encodePacked(accum, bytes32(0)));
                    }
                }
            } else if (val != 0) {
                // accum represents the smaller sub trees, since it is earlier in the expansion
                // we put the larger subtrees on the left
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
    ///         exact power of two (complete), and must equal to or less than the least significant bit
    ///         in the existing tree.
    ///         If the me is empty, will just append directly.
    /// @param me           The merkle expansion to append a complete sub tree to
    /// @param level        The level at which to append the complete subtree
    /// @param subtreeRoot  The root of the complete subtree to be appended
    function appendCompleteSubTree(bytes32[] memory me, uint256 level, bytes32 subtreeRoot)
        internal
        pure
        returns (bytes32[] memory)
    {
        // we use number representations of the levels elsewhere, so we need to ensure we're appending a leve
        // that's too high to use in uint
        require(level < MAX_LEVEL, "Level too high");
        require(subtreeRoot != 0, "Cannot append empty subtree");
        require(me.length <= MAX_LEVEL, "Merkle expansion too large");

        if (me.length == 0) {
            bytes32[] memory empty = new bytes32[](level + 1);
            empty[level] = subtreeRoot;
            return empty;
        }

        // This technically isn't necessary since it would be caught by the i < level check
        // on the last loop of the for-loop below, but we add it for a clearer error message
        require(level < me.length, "Level greater than highest level of current expansion");

        bytes32 accumHash = subtreeRoot;
        uint256 meSize = treeSize(me);
        uint256 postSize = meSize + 2 ** level;

        // if by appending the sub tree we increase the numbe of most sig bits of the size, that means
        // we'll need more space in the expansion to describe the tree, so we enlarge by one
        bytes32[] memory next = UintUtilsLib.mostSignificantBit(postSize) > UintUtilsLib.mostSignificantBit(meSize)
            ? new bytes32[](me.length + 1)
            : new bytes32[](me.length );

        // ensure we're never creating an expansion that's too big
        require(next.length <= MAX_LEVEL, "Append creates oversize tree");

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
        // so we have a new highest complete sub tree in the level above - this was why we
        // increased the storeage above
        if (accumHash != 0) {
            next[next.length - 1] = accumHash;
        }

        // it should never be possible to achieve this ever we sized the array correctly
        // so this is just a sanity check
        require(next[next.length - 1] != 0, "Last entry zero");

        return next;
    }

    /// @notice Append a leaf to a subtree
    /// @dev    Leaves are just complete subtrees at level 0, however we hash the leaf before putting it
    ///         into the tree to avoid root collisions.
    /// @param me   The merkle expansion to append a leaf to
    /// @param leaf The leaf to append - will be hashed in here before appending
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
    /// @param startSize    The size of the start tree to find the maximum append to
    /// @param endSize      The size of the end tree to find a maximum append under
    function maximumAppendBetween(uint256 startSize, uint256 endSize) internal pure returns (uint256) {
        // Since the tree is binary we can represent it using the binary representation of a number
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
        uint256 msb = UintUtilsLib.mostSignificantBit(startSize ^ endSize);
        uint256 mask = (1 << (msb) + 1) - 1;
        uint256 y = startSize & mask;
        uint256 z = endSize & mask;

        // Since in the verification we will be appending at start size, the highest level at which we
        // can append is the lowest complete subtree - the least significant bit
        if (y != 0) {
            return UintUtilsLib.leastSignificantBit(y);
        }

        // y == 0, therefore we can append at any of levels where start and end differ
        // The highest level that we can append at without surpassing the end, is the most significant
        // bit of the end
        if (z != 0) {
            return UintUtilsLib.mostSignificantBit(z);
        }

        // since we enforce that start < end, we know that y and z cannot both be 0
        revert("Both y and z cannot be zero");
    }

    /// @notice Calculate the full tree size represented by a merkle expansion
    /// @param me   The merkle expansion to calculate the tree size of
    function treeSize(bytes32[] memory me) internal pure returns (uint256) {
        uint256 sum = 0;
        for (uint256 i = 0; i < me.length; i++) {
            if (me[i] != 0) {
                sum += 2 ** i;
            }
        }
        return sum;
    }

    /// @notice Verify that a pre-root commits to a prefix of the leaves committed by a post-root
    /// @dev    Verifies by appending sub trees to the pre tree until we get to the size of the post tree
    ///         and then checking that the root of the calculated post tree is equal to the supplied one
    /// @param preRoot      The root of the tree which is a prefix of the post tree
    /// @param preSize      The size of the pre-tree
    /// @param postRoot     The root the post-tree - the tree which we're proving pre is a prefix of
    /// @param postSize     The size of the post-tree
    /// @param preExpansion The merkle expansion of the pre-tree
    /// @param proof        The proof is the minimum set of complete sub-tree hashes that can be appended to
    ///                     the pre-tree in order to form the post tree
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
        require(treeSize(preExpansion) == preSize, "Pre size does not match expansion");
        require(preSize < postSize, "Pre size not less than post size");

        uint256 size = preSize;
        uint256 proofIndex = 0;
        // we clone here to avoid mutating the input arguments
        // which could be unexpected for callers
        bytes32[] memory exp = ArrayUtilsLib.slice(preExpansion, 0, preExpansion.length);

        // Iteratively append a tree at the maximum possible level until we get to the post size
        while (size < postSize) {
            uint256 level = maximumAppendBetween(size, postSize);

            require(proofIndex < proof.length, "Index out of range");
            exp = appendCompleteSubTree(exp, level, proof[proofIndex]);

            uint256 numLeaves = 1 << level;
            size += numLeaves;
            assert(size <= postSize);
            proofIndex++;
        }

        // Check that the calculated root is equal to the provided post root
        require(root(exp) == postRoot, "Post expansion root not equal post");

        // ensure that we consumed the full proof
        // this is just a safety check to guard against mistakenly supplied args
        require(proofIndex == proof.length, "Incomplete proof usage");
    }

    /// @notice Using the provided proof verify that the provided leaf is included in the roothash at
    ///         the specified index. Note that here we use a 0-indexed value for the leaf number, whereas
    ///         elsewhere we use size.
    /// @param rootHash The root hash to prove inclusion in
    /// @param leaf     The leaf preimage to prove inclusion - will be hashed in here before checking inclusion
    /// @param index    The index of the leaf in the tree
    /// @param proof    The path from the leaf to the root
    function verifyInclusionProof(bytes32 rootHash, bytes32 leaf, uint256 index, bytes32[] memory proof)
        internal
        pure
    {
        bytes32 calculatedRoot = MerkleLib.calculateRoot(proof, index, keccak256(abi.encodePacked(leaf)));
        require(rootHash == calculatedRoot, "Invalid inclusion proof");
    }
}
