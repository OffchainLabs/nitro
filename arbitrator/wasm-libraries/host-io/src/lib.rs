use arbutil::PreimageType;
use go_abi::*;
use std::convert::TryInto;

extern "C" {
    pub fn wavm_get_globalstate_bytes32(idx: u32, ptr: *mut u8);
    pub fn wavm_set_globalstate_bytes32(idx: u32, ptr: *const u8);
    pub fn wavm_get_globalstate_u64(idx: u32) -> u64;
    pub fn wavm_set_globalstate_u64(idx: u32, val: u64);
    pub fn wavm_read_keccak_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_sha2_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_eth_versioned_hash_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
}

#[repr(C, align(256))]
struct MemoryLeaf([u8; 32]);

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_getGlobalStateBytes32(
    sp: GoStack,
) {
    let idx = sp.read_u64(0) as u32;
    let out_ptr = sp.read_u64(1);
    let mut out_len = sp.read_u64(2);
    if out_len < 32 {
        eprintln!(
            "Go attempting to read block hash into {} bytes long buffer",
            out_len,
        );
    } else {
        out_len = 32;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_get_globalstate_bytes32(idx, our_ptr);
    write_slice(&our_buf.0[..(out_len as usize)], out_ptr);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_setGlobalStateBytes32(
    sp: GoStack,
) {
    let idx = sp.read_u64(0) as u32;
    let src_ptr = sp.read_u64(1);
    let src_len = sp.read_u64(2);
    if src_len != 32 {
        eprintln!(
            "Go attempting to set block hash from {} bytes long buffer",
            src_len,
        );
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    our_buf.0.copy_from_slice(&read_slice(src_ptr, src_len));
    let our_ptr = our_buf.0.as_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    wavm_set_globalstate_bytes32(idx, our_ptr);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_getGlobalStateU64(sp: GoStack) {
    let idx = sp.read_u64(0) as u32;
    sp.write_u64(1, wavm_get_globalstate_u64(idx));
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_setGlobalStateU64(sp: GoStack) {
    let idx = sp.read_u64(0) as u32;
    wavm_set_globalstate_u64(idx, sp.read_u64(1));
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_readInboxMessage(sp: GoStack) {
    let msg_num = sp.read_u64(0);
    let offset = sp.read_u64(1);
    let out_ptr = sp.read_u64(2);
    let out_len = sp.read_u64(3);
    if out_len != 32 {
        eprintln!(
            "Go attempting to read inbox message with out len {}",
            out_len,
        );
        sp.write_u64(5, 0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_inbox_message(msg_num, our_ptr, offset as usize);
    assert!(read <= 32);
    write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(5, read as u64);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_readDelayedInboxMessage(
    sp: GoStack,
) {
    let seq_num = sp.read_u64(0);
    let offset = sp.read_u64(1);
    let out_ptr = sp.read_u64(2);
    let out_len = sp.read_u64(3);
    if out_len != 32 {
        eprintln!(
            "Go attempting to read inbox message with out len {}",
            out_len,
        );
        sp.write_u64(4, 0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_delayed_inbox_message(seq_num, our_ptr, offset as usize);
    assert!(read <= 32);
    write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(5, read as u64);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_wavmio_resolveTypedPreimage(sp: GoStack) {
    let preimage_type = sp.read_u8(0);
    let hash_ptr = sp.read_u64(1);
    let hash_len = sp.read_u64(2);
    let offset = sp.read_u64(4);
    let out_ptr = sp.read_u64(5);
    let out_len = sp.read_u64(6);
    if hash_len != 32 || out_len != 32 {
        eprintln!(
            "Go attempting to resolve preimage with hash len {} and out len {}",
            hash_len, out_len,
        );
        sp.write_u64(8, 0);
        return;
    }
    let Ok(preimage_type) = preimage_type.try_into() else {
        eprintln!(
            "Go trying to resolve preimage with unknown type {}",
            preimage_type
        );
        sp.write_u64(8, 0);
        return;
    };
    let mut our_buf = MemoryLeaf([0u8; 32]);
    our_buf.0.copy_from_slice(&read_slice(hash_ptr, hash_len));
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let preimage_reader = match preimage_type {
        PreimageType::Keccak256 => wavm_read_keccak_256_preimage,
        PreimageType::Sha2_256 => wavm_read_sha2_256_preimage,
        PreimageType::EthVersionedHash => wavm_read_eth_versioned_hash_preimage,
    };
    let read = preimage_reader(our_ptr, offset as usize);
    assert!(read <= 32);
    write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(8, read as u64);
}
