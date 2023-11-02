// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{get_js, wavm, WavmJsEnv};
use arbutil::evm::{api::EvmApiMethod, js::ApiValue};

static mut STYLUS_RESULT: Option<Box<[ApiValue]>> = None;

/// Executes a Stylus closure, calling back into go via `resume`.
/// Returns the number of outputs, which are stored in the `STYLUS_RESULT` singleton.
///
/// # Safety
///
/// Corrupts the stack pointer. No `GoStack` functions may be ran until `sp.restore_stack()`,
/// which happens after `program_call_main` in `user-host`'s `link.rs`.
#[no_mangle]
pub unsafe extern "C" fn go_stub__run_api_closure(
    api_id: u32,
    method: EvmApiMethod,
    data: *const *const u8,
    lens: *const usize,
    num_args: usize,
) -> usize {
    let mut args = vec![];
    for i in 0..num_args {
        let data = wavm::caller_load32(data.add(i) as usize);
        let len = wavm::caller_load32(lens.add(i) as usize);
        let arg = wavm::read_slice_usize(data as usize, len as usize);
        args.push(ApiValue(arg));
    }

    let js = get_js();
    let js_env = &mut WavmJsEnv::new_sans_sp();
    let outs = js.call_stylus_func(api_id, method, args, js_env).unwrap();

    let num_outs = outs.len();
    STYLUS_RESULT = Some(outs.into_boxed_slice());
    num_outs
}

/// Copies the current closure results' lengths.
///
/// # Safety
///
/// Panics if no result exists.
/// Non-reentrant.
#[no_mangle]
pub unsafe extern "C" fn go_stub__read_api_result_lens(lens: *mut usize) {
    for (index, out) in STYLUS_RESULT.as_ref().unwrap().iter().enumerate() {
        wavm::caller_store32(lens.add(index) as usize, out.0.len() as u32);
    }
}

/// Copies the bytes of the current closure results, clearing the `STYLUS_RESULT` singleton.
///
/// # Safety
///
/// Panics if no result exists.
/// Unsound if `data` cannot store the bytes.
/// Non-reentrant.
#[no_mangle]
pub unsafe extern "C" fn go_stub__move_api_result_data(data: *const *mut u8) {
    for (index, out) in STYLUS_RESULT.take().unwrap().iter().enumerate() {
        let ptr = wavm::caller_load32(data.add(index) as usize);
        wavm::write_slice_usize(&out.0, ptr as usize)
    }
}
