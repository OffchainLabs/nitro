use wasmer::FunctionEnvMut;

use crate::{
    Escape, MaybeEscape, Ptr, platform,
    replay::CustomEnvData,
    state::{gp, sp1_env},
};

pub fn ecrecover(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    hash: Ptr,
    hash_len: u32,
    sig: Ptr,
    sig_len: u32,
    output: Ptr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::arbcrypto::ecrecovery(
        &mut mem,
        state,
        gp(hash),
        hash_len,
        gp(sig),
        sig_len,
        gp(output),
    ))
}

pub fn keccak256(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    input: Ptr,
    input_length: u32,
    output: Ptr,
) -> MaybeEscape {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::arbcrypto::keccak256(&mut mem, state, gp(input), input_length, gp(output));
    Ok(())
}

pub fn dump_elf(mut ctx: FunctionEnvMut<CustomEnvData>) -> MaybeEscape {
    let data = ctx.data_mut();
    assert!(!data.input_initialized());

    platform::dump_elf();

    Ok(())
}
