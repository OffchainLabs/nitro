// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Bytes32;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn emit_log(data: *const u8, len: usize, topics: usize);
}

pub fn log(topics: &[Bytes32], data: &[u8]) -> Result<(), &'static str> {
    if topics.len() > 4 {
        return Err("too many topics");
    }
    let mut bytes: Vec<u8> = vec![];
    bytes.extend(topics.iter().flat_map(|x| x.0.iter()));
    bytes.extend(data);
    unsafe { emit_log(bytes.as_ptr(), bytes.len(), topics.len()) }
    Ok(())
}

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn evm_blockhash(num: *const u8, dest: *mut u8);
    pub(crate) fn evm_gas_left() -> u64;
    pub(crate) fn evm_ink_left() -> u64;
}

pub fn blockhash(num: Bytes32) -> Option<Bytes32> {
    let mut dest = [0; 32];
    unsafe { evm_blockhash(num.ptr(), dest.as_mut_ptr()) };
    (dest != [0; 32]).then_some(Bytes32(dest))
}

pub fn gas_left() -> u64 {
    unsafe { evm_gas_left() }
}

pub fn ink_left() -> u64 {
    unsafe { evm_ink_left() }
}
