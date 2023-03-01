// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn debug_println(text: *const u8, len: usize);
}

pub fn println<T: AsRef<str>>(text: T) {
    let text = text.as_ref();
    unsafe { debug_println(text.as_ptr(), text.len()) };
}
