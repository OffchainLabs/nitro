//! RPC-layer handlers for NodeInterface (0xc8) methods that require
//! chain-history or call-stack access beyond what a precompile can do.
//!
//! In Nitro these are handled by `InterceptRPCMessage` before the EVM
//! dispatches — we implement the same pattern as an `eth_call` override
//! on `ArbEthApi`. Precompile-level fallbacks for these methods return
//! zero / empty (see `arb_precompiles::nodeinterface`) so callers that
//! don't go through `eth_call` still get a valid response.

use alloy_primitives::{address, Address, Bytes, B256, U256};

/// NodeInterface virtual contract address.
pub const NODE_INTERFACE_ADDRESS: Address = address!("00000000000000000000000000000000000000c8");

// Function selectors (keccak256("name(arg types)")[0..4]).
pub const SEL_GAS_ESTIMATE_COMPONENTS: [u8; 4] = [0xc9, 0x4e, 0x6e, 0xeb];
pub const SEL_GAS_ESTIMATE_L1_COMPONENT: [u8; 4] = [0x77, 0xd4, 0x88, 0xa2];
pub const SEL_L2_BLOCK_RANGE_FOR_L1: [u8; 4] = [0x48, 0xe7, 0xf8, 0x11];
pub const SEL_GET_L1_CONFIRMATIONS: [u8; 4] = [0xe5, 0xca, 0x23, 0x8c];
pub const SEL_FIND_BATCH_CONTAINING_BLOCK: [u8; 4] = [0x81, 0xf1, 0xad, 0xaf];
pub const SEL_CONSTRUCT_OUTBOX_PROOF: [u8; 4] = [0x42, 0x69, 0x63, 0x50];

/// Decode a packed header's mix_hash field to `(sendCount, l1BlockNumber,
/// arbosVersion)`.
pub fn unpack_mix_hash(mix: B256) -> (u64, u64, u64) {
    let b = mix.0;
    let send_count = u64::from_be_bytes(b[0..8].try_into().unwrap_or_default());
    let l1_block = u64::from_be_bytes(b[8..16].try_into().unwrap_or_default());
    let arbos_version = u64::from_be_bytes(b[16..24].try_into().unwrap_or_default());
    (send_count, l1_block, arbos_version)
}

/// Extract the `bytes` parameter (data) from an ABI-encoded
/// `(address, bool, bytes)` gas-estimate call, returning its length.
///
/// Calldata layout:
///   selector(4) + address(32) + bool(32) + offset(32) + length(32) + data…
pub fn gas_estimate_data_len(input: &[u8]) -> u64 {
    if input.len() < 4 + 32 * 4 {
        return 0;
    }
    let len_start = 4 + 32 * 3;
    let len_bytes = &input[len_start..len_start + 32];
    U256::from_be_slice(len_bytes).try_into().unwrap_or(0u64)
}

/// ABI-encode the `(uint64, uint64)` result of `l2BlockRangeForL1`.
pub fn encode_l2_block_range(first: u64, last: u64) -> Bytes {
    let mut out = vec![0u8; 64];
    out[24..32].copy_from_slice(&first.to_be_bytes());
    out[56..64].copy_from_slice(&last.to_be_bytes());
    Bytes::from(out)
}

/// ABI-encode the `(uint64, uint64, uint256, uint256)` result of
/// `gasEstimateComponents`: `(gasEstimate, gasEstimateForL1, baseFee,
/// l1BaseFeeEstimate)`.
pub fn encode_gas_estimate_components(
    gas_total: u64,
    gas_for_l1: u64,
    basefee: U256,
    l1_base_fee: U256,
) -> Bytes {
    let mut out = vec![0u8; 128];
    out[24..32].copy_from_slice(&gas_total.to_be_bytes());
    out[56..64].copy_from_slice(&gas_for_l1.to_be_bytes());
    out[64..96].copy_from_slice(&basefee.to_be_bytes::<32>());
    out[96..128].copy_from_slice(&l1_base_fee.to_be_bytes::<32>());
    Bytes::from(out)
}

/// Decode the `uint64` argument from the selector `blockL1Num(uint64)` /
/// `l2BlockRangeForL1(uint64)` / `findBatchContainingBlock(uint64)` /
/// `nitroGenesisBlock()` (no arg, returns 0).
pub fn decode_single_u64_arg(input: &[u8]) -> Option<u64> {
    if input.len() < 4 + 32 {
        return None;
    }
    U256::from_be_slice(&input[4..36]).try_into().ok()
}

