// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

library MerkleExpansionLib {
    // A complete tree is a balanced binary tree - each node has two children except the leaf
    // Leaves have no children, they are a special type of complete tree of size one
    // A tree (can be incomplete) is the sum of a number of complete sub trees. Since the tree is binary
    // only one or zero complete tree at each height is enough to define any size of tree.
    // The root of a tree (incomplete or otherwise) is defined as the cumulative hashing of all of the
    // roots of each of it's complete subtrees.

    // The minimal amount of information we need to keep in order to compute the root of a tree
    // is the roots of each of it's sub trees, and the heights of each of those trees
    // A "merkle expansion" is this information - it contains the root of each sub tree, the height
    // of the tree is the index in the array, the subtree root is the value.

    function root(bytes32[] memory self) internal pure returns (bytes32) {
        bool empty = true;
        bytes32 accum = 0;
        for (uint256 i = 0; i < self.length; i++) {
            bytes32 val = self[i];
            if (empty) {
                if (val != 0) {
                    empty = false;
                    accum = val;
                } else {
                    // CHRIS: TODO: why dont we add the lower levels as 0 and cumulatively hash
                    // CHRIS: TODO: them. We dont technically need to because we hash a leaf when it's added
                    // CHRIS: TODO: but if we didnt we'd get a collision because of what we're doing here
                    // CHRIS: TODO: so maybe we should do it anyway for consistency
                }
            } else {
                accum = keccak256(abi.encodePacked(accum, val));
            }
        }

        return accum;
    }

    function isCompleteTree(bytes32[] memory self) internal pure returns (bool) {
        return root(self) == self[self.length - 1];
    }

    // Adding nodes to a merkle expansion is done by appending sub trees.
    // Subtrees are always appended on the right, in ascending order.
    // This means we cannot append a subtree at a height above the lowest complete
    // sub tree, as doing so would leave a "hole" in the tree

    // CHRIS: TODO: revert everything to pure

    function appendCompleteSubTree(bytes32[] memory self, uint256 level, bytes32 subtreeRoot)
        internal
        pure
        returns (bytes32[] memory)
    {
        // CHRIS: TODO: could assert some max sizes here?
        if (self.length == 0) {
            bytes32[] memory empty = new bytes32[](level + 1);
            for (uint256 i = 0; i <= level; i++) {
                if (i == level) {
                    empty[i] = subtreeRoot;
                    return empty;
                } else {
                    empty[i] = 0;
                }
            }
        }

        if (level >= self.length) {
            revert("Too high");
        }

        bytes32 accumHash = subtreeRoot;
        bytes32[] memory next = new bytes32[](self.length);

        // loop through all the levels in self and try to append the new subtree
        // since each node has two children by appending a subtree we may complete another one
        // in the level above. So we move through the levels updating the result at each level
        for (uint256 i = 0; i < self.length; i++) {
            // we can only append at the height of the smallest complete sub tree or below
            // appending above this height would mean create "holes" in the tree
            // we can find the smallest complete sub tree by looking for the first entry in the merkle expansion
            if (i < level) {
                // we're below the level we want to append - no complete sub trees allowed down here
                // if the level is 0 there are no complete subtrees, and we therefore cannot be too low
                require(self[i] == 0, "Too low");
            } else {
                // we're at or above the level
                if (accumHash == 0) {
                    // no more changes to propagate upwards - just fill the tree
                    next[i] = self[i];
                } else {
                    // we have a change to propagate
                    if (self[i] == 0) {
                        // if the level is currently empty we can just add the change
                        next[i] = accumHash;
                        // and then there's nothing more to propagate
                        accumHash = 0;
                    } else {
                        // if the level is not currently empty then we combine it with propagation
                        // change, and propagate that to the level above. This level is now part of a complete subtree
                        // so we zero it out
                        next[i] = 0;
                        accumHash = keccak256(abi.encodePacked(self[i], accumHash));
                    }
                }
            }
        }

        // we had a final change to propagate above the existing highest complete sub tree
        // so we have a new highest complete sub tree in the level above
        if (accumHash != 0) {
            // CHRIS: TODO: find a better way to do this - too much copying
            // CHRIS: TODO: we need to copy into a bigger array
            next = append(next, accumHash);
        }

        return next;
    }

    function appendLeaf(bytes32[] memory self, bytes32 leaf) internal pure returns (bytes32[] memory) {
        // it's important that we hash the leaf, this ensures that this leaf cannot be a collision with any other non leaf
        // or root node, since these are always the hash of 64 bytes of data, and we're hashing 32 bytes
        return appendCompleteSubTree(self, 0, keccak256(abi.encodePacked(leaf)));
    }
}

// CHRIS: TODO: move to other utils?
function append(bytes32[] memory arr, bytes32 newItem) pure returns (bytes32[] memory) {
    bytes32[] memory clone = new bytes32[](arr.length + 1);
    for (uint256 i = 0; i < arr.length; i++) {
        clone[i] = arr[i];
    }
    clone[clone.length - 1] = newItem;
    return clone;
}

// start index inclusive, end index not
// CHRIS: TODO: move to other utils?
function slice(bytes32[] memory arr, uint256 startIndex, uint256 endIndex) pure returns (bytes32[] memory) {
    bytes32[] memory newArr = new bytes32[](endIndex - startIndex);
    for (uint256 i = startIndex; i < endIndex; i++) {
        newArr[i - startIndex] = arr[i];
    }
    return newArr;
}

