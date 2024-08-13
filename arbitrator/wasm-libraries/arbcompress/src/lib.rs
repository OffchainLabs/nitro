// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc)] // TODO: add safety docs

use brotli::{BrotliStatus, Dictionary};
use caller_env::{self, GuestPtr};
use paste::paste;

macro_rules! wrap {
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty);*) => {
        paste! {
            $(
                #[no_mangle]
                pub unsafe extern "C" fn [<arbcompress__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                    caller_env::brotli::$func_name(
                        &mut caller_env::static_caller::STATIC_MEM,
                        &mut caller_env::static_caller::STATIC_ENV,
                        $($arg_name),*
                    )
                }
            )*
        }
    };
}

wrap! {
    fn brotli_compress(
        in_buf_ptr: GuestPtr,
        in_buf_len: u32,
        out_buf_ptr: GuestPtr,
        out_len_ptr: GuestPtr,
        level: u32,
        window_size: u32,
        dictionary: Dictionary
    ) -> BrotliStatus;

    fn brotli_decompress(
        in_buf_ptr: GuestPtr,
        in_buf_len: u32,
        out_buf_ptr: GuestPtr,
        out_len_ptr: GuestPtr,
        dictionary: Dictionary
    ) -> BrotliStatus
}