/// Binary-search headers to find the block-range that was emitted against
/// the given L1 block. The predicate on each header is
/// `l1_block_number_from_mix_hash(header.mix_hash)`.
///
/// Returns `(first_block, last_block)` inclusive. If no L2 block maps to
/// `target_l1_block`, returns `None`.
pub fn find_l2_block_range<F>(target_l1_block: u64, best: u64, mix_hash_of: F) -> Option<(u64, u64)>
where
    F: Fn(u64) -> Option<B256>,
{
    if best == 0 {
        return None;
    }

    // Lower bound: smallest L2 block N such that l1BlockNumber(N) >= target.
    let mut lo = 0u64;
    let mut hi = best;
    while lo < hi {
        let mid = lo + (hi - lo) / 2;
        let mix = mix_hash_of(mid)?;
        let (_, l1_bn, _) = unpack_mix_hash(mix);
        if l1_bn < target_l1_block {
            lo = mid + 1;
        } else {
            hi = mid;
        }
    }
    let first = lo;
    // Check we actually matched (may have overshot into a different L1 block).
    let first_mix = mix_hash_of(first)?;
    let (_, first_l1, _) = unpack_mix_hash(first_mix);
    if first_l1 != target_l1_block {
        return None;
    }

    // Upper bound: largest L2 block N such that l1BlockNumber(N) <= target.
    let mut lo = first;
    let mut hi = best;
    while lo < hi {
        let mid = lo + (hi - lo).div_ceil(2);
        let mix = mix_hash_of(mid)?;
        let (_, l1_bn, _) = unpack_mix_hash(mix);
        if l1_bn > target_l1_block {
            hi = mid - 1;
        } else {
            lo = mid;
        }
    }
    Some((first, lo))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unpack_mix_hash_layout() {
        let mut mix = [0u8; 32];
        mix[0..8].copy_from_slice(&42u64.to_be_bytes());
        mix[8..16].copy_from_slice(&100u64.to_be_bytes());
        mix[16..24].copy_from_slice(&30u64.to_be_bytes());
        let (sc, l1, v) = unpack_mix_hash(B256::from(mix));
        assert_eq!(sc, 42);
        assert_eq!(l1, 100);
        assert_eq!(v, 30);
    }

    #[test]
    fn gas_estimate_data_len_parses_abi_length() {
        let mut input = vec![0u8; 4 + 32 * 4 + 10];
        input[0..4].copy_from_slice(&SEL_GAS_ESTIMATE_COMPONENTS);
        // length is at offset 4 + 96
        let len_start = 4 + 96;
        input[len_start + 24..len_start + 32].copy_from_slice(&10u64.to_be_bytes());
        assert_eq!(gas_estimate_data_len(&input), 10);
    }

    #[test]
    fn gas_estimate_data_len_short_input_zero() {
        assert_eq!(gas_estimate_data_len(&[]), 0);
        assert_eq!(gas_estimate_data_len(&[0u8; 100]), 0);
    }

    #[test]
    fn encode_l2_block_range_pads_correctly() {
        let out = encode_l2_block_range(5, 10);
        assert_eq!(out.len(), 64);
        assert_eq!(U256::from_be_slice(&out[0..32]), U256::from(5u64));
        assert_eq!(U256::from_be_slice(&out[32..64]), U256::from(10u64));
    }

    #[test]
    fn encode_gas_estimate_components_layout() {
        let out = encode_gas_estimate_components(
            100_000,
            5_000,
            U256::from(1_000_000u64),
            U256::from(50_000_000u64),
        );
        assert_eq!(out.len(), 128);
        assert_eq!(U256::from_be_slice(&out[0..32]), U256::from(100_000u64));
        assert_eq!(U256::from_be_slice(&out[32..64]), U256::from(5_000u64));
        assert_eq!(U256::from_be_slice(&out[64..96]), U256::from(1_000_000u64));
        assert_eq!(U256::from_be_slice(&out[96..128]), U256::from(50_000_000u64));
    }

    #[test]
    fn decode_single_u64_arg_reads_last_8_bytes() {
        let mut input = vec![0u8; 4 + 32];
        input[4 + 24..4 + 32].copy_from_slice(&12345u64.to_be_bytes());
        assert_eq!(decode_single_u64_arg(&input), Some(12345));
    }

    #[test]
    fn decode_single_u64_arg_rejects_short_input() {
        assert_eq!(decode_single_u64_arg(&[0u8; 10]), None);
    }

    fn mix_with_l1_block(l1: u64) -> B256 {
        let mut m = [0u8; 32];
        m[8..16].copy_from_slice(&l1.to_be_bytes());
        B256::from(m)
    }

    #[test]
    fn find_l2_block_range_hits_exact_l1() {
        // L2 blocks 0..10, each maps to L1 block = 1000 + (L2 / 3).
        let mix_hash_of = |l2: u64| Some(mix_with_l1_block(1000 + l2 / 3));
        let range = find_l2_block_range(1001, 10, mix_hash_of);
        // L1 block 1001 = L2 blocks 3..=5
        assert_eq!(range, Some((3, 5)));
    }

    #[test]
    fn find_l2_block_range_miss_returns_none() {
        let mix_hash_of = |l2: u64| Some(mix_with_l1_block(1000 + l2 / 3));
        // Query a higher L1 block than any recorded.
        assert_eq!(find_l2_block_range(9999, 10, mix_hash_of), None);
    }

    #[test]
    fn find_l2_block_range_empty_chain() {
        assert_eq!(find_l2_block_range(1, 0, |_| None), None);
    }

    #[test]
    fn find_l2_block_range_first_block_match() {
        let mix_hash_of = |l2: u64| Some(mix_with_l1_block(1000 + l2));
        assert_eq!(find_l2_block_range(1000, 5, mix_hash_of), Some((0, 0)));
    }

    #[test]
    fn find_l2_block_range_all_same_l1() {
        let mix_hash_of = |_: u64| Some(mix_with_l1_block(42));
        assert_eq!(find_l2_block_range(42, 5, mix_hash_of), Some((0, 5)));
    }
}
