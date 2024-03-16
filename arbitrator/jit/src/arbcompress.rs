// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::caller_env::{JitEnv, JitExecEnv};
use crate::machine::Escape;
use crate::machine::WasmEnvMut;
use caller_env::brotli::{BrotliStatus, Dictionary};
use caller_env::{self, GuestPtr};

macro_rules! wrap {
    ($(fn $func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty);*) => {
        $(
            pub fn $func_name(mut src: WasmEnvMut, $($arg_name : $arg_type),*) -> Result<$return_type, Escape> {
                let (mut mem, wenv) = src.jit_env();

                Ok(caller_env::brotli::$func_name(&mut mem, &mut JitExecEnv { wenv }, $($arg_name),*))
            }
        )*
    };
}

wrap! {
    fn brotli_decompress(
        in_buf_ptr: GuestPtr,
        in_buf_len: u32,
        out_buf_ptr: GuestPtr,
        out_len_ptr: GuestPtr,
        dictionary: Dictionary
    ) -> BrotliStatus;

    fn brotli_compress(
        in_buf_ptr: GuestPtr,
        in_buf_len: u32,
        out_buf_ptr: GuestPtr,
        out_len_ptr: GuestPtr,
        level: u32,
        window_size: u32
    ) -> BrotliStatus
}
