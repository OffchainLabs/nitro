// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc)]

use arbutil::Bytes32;
use fnv::FnvHashMap as HashMap;
use lazy_static::lazy_static;
use parking_lot::Mutex;
use prover::programs::prelude::StylusConfig;

mod ink;
pub mod host;

pub(crate) static mut ARGS: Vec<u8> = vec![];
pub(crate) static mut OUTS: Vec<u8> = vec![];
pub(crate) static mut LOGS: Vec<Vec<u8>> = vec![];
pub(crate) static mut CONFIG: Option<StylusConfig> = None;
pub(crate) static mut OPEN_PAGES: u16 = 0;
pub(crate) static mut EVER_PAGES: u16 = 0;

lazy_static! {
    static ref KEYS: Mutex<HashMap<Bytes32, Bytes32>> = Mutex::new(HashMap::default());
}

/// Mock type representing a `user_host::Program`
pub struct Program;

#[no_mangle]
pub unsafe extern "C" fn user_test__prepare(
    len: usize,
    version: u32,
    max_depth: u32,
    ink_price: u64,
    hostio_ink: u64,
) -> *const u8 {
    let config = StylusConfig::new(version, max_depth, ink_price, hostio_ink);
    CONFIG = Some(config);
    ARGS = vec![0; len];
    ARGS.as_ptr()
}

#[no_mangle]
pub unsafe extern "C" fn user_test__set_pages(pages: u16) {
    OPEN_PAGES = OPEN_PAGES.saturating_add(pages);
    EVER_PAGES = EVER_PAGES.max(OPEN_PAGES);
}

#[no_mangle]
pub unsafe extern "C" fn user_test__get_outs_ptr() -> *const u8 {
    OUTS.as_ptr()
}

#[no_mangle]
pub unsafe extern "C" fn user_test__get_outs_len() -> usize {
    OUTS.len()
}
