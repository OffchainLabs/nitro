// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{ffi::CStr, os::raw::c_char, ptr};

use eyre::Report;

pub unsafe fn c_string_to_string(c_str: *const c_char) -> eyre::Result<String> {
    if c_str.is_null() {
        eyre::bail!("unexpected null string pointer");
    }
    CStr::from_ptr(c_str)
        .to_str()
        .map(str::to_owned)
        .map_err(Report::from)
}

/// Copies the str-data into a libc free-able C string.
pub fn str_to_c_string(text: &str) -> *mut libc::c_char {
    unsafe {
        let buf = libc::malloc(text.len() + 1); // includes null-terminating byte
        if buf.is_null() {
            panic!("Failed to allocate memory for error string");
        }
        ptr::copy_nonoverlapping(text.as_ptr(), buf as *mut u8, text.len());
        *(buf as *mut u8).add(text.len()) = 0;
        buf as *mut libc::c_char
    }
}

pub fn err_to_c_string(err: Report) -> *mut libc::c_char {
    str_to_c_string(&format!("{err:?}"))
}
