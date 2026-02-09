// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)] // TODO: add safety docs

use arbutil::PreimageType;
use caller_env::{static_caller::STATIC_MEM, GuestPtr, MemAccess};
use core::convert::TryInto;
use core::ops::{Deref, DerefMut, Index, RangeTo};

extern "C" {
    pub fn wavm_get_globalstate_bytes32(idx: u32, ptr: *mut u8);
    pub fn wavm_set_globalstate_bytes32(idx: u32, ptr: *const u8);
    pub fn wavm_get_globalstate_u64(idx: u32) -> u64;
    pub fn wavm_set_globalstate_u64(idx: u32, val: u64);
    pub fn wavm_read_keccak_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_sha2_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_eth_versioned_hash_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_dacertificate_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_validate_certificate(ptr: *const u8, preimage_type: u8) -> u8;
}

#[repr(C, align(256))]
struct MemoryLeaf([u8; 32]);

impl Deref for MemoryLeaf {
    type Target = [u8; 32];

    fn deref(&self) -> &[u8; 32] {
        &self.0
    }
}

impl DerefMut for MemoryLeaf {
    fn deref_mut(&mut self) -> &mut [u8; 32] {
        &mut self.0
    }
}

impl Index<RangeTo<usize>> for MemoryLeaf {
    type Output = [u8];

    fn index(&self, index: RangeTo<usize>) -> &[u8] {
        &self.0[index]
    }
}

#[no_mangle]
pub unsafe extern "C" fn wavmio__getGlobalStateBytes32(idx: u32, out_ptr: GuestPtr) {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_get_globalstate_bytes32(idx, our_ptr);
    STATIC_MEM.write_slice(out_ptr, &our_buf[..32]);
}

/// Writes 32-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__setGlobalStateBytes32(idx: u32, src_ptr: GuestPtr) {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let value = STATIC_MEM.read_slice(src_ptr, 32);
    our_buf.copy_from_slice(&value);

    let our_ptr = our_buf.as_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_set_globalstate_bytes32(idx, our_ptr);
}

/// Reads 8-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__getGlobalStateU64(idx: u32) -> u64 {
    wavm_get_globalstate_u64(idx)
}

/// Writes 8-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__setGlobalStateU64(idx: u32, val: u64) {
    wavm_set_globalstate_u64(idx, val);
}

/// Reads an inbox message
#[no_mangle]
pub unsafe extern "C" fn wavmio__readInboxMessage(
    msg_num: u64,
    offset: usize,
    out_ptr: GuestPtr,
) -> usize {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);

    let read = wavm_read_inbox_message(msg_num, our_ptr, offset);
    assert!(read <= 32);
    STATIC_MEM.write_slice(out_ptr, &our_buf[..read]);
    read
}

/// Reads a delayed inbox message
#[no_mangle]
pub unsafe extern "C" fn wavmio__readDelayedInboxMessage(
    msg_num: u64,
    offset: usize,
    out_ptr: GuestPtr,
) -> usize {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);

    let read = wavm_read_delayed_inbox_message(msg_num, our_ptr, offset);
    assert!(read <= 32);
    STATIC_MEM.write_slice(out_ptr, &our_buf[..read]);
    read
}

/// Retrieves up to 32 bytes of the preimage of the given hash, at the given offset.
#[no_mangle]
pub unsafe extern "C" fn wavmio__resolveTypedPreimage(
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: usize,
    out_ptr: GuestPtr,
) -> usize {
    let mut our_buf = read_hash(hash_ptr);
    let read = read_preimage_slice(
        preimage_type.try_into().expect("unsupported preimage type"),
        our_buf.as_mut_ptr(),
        offset,
    );
    STATIC_MEM.write_slice(out_ptr, &our_buf[..read]);
    read
}

/// Retrieves up to the `allocated_output_space` bytes of the preimage of the given hash, at the
/// given offset.
#[no_mangle]
pub unsafe extern "C" fn wavmio__readPreimage(
    preimage_type: u8,
    hash_ptr: GuestPtr,
    out_ptr: GuestPtr,
    preimage_offset: u32,
    allocated_output_space: u32,
) -> usize {
    let hash = read_hash(hash_ptr);
    let preimage = read_full_preimage(preimage_type, hash);

    let preimage_len = preimage.len() as u32;
    assert!(preimage_offset < preimage_len, "preimage offset must be smaller than preimage length");

    let read_len = core::cmp::min(allocated_output_space, preimage_len - preimage_offset);
    let read_start = preimage_offset as usize;
    let read_end = read_start + read_len as usize;
    STATIC_MEM.write_slice(out_ptr, &preimage[read_start..read_end]);

    preimage_len as usize
}

/// Read the hash from guest memory into our aligned memory.
unsafe fn read_hash(hash_ptr: GuestPtr) -> MemoryLeaf {
    let mut buf = MemoryLeaf(STATIC_MEM.read_fixed(hash_ptr));
    let our_ptr = buf.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    buf
}

/// Read the full preimage in chunks of 32 bytes. All done in our memory.
unsafe fn read_full_preimage(preimage_type: u8, hash: MemoryLeaf) -> Vec<u8> {
    let preimage_type = preimage_type.try_into().expect("unsupported preimage type");
    let mut slices = vec![];
    for offset in (0..).step_by(32) {
        let mut hash = MemoryLeaf(*hash);
        let read = read_preimage_slice(preimage_type, hash.as_mut_ptr(), offset);
        slices.push(hash.0[..read].to_vec());
        if read < 32 {
            break;
        }
    }
    slices.concat()
}

/// Read up to 32 bytes of the preimage of the given hash, at the given offset.  The `our_ptr`
/// argument initially contains the hash, and will be overwritten with preimage data.
///
/// Returns the number of bytes read.
unsafe fn read_preimage_slice(
    preimage_type: PreimageType,
    our_ptr: *mut u8,
    offset: usize,
) -> usize {
    let preimage_reader = match preimage_type {
        PreimageType::Keccak256 => wavm_read_keccak_256_preimage,
        PreimageType::Sha2_256 => wavm_read_sha2_256_preimage,
        PreimageType::EthVersionedHash => wavm_read_eth_versioned_hash_preimage,
        PreimageType::DACertificate => wavm_read_dacertificate_preimage,
    };
    let read = preimage_reader(our_ptr, offset);
    assert!(read <= 32);
    read
}

/// Validates a DACertificate certificate, other preimage types are always valid.
#[no_mangle]
pub unsafe extern "C" fn wavmio__validateCertificate(preimage_type: u8, hash_ptr: GuestPtr) -> u8 {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let hash = STATIC_MEM.read_slice(hash_ptr, 32);
    our_buf.copy_from_slice(&hash);

    let our_ptr = our_buf.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);

    wavm_validate_certificate(our_ptr, preimage_type)
}

/// A hook called just before the first IO operation.
#[no_mangle]
pub unsafe extern "C" fn hooks__beforeFirstIO() {}
