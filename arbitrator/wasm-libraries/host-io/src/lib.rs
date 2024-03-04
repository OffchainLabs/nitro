// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use callerenv::{Uptr, MemAccess, static_caller::STATIC_MEM};

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

#[no_mangle]
pub unsafe extern "C" fn wavmio__getGlobalStateBytes32(idx: u32, out_ptr: Uptr) {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_get_globalstate_bytes32(idx, our_ptr);
    STATIC_MEM.write_slice(out_ptr, &our_buf.0[..32]);
}

/// Writes 32-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__setGlobalStateBytes32(idx: u32, src_ptr: Uptr) {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let value = STATIC_MEM.read_slice(src_ptr, 32);
    our_buf.0.copy_from_slice(&value);
    let our_ptr = our_buf.0.as_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_set_globalstate_bytes32(idx, our_ptr);
}

/// Reads 8-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__getGlobalStateU64(idx: u32) -> u64{
    wavm_get_globalstate_u64(idx)
}

/// Writes 8-bytes of global state
#[no_mangle]
pub unsafe extern "C" fn wavmio__setGlobalStateU64(idx: u32, val: u64) {
    wavm_set_globalstate_u64(idx, val);
}

/// Reads an inbox message
#[no_mangle]
pub unsafe extern "C" fn wavmio__readInboxMessage(msg_num: u64, offset: usize, out_ptr: Uptr) -> usize {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_inbox_message(msg_num, our_ptr, offset);
    assert!(read <= 32);
    STATIC_MEM.write_slice(out_ptr, &our_buf.0[..read]);
    read
}

/// Reads a delayed inbox message
#[no_mangle]
pub unsafe extern "C" fn wavmio__readDelayedInboxMessage(msg_num: u64, offset: usize, out_ptr: Uptr) -> usize {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_delayed_inbox_message(msg_num, our_ptr, offset as usize);
    assert!(read <= 32);
    STATIC_MEM.write_slice(out_ptr, &our_buf.0[..read]);
    read
}

/// Retrieves the preimage of the given hash.
#[no_mangle]
pub unsafe extern "C" fn wavmio__resolvePreImage(hash_ptr: Uptr, offset: usize, out_ptr: Uptr) -> usize {
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let hash = STATIC_MEM.read_slice(hash_ptr, 32);
    our_buf.0.copy_from_slice(&hash);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_pre_image(our_ptr, offset);
    assert!(read <= 32);
    STATIC_MEM.write_slice(out_ptr, &our_buf.0[..read]);
    read
}
