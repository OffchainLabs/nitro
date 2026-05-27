use wasmer::FunctionEnvMut;

use crate::{MaybeEscape, platform, replay::CustomEnvData};

pub fn dump_elf(mut ctx: FunctionEnvMut<CustomEnvData>) -> MaybeEscape {
    let data = ctx.data_mut();
    assert!(!data.input_initialized());

    platform::dump_elf();

    Ok(())
}
