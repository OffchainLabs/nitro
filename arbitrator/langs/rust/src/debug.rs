// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::hostio;

pub fn println<T: AsRef<str>>(text: T) {
    let text = text.as_ref();
    unsafe { hostio::log_txt(text.as_ptr(), text.len()) };
}
