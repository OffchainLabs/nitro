// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::machine::WasmEnvMut;

macro_rules! reject {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub fn $f(_: WasmEnvMut, _: u32) {
                unimplemented!("link.rs {} not supported", stringify!($f));
            }
        )*
    }
}

reject!(
    compile_user_wasm,
    call_user_wasm,
    read_rust_vec,
    free_rust_vec,
    rust_params_impl,
);
