// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;
use go_abi::*;

extern "C" {
    pub fn wavm_get_globalstate_bytes32(idx: u32, ptr: *mut u8);
    pub fn wavm_set_globalstate_bytes32(idx: u32, ptr: *const u8);
    pub fn wavm_get_globalstate_u64(idx: u32) -> u64;
    pub fn wavm_set_globalstate_u64(idx: u32, val: u64);
    pub fn wavm_read_pre_image(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
}

#[repr(C, align(256))]
struct MemoryLeaf([u8; 32]);

/// Reads 32-bytes of global state
/// Safety: λ(idx uint64, output []byte)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_getGlobalStateBytes32(sp: usize) {
    let mut sp = GoStack::new(sp);
    let idx = sp.read_u64() as u32;
    let (out_ptr, mut out_len) = sp.read_go_slice();

    if out_len < 32 {
        eprintln!("Go attempting to read block hash into {out_len} bytes long buffer");
    } else {
        out_len = 32;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_get_globalstate_bytes32(idx, our_ptr);
    wavm::write_slice(&our_buf.0[..(out_len as usize)], out_ptr);
}

/// Writes 32-bytes of global state
/// Safety: λ(idx uint64, val []byte)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_setGlobalStateBytes32(sp: usize) {
    let mut sp = GoStack::new(sp);
    let idx = sp.read_u64() as u32;
    let (src_ptr, src_len) = sp.read_go_slice();

    if src_len != 32 {
        eprintln!("Go attempting to set block hash from {src_len} bytes long buffer");
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let value = wavm::read_slice(src_ptr, src_len);
    our_buf.0.copy_from_slice(&value);
    let our_ptr = our_buf.0.as_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_set_globalstate_bytes32(idx, our_ptr);
}

/// Reads 8-bytes of global state
/// Safety: λ(idx uint64) uint64
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_getGlobalStateU64(sp: usize) {
    let mut sp = GoStack::new(sp);
    let idx = sp.read_u64() as u32;
    sp.write_u64(wavm_get_globalstate_u64(idx));
}

/// Writes 8-bytes of global state
/// Safety: λ(idx uint64, val uint64)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_setGlobalStateU64(sp: usize) {
    let mut sp = GoStack::new(sp);
    let idx = sp.read_u64() as u32;
    wavm_set_globalstate_u64(idx, sp.read_u64());
}

/// Reads an inbox message
/// Safety: λ(msgNum uint64, offset uint32, output []byte) uint32
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_readInboxMessage(sp: usize) {
    let mut sp = GoStack::new(sp);
    let msg_num = sp.read_u64();
    let offset = sp.read_u64();
    let (out_ptr, out_len) = sp.read_go_slice();

    if out_len != 32 {
        eprintln!(
            "Go attempting to read inbox message with out len {}",
            out_len,
        );
        sp.write_u64(0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_inbox_message(msg_num, our_ptr, offset as usize);
    assert!(read <= 32);
    wavm::write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(read as u64);
}

/// Reads a delayed inbox message
/// Safety: λ(seqNum uint64, offset uint32, output []byte) uint32
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_readDelayedInboxMessage(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let seq_num = sp.read_u64();
    let offset = sp.read_u64();
    let (out_ptr, out_len) = sp.read_go_slice();

    if out_len != 32 {
        eprintln!(
            "Go attempting to read inbox message with out len {}",
            out_len,
        );
        sp.write_u64(0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_delayed_inbox_message(seq_num, our_ptr, offset as usize);
    assert!(read <= 32);
    wavm::write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(read as u64);
}

/// Retrieves the preimage of the given hash.
/// Safety: λ(hash []byte, offset uint32, output []byte) uint32
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_resolvePreImage(sp: usize) {
    let mut sp = GoStack::new(sp);
    let (hash_ptr, hash_len) = sp.read_go_slice();
    let offset = sp.read_u64();
    let (out_ptr, out_len) = sp.read_go_slice();

    if hash_len != 32 || out_len != 32 {
        eprintln!(
            "Go attempting to resolve pre image with hash len {} and out len {}",
            hash_len, out_len,
        );
        sp.write_u64(0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let hash = wavm::read_slice(hash_ptr, hash_len);
    our_buf.0.copy_from_slice(&hash);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_pre_image(our_ptr, offset as usize);
    assert!(read <= 32);
    wavm::write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(read as u64);
}