// CHRIS: TODO: rework this file and add proper unit tests
// CHRIS: TODO: integration test with proof.go?

library HistoryRootLib {
    function hasState(bytes32 historyRoot, bytes32 state, uint256 stateHeight, bytes memory proof)
        internal
        pure
        returns (bool)
    {
        // CHRIS: TODO: do a merkle proof check
        return true;
    }

    function expansionFromLeaves(bytes32[] memory leaves, uint256 leafStartIndex, uint256 leafEndIndex)
        internal
        pure
        returns (bytes32[] memory)
    {
        require(leafStartIndex < leafEndIndex, "Start not less than end");
        require(leafEndIndex <= leaves.length, "Leaf end not less than leaf length");
        bytes32[] memory expansion = new bytes32[](0);
        for (uint256 i = leafStartIndex; i < leafEndIndex; i++) {
            expansion = MerkleExpansionLib.appendLeaf(expansion, leaves[i]);
        }

        return expansion;
    }

    function leastSignificantBit(uint256 x) internal pure returns (uint256 msb) {
        require(x != 0, "Zero has no significant bits");
        // CHRIS: TODO: could do this in log time
        uint256 i = 0;
        while ((x <<= 1) != 0) {
            ++i;
        }
        return 256 - i - 1;
    }

    // CHRIS: TODO: copied from challengemanager lib - should remove and reuse
    // take from https://solidity-by-example.org/bitwise/
    // Find most significant bit using binary search
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

    function verifyPrefixProof(
        bytes32 pre,
        uint256 preHeight,
        bytes32 post,
        uint256 postHeight,
        bytes32[] memory preExpansion,
        bytes32[] memory proof
    ) internal pure {
        // CHRIS: TODO: check that preExpansion.root == pre
        require(MerkleExpansionLib.root(preExpansion) == pre, "Pre expansion root mismatch");

        uint256 height = preHeight;
        uint256 proofIndex = 0;
        // walk the tree from the current height all the way up to the post height
        // adding subtrees from the proof as we go

        while (height < postHeight) {
            // the binary representation of the height shows us the structure
            // of the binary tree at that height.
            // height looks like:        xxxxxxyyyy
            // postHeight looks like:    xxxxxxzzzz
            // where x are the complete subtrees that they share, and y and z
            // are where they differ.

            // find the bit index at which the trees differ

            uint256 msb = mostSignificantBit(height ^ postHeight);

            uint256 mask = (1 << (msb + 1)) - 1;

            // remove the parts of height and postHeight that are the same
            uint256 y = height & mask;
            uint256 z = postHeight & mask;

            uint256 level;
            if (y != 0) {
                level = leastSignificantBit(y);
            } else if (z != 0) {
                level = mostSignificantBit(z);
            } else {
                revert("y and z can only both be zero when height == postHeight");
            }

            preExpansion = MerkleExpansionLib.appendCompleteSubTree(preExpansion, level, proof[proofIndex]);

            uint256 numLeaves = 1 << level;

            height += numLeaves;

            proofIndex++;
        }

        require(proofIndex == proof.length, "Incomplete proof usage");
        require(MerkleExpansionLib.root(preExpansion) == post, "Post expansion root not equal post");
    }

    function generatePrefixProof(uint256 preHeight, bytes32[] memory newLeaves)
        internal
        pure
        returns (bytes32[] memory)
    {
        uint256 height = preHeight;

        uint256 postHeight = height + newLeaves.length;

        // CHRIS: TODO: better to assign 256 size here and then we dont need to append?
        bytes32[] memory proof = new bytes32[](0);

        // walk the tree from the current height all the way up to the post height
        // we want to find the list of subtrees which when appended to pre will return post
        while (height < postHeight) {
            // the binary representation of the height shows us the structure
            // of the binary tree at that height.
            // height looks like:        xxxxxxyyyy
            // postHeight looks like:    xxxxxxzzzz
            // where x are the complete subtrees that they share, and y and z
            // are where they differ. The difference between these two heights
            // shows us the subtrees we need to fill in order to get from height to postHeight

            // find the bit indx at which the trees differ
            uint256 msb = mostSignificantBit(height ^ postHeight);

            uint256 mask = (1 << (msb + 1)) - 1;

            // remove the parts of height and postHeight that are the same
            uint256 y = height & mask;

            uint256 z = postHeight & mask;

            uint256 level;
            if (y != 0) {
                // add subtrees in ascending order until y is 0
                level = leastSignificantBit(y);
            } else if (z != 0) {
                // add subtrees in ascending order until z is 0
                level = mostSignificantBit(z);
            } else {
                revert("y and z can only both be zero when height == postHeight");
            }

            // add 2^level leaves

            uint256 numLeaves = 1 << level;

            uint256 startIndex = height - preHeight;

            uint256 endIndex = startIndex + numLeaves;

            // form the subtree
            bytes32[] memory exp = expansionFromLeaves(newLeaves, startIndex, endIndex);
            proof = append(proof, MerkleExpansionLib.root(exp));
            height += numLeaves;
        }

        return proof;
    }
}
