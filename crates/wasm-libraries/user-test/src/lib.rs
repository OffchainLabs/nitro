// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)]

use arbutil::{evm::EvmData, Bytes32};
use fnv::FnvHashMap as HashMap;
use lazy_static::lazy_static;
use parking_lot::Mutex;
use prover::programs::prelude::StylusConfig;

pub mod host;
mod ink;
mod program;

#[derive(Default)]
struct GlobalState {
    args: Vec<u8>,
    outs: Vec<u8>,
    logs: Vec<Vec<u8>>,
    config: Option<StylusConfig>,
    open_pages: u16,
    ever_pages: u16,
}

lazy_static! {
    static ref GLOBAL_STATE: Mutex<GlobalState> = Default::default();
    static ref KEYS: Mutex<HashMap<Bytes32, Bytes32>> = Default::default();
    static ref EVM_DATA: EvmData = Default::default();
}

#[no_mangle]
pub unsafe extern "C" fn user_test__prepare(
    len: usize,
    version: u16,
    max_depth: u32,
    ink_price: u32,
) -> *const u8 {
    let config = StylusConfig::new(version, max_depth, ink_price);
    let mut gs = GLOBAL_STATE.lock();
    gs.config = Some(config);
    gs.args = vec![0; len];
    gs.args.as_ptr()
}

#[no_mangle]
pub unsafe extern "C" fn user_test__set_pages(pages: u16) {
    let mut gs = GLOBAL_STATE.lock();
    gs.open_pages = gs.open_pages.saturating_add(pages);
    gs.ever_pages = gs.ever_pages.max(gs.open_pages);
}

#[no_mangle]
pub unsafe extern "C" fn user_test__get_outs_ptr() -> *const u8 {
    GLOBAL_STATE.lock().outs.as_ptr()
}

#[no_mangle]
pub unsafe extern "C" fn user_test__get_outs_len() -> usize {
    GLOBAL_STATE.lock().outs.len()
}
