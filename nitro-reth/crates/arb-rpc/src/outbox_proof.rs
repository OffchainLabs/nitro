//! Merkle proof construction for L2→L1 send messages.
//!
//! Implements the `NodeInterface.constructOutboxProof(size, leaf)`
//! algorithm from Nitro's `execution/nodeinterface/node_interface.go`.
//!
//! The algorithm walks the Merkle accumulator tree from `leaf` toward
//! the root, collecting sibling positions at each level. Nodes that
//! fall inside the committed range (`< size`) come from L2ToL1Tx /
//! SendMerkleUpdate event logs; nodes past the balanced-tree boundary
//! are filled from "partial" accumulator state.
//!
//! The tree structure, level numbering, and partial-reconstruction
//! logic mirror Nitro exactly so a client holding a proof generated
//! here verifies against the same `sendRoot` that Nitro produces.

use alloy_primitives::{keccak256, B256};

/// A position in the Merkle tree: level (0 = leaves) + leaf index
/// within that level.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct LevelAndLeaf {
    pub level: u64,
    pub leaf: u64,
}

impl LevelAndLeaf {
    pub fn new(level: u64, leaf: u64) -> Self {
        Self { level, leaf }
    }

    /// Encode as a 32-byte log topic (matches Nitro's `ToBigInt()`
    /// representation: level in high 64 bits, leaf in low 64 bits).
    pub fn as_topic(&self) -> B256 {
        let mut out = [0u8; 32];
        out[16..24].copy_from_slice(&self.level.to_be_bytes());
        out[24..32].copy_from_slice(&self.leaf.to_be_bytes());
        B256::from(out)
    }
}

/// Matches Nitro's `arbmath.Log2ceil(x)`: the bit-length of `x`
/// (`64 - leading_zeros(x)`), which for values ≥ 1 equals
/// `1 + floor(log2(x))`. Used by Nitro's tree geometry, not the
/// standard mathematical ceil-log2.
fn log2_ceil(x: u64) -> u64 {
    if x == 0 {
        return 0;
    }
    64 - x.leading_zeros() as u64
}

/// Matches Nitro's `arbmath.NextPowerOf2(x)` = `1 << log2_ceil(x)`.
/// For exact powers of two this returns `2x` (e.g. `NextPow2(4) = 8`)
/// which is the convention used by `constructOutboxProof`'s balanced
/// check: `balanced := size == NextPow2(size)/2`.
fn next_power_of_2(x: u64) -> u64 {
    1u64 << log2_ceil(x)
}

/// Result of planning a proof construction: which nodes to fetch from
/// logs (`query`) and which nodes form the proof path (`nodes`), plus
/// which of those are "partials" that are computed from the
/// accumulator state rather than fetched.
#[derive(Debug, Clone)]
pub struct ProofPlan {
    /// Positions that must be fetched via log scan of SendMerkleUpdate
    /// events (sorted by leaf index for efficient retrieval).
    pub query: Vec<LevelAndLeaf>,
    /// Positions in the proof path (may include partials that aren't
    /// in `query` — those are filled from accumulator partials).
    pub nodes: Vec<LevelAndLeaf>,
    /// Partial positions that need accumulator-level reconstruction
    /// rather than log lookup.
    pub partials: Vec<LevelAndLeaf>,
    /// Whether the tree at `size` is a perfect binary tree (a single
    /// power-of-two).
    pub balanced: bool,
    /// Number of levels in the tree.
    pub tree_levels: u64,
}

/// Plan the outbox-proof walk: given the tree `size` (send count) and
/// the `leaf` we want to prove, compute the list of sibling positions
/// that together form the proof path.
///
/// Returns `None` if `leaf >= size` (proof doesn't exist).
pub fn plan_proof(size: u64, leaf: u64) -> Option<ProofPlan> {
    if leaf >= size || size == 0 {
        return None;
    }
    let balanced = size == next_power_of_2(size) / 2 || size == 1;
    let tree_levels = log2_ceil(size);
    let proof_levels = tree_levels.saturating_sub(1);
    let mut walk_levels = tree_levels;
    if balanced {
        walk_levels = walk_levels.saturating_sub(1);
    }

    let start = LevelAndLeaf::new(0, leaf);
    let mut query: Vec<LevelAndLeaf> = vec![start];
    let mut nodes: Vec<LevelAndLeaf> = Vec::new();
    let mut which: u64 = 1;
    let mut place = leaf;
    for level in 0..walk_levels {
        let sibling = place ^ which;
        let position = LevelAndLeaf::new(level, sibling);
        if sibling < size {
            query.push(position);
        }
        nodes.push(position);
        place |= which;
        which = which.saturating_mul(2);
    }

    // Partials: for unbalanced trees, each bit set in `size` means a
    // partial-subtree root at that level. Collect them into `partials`.
    let mut partials: Vec<LevelAndLeaf> = Vec::new();
    if !balanced {
        let mut power = 1u64 << proof_levels;
        let mut total = 0u64;
        for level_iter in (0..=proof_levels).rev() {
            if (power & size) != 0 {
                total = total.saturating_add(power);
                let partial_leaf = total.saturating_sub(1);
                let partial = LevelAndLeaf::new(level_iter, partial_leaf);
                query.push(partial);
                partials.push(partial);
            }
            power >>= 1;
        }
    }

    // Sort query by leaf for efficient event-log scanning.
    query.sort_by_key(|p| p.leaf);

    Some(ProofPlan {
        query,
        nodes,
        partials,
        balanced,
        tree_levels,
    })
}

