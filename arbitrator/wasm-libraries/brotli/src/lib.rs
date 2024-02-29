// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use callerenv::{
    self,
    MemAccess,
    ExecEnv,
    Uptr
};
use paste::paste;

macro_rules! wrap {
    ($func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty) => {
        paste! {
            #[no_mangle]
            pub unsafe extern "C" fn [<arbcompress__ $func_name>]($($arg_name : $arg_type),*) -> $return_type {
                callerenv::brotli::$func_name(
                    &mut callerenv::static_caller::STATIC_MEM,
                    &mut callerenv::static_caller::STATIC_ENV,
                    $($arg_name),*)
            }
        }
    };
}

wrap!(brotli_decompress(in_buf_ptr: Uptr, in_buf_len: u32, out_buf_ptr: Uptr, out_len_ptr: Uptr) -> u32);

wrap!(brotli_compress(in_buf_ptr: Uptr, in_buf_len: u32, out_buf_ptr: Uptr, out_len_ptr: Uptr, level: u32, window_size: u32) -> u32);