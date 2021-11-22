use go_abi::*;

extern "C" {
    pub fn wavm_get_last_block_hash(ptr: *mut u8);
    pub fn wavm_set_last_block_hash(ptr: *const u8);
    pub fn wavm_advance_inbox_position();
    pub fn wavm_read_pre_image(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_get_position_within_message() -> u64;
    pub fn wavm_set_position_within_message(pos: u64);
    pub fn wavm_get_inbox_position() -> u64;
}

#[repr(C, align(256))]
struct MemoryLeaf([u8; 32]);

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_getLastBlockHash(sp: GoStack) {
    let out_ptr = sp.read_u64(0);
    let mut out_len = sp.read_u64(1);
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
    wavm_get_last_block_hash(our_ptr);
    write_slice(&our_buf.0[..(out_len as usize)], out_ptr);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_setLastBlockHash(sp: GoStack) {
    let src_ptr = sp.read_u64(0);
    let src_len = sp.read_u64(1);
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
    wavm_set_last_block_hash(our_ptr);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_advanceInboxMessage(
    _sp: GoStack,
) {
    wavm_advance_inbox_position();
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_readInboxMessage(sp: GoStack) {
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
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_readDelayedInboxMessage(sp: GoStack) {
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
    sp.write_u64(4, read as u64);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_resolvePreImage(sp: GoStack) {
    let hash_ptr = sp.read_u64(0);
    let hash_len = sp.read_u64(1);
    let offset = sp.read_u64(3);
    let out_ptr = sp.read_u64(4);
    let out_len = sp.read_u64(5);
    if hash_len != 32 || out_len != 32 {
        eprintln!(
            "Go attempting to resolve pre image with hash len {} and out len {}",
            hash_len, out_len,
        );
        sp.write_u64(7, 0);
        return;
    }
    let mut our_buf = MemoryLeaf([0u8; 32]);
    our_buf.0.copy_from_slice(&read_slice(hash_ptr, hash_len));
    let our_ptr = our_buf.0.as_mut_ptr();
    assert_eq!(our_ptr as usize % 32, 0);
    let read = wavm_read_pre_image(our_ptr, offset as usize);
    assert!(read <= 32);
    write_slice(&our_buf.0[..read], out_ptr);
    sp.write_u64(7, read as u64);
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_getPositionWithinMessage(
    sp: GoStack,
) {
    sp.write_u64(0, wavm_get_position_within_message());
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_setPositionWithinMessage(
    sp: GoStack,
) {
    wavm_set_position_within_message(sp.read_u64(0));
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_arbstate_wavmio_getInboxPosition(
    sp: GoStack,
) {
    sp.write_u64(0, wavm_get_inbox_position());
}
