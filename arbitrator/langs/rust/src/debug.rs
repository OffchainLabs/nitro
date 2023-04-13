// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(dead_code)]

#[link(wasm_import_module = "console")]
extern "C" {
    pub(crate) fn log_txt(text: *const u8, len: usize);
    pub(crate) fn log_i32(value: i32);
    pub(crate) fn log_i64(value: i64);
    pub(crate) fn log_f32(value: f32);
    pub(crate) fn log_f64(value: f64);
}

pub fn println<T: AsRef<str>>(text: T) {
    let text = text.as_ref();
    unsafe { log_txt(text.as_ptr(), text.len()) };
}
