// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::callerenv::jit_env;
use crate::machine::Escape;
use crate::machine::WasmEnvMut;
use callerenv::{self, Uptr};

macro_rules! wrap {
    ($func_name:ident ($($arg_name:ident : $arg_type:ty),* ) -> $return_type:ty) => {
        pub fn $func_name(mut src: WasmEnvMut, $($arg_name : $arg_type),*) -> Result<$return_type, Escape> {
            let (mut mem, mut env) = jit_env(&mut src);

            Ok(callerenv::brotli::$func_name(&mut mem, &mut env, $($arg_name),*))
        }
    };
}

wrap!(brotli_decompress(
    in_buf_ptr: Uptr,
    in_buf_len: u32,
    out_buf_ptr: Uptr,
    out_len_ptr: Uptr) -> u32);

wrap!(brotli_compress(
    in_buf_ptr: Uptr,
    in_buf_len: u32,
    out_buf_ptr: Uptr,
    out_len_ptr: Uptr,
    level: u32,
    window_size: u32
) -> u32);
