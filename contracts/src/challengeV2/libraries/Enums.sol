// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/bold/blob/main/LICENSE
// SPDX-License-Identifier: BUSL-1.1
//
pragma solidity ^0.8.17;

/// @notice The status of the edge
/// - Pending: Yet to be confirmed. Not all edges can be confirmed.
/// - Confirmed: Once confirmed it cannot transition back to pending
enum EdgeStatus {
    Pending,
    Confirmed
}

/// @notice The type of the edge. Challenges are decomposed into 3 types of subchallenge
///         represented here by the edge type. Edges are initially created of type Block
///         and are then bisected until they have length one. After that new BigStep edges are
///         added that claim a Block type edge, and are then bisected until they have length one.
///         Then a SmallStep edge is added that claims a length one BigStep edge, and these
///         SmallStep edges are bisected until they reach length one. A length one small step edge
///         can then be directly executed using a one-step proof.
enum EdgeType {
    Block,
    BigStep,
    SmallStep
}