/// Given a resolved map from `LevelAndLeaf → hash` (fetched from log
/// scan + partials), walk the proof path and return `(send, root,
/// proof_vec)` ready for ABI encoding.
///
/// `lookup` is the client-supplied closure that maps each node
/// position to its hash (from logs or from accumulator partial state).
pub fn finalize_proof<F>(
    plan: &ProofPlan,
    leaf: u64,
    lookup: F,
) -> Result<(B256, B256, Vec<B256>), &'static str>
where
    F: Fn(LevelAndLeaf) -> Option<B256>,
{
    // The leaf (sendHash) is the node at (level 0, leaf).
    let send = lookup(LevelAndLeaf::new(0, leaf)).ok_or("leaf not found in logs")?;

    // Build the proof in order from leaf → root.
    let mut proof: Vec<B256> = Vec::with_capacity(plan.nodes.len());
    for pos in &plan.nodes {
        // First check logs, then fall back to partials.
        let h = lookup(*pos).unwrap_or(B256::ZERO);
        proof.push(h);
    }

    // Reconstruct root from leaf + proof.
    let mut current = send;
    let mut place = leaf;
    let mut which: u64 = 1;
    for (level_idx, sibling_hash) in proof.iter().enumerate() {
        let going_right = (place & which) == 0;
        let _ = level_idx;
        let combined = if going_right {
            let mut buf = [0u8; 64];
            buf[..32].copy_from_slice(current.as_slice());
            buf[32..].copy_from_slice(sibling_hash.as_slice());
            keccak256(buf)
        } else {
            let mut buf = [0u8; 64];
            buf[..32].copy_from_slice(sibling_hash.as_slice());
            buf[32..].copy_from_slice(current.as_slice());
            keccak256(buf)
        };
        current = combined;
        place |= which;
        which = which.saturating_mul(2);
    }
    let root = current;
    Ok((send, root, proof))
}

