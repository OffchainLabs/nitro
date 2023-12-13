// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn wavm_set_error_policy(status: u32);
}

#[repr(u32)]
pub enum ErrorPolicy {
    ChainHalt,
    Recover,
}

pub unsafe fn set_error_policy(policy: ErrorPolicy) {
    wavm_set_error_policy(policy as u32);
}
