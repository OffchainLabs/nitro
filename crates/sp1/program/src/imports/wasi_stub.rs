//! WASI stubs — thin wrappers delegating to caller_env::wasip1_stub.

use wasmer::FunctionEnvMut;

use crate::{platform, replay::CustomEnvData};

pub fn proc_exit(mut ctx: FunctionEnvMut<CustomEnvData>, code: u32) {
    let (data, _store) = ctx.data_and_store_mut();

    if code == 0 {
        platform::print_string(
            1,
            format!(
                "Validation succeeds with hash 0x{}",
                hex::encode(data.input().large_globals[0])
            )
            .as_bytes(),
        );
    }

    platform::exit(code);
}
