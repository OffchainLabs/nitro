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
    bytes.extend(topics.iter().map(|x| x.0.iter()).flatten());
    bytes.extend(data);
    unsafe { emit_log(bytes.as_ptr(), bytes.len(), topics.len()) }
    Ok(())
}