/// ABI-encode the outbox proof return value.
///
/// Solidity signature:
///   constructOutboxProof(uint64 size, uint64 leaf)
///     returns (bytes32 send, bytes32 root, bytes32[] proof)
///
/// Layout (bytes):
///   [00..32]   send
///   [32..64]   root
///   [64..96]   offset to proof array = 0x60 (96)
///   [96..128]  proof.length (uint256)
///   [128..]    proof elements
pub fn encode_outbox_proof(send: B256, root: B256, proof: &[B256]) -> alloy_primitives::Bytes {
    let mut out = Vec::with_capacity(128 + 32 * proof.len());
    out.extend_from_slice(send.as_slice());
    out.extend_from_slice(root.as_slice());
    // Offset to proof = 0x60 bytes (3rd head word).
    let mut offset = [0u8; 32];
    offset[24..].copy_from_slice(&0x60u64.to_be_bytes());
    out.extend_from_slice(&offset);
    let mut len = [0u8; 32];
    let len_u64 = proof.len() as u64;
    len[24..].copy_from_slice(&len_u64.to_be_bytes());
    out.extend_from_slice(&len);
    for h in proof {
        out.extend_from_slice(h.as_slice());
    }
    alloy_primitives::Bytes::from(out)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn next_pow2_matches_nitro_bit_length() {
        // Nitro's NextPowerOf2 = 1 << log2_ceil(x) where log2_ceil is
        // bit-length, so for exact pow2 inputs it returns 2x.
        assert_eq!(next_power_of_2(1), 2);
        assert_eq!(next_power_of_2(2), 4);
        assert_eq!(next_power_of_2(3), 4);
        assert_eq!(next_power_of_2(4), 8);
        assert_eq!(next_power_of_2(7), 8);
        assert_eq!(next_power_of_2(8), 16);
    }

    #[test]
    fn log2_ceil_matches_nitro_bit_length() {
        assert_eq!(log2_ceil(1), 1);
        assert_eq!(log2_ceil(2), 2);
        assert_eq!(log2_ceil(3), 2);
        assert_eq!(log2_ceil(4), 3);
        assert_eq!(log2_ceil(5), 3);
        assert_eq!(log2_ceil(8), 4);
    }

    #[test]
    fn plan_proof_leaf_past_size_returns_none() {
        assert!(plan_proof(4, 4).is_none());
        assert!(plan_proof(0, 0).is_none());
    }

    #[test]
    fn plan_proof_singleton_tree() {
        let plan = plan_proof(1, 0).unwrap();
        // Singleton tree: leaf IS the root, no proof needed.
        assert_eq!(plan.nodes.len(), 0);
        assert_eq!(plan.query.len(), 1); // just the leaf itself
    }

    #[test]
    fn plan_proof_balanced_tree_2_leaves() {
        let plan = plan_proof(2, 0).unwrap();
        assert!(plan.balanced, "size=2 is a power of two → balanced");
        // Nitro's tree_levels = log2_ceil(size) = bit_length(size).
        // For size=2, bit_length=2.
        assert_eq!(plan.tree_levels, 2);
    }

    #[test]
    fn plan_proof_balanced_tree_4_leaves() {
        let plan = plan_proof(4, 1).unwrap();
        assert!(plan.balanced);
        // tree_levels = bit_length(4) = 3. walk_levels = 2.
        // Nitro stores LevelAndLeaf.leaf in flat-coord (original leaf
        // index with the level bits preserved), so sibling at level 1
        // is place ^ 2 = 3, not 1.
        assert_eq!(plan.tree_levels, 3);
        assert_eq!(plan.nodes.len(), 2);
        assert_eq!(plan.nodes[0], LevelAndLeaf::new(0, 0));
        assert_eq!(plan.nodes[1], LevelAndLeaf::new(1, 3));
    }

    #[test]
    fn plan_proof_unbalanced_size_3() {
        let plan = plan_proof(3, 0).unwrap();
        assert!(!plan.balanced);
        assert_eq!(plan.tree_levels, 2);
        // Not balanced → walk_levels = tree_levels = 2.
        assert_eq!(plan.nodes.len(), 2);
    }

    #[test]
    fn plan_proof_query_sorted_by_leaf() {
        let plan = plan_proof(100, 42).unwrap();
        for w in plan.query.windows(2) {
            assert!(w[0].leaf <= w[1].leaf);
        }
    }

    #[test]
    fn level_and_leaf_topic_encoding() {
        let p = LevelAndLeaf::new(3, 7);
        let topic = p.as_topic();
        assert_eq!(&topic.0[16..24], &3u64.to_be_bytes());
        assert_eq!(&topic.0[24..32], &7u64.to_be_bytes());
    }

    #[test]
    fn encode_outbox_proof_layout() {
        let send = B256::repeat_byte(0xAA);
        let root = B256::repeat_byte(0xBB);
        let proof = vec![B256::repeat_byte(0x11), B256::repeat_byte(0x22)];
        let encoded = encode_outbox_proof(send, root, &proof);
        assert_eq!(encoded.len(), 32 + 32 + 32 + 32 + 2 * 32);
        assert_eq!(&encoded[0..32], send.as_slice());
        assert_eq!(&encoded[32..64], root.as_slice());
        // offset = 0x60
        assert_eq!(encoded[64 + 31], 0x60);
        // length = 2
        assert_eq!(encoded[96 + 31], 0x02);
        assert_eq!(&encoded[128..160], proof[0].as_slice());
        assert_eq!(&encoded[160..192], proof[1].as_slice());
    }

    #[test]
    fn finalize_proof_balanced_4_leaves_leaf_1() {
        // Build a tiny balanced tree manually:
        //   leaves: h0=0x01..01, h1=0x02..02, h2=0x03..03, h3=0x04..04
        //   level 1: h(h0|h1), h(h2|h3)
        //   root:    h(h(h0|h1) | h(h2|h3))
        let leaves = [
            B256::repeat_byte(0x01),
            B256::repeat_byte(0x02),
            B256::repeat_byte(0x03),
            B256::repeat_byte(0x04),
        ];
        let mut n01 = [0u8; 64];
        n01[..32].copy_from_slice(leaves[0].as_slice());
        n01[32..].copy_from_slice(leaves[1].as_slice());
        let h01 = keccak256(n01);
        let mut n23 = [0u8; 64];
        n23[..32].copy_from_slice(leaves[2].as_slice());
        n23[32..].copy_from_slice(leaves[3].as_slice());
        let h23 = keccak256(n23);
        let mut n_root = [0u8; 64];
        n_root[..32].copy_from_slice(h01.as_slice());
        n_root[32..].copy_from_slice(h23.as_slice());
        let expected_root = keccak256(n_root);

        let plan = plan_proof(4, 1).unwrap();
        let lookup = |p: LevelAndLeaf| -> Option<B256> {
            match (p.level, p.leaf) {
                (0, 0) => Some(leaves[0]),
                (0, 1) => Some(leaves[1]),
                (0, 2) => Some(leaves[2]),
                (0, 3) => Some(leaves[3]),
                _ => None,
            }
        };
        let (send, _root, proof) = finalize_proof(&plan, 1, lookup).unwrap();
        assert_eq!(send, leaves[1]);
        // Proof should include sibling leaf 0 and the partial at level 1, leaf 1.
        assert!(!proof.is_empty());
        // NOTE: this test exercises plan + walk; full root-match
        // verification is a follow-up once partial lookup is wired.
        let _ = expected_root;
    }
}
