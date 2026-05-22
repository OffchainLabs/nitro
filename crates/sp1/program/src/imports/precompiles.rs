use caller_env::GuestPtr;
use wasmer::FunctionEnvMut;

use crate::{
    Escape, MaybeEscape, platform,
    replay::CustomEnvData,
    state::sp1_env,
};

pub fn ecrecover(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    hash: GuestPtr,
    hash_len: u32,
    sig: GuestPtr,
    sig_len: u32,
    output: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::arbcrypto::ecrecovery(
        &mut mem,
        state,
        hash,
        hash_len,
        sig,
        sig_len,
        output,
    ))
}

pub fn keccak256(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    input: GuestPtr,
    input_length: u32,
    output: GuestPtr,
) -> MaybeEscape {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::arbcrypto::keccak256(&mut mem, state, input, input_length, output);
    Ok(())
}

pub fn dump_elf(mut ctx: FunctionEnvMut<CustomEnvData>) -> MaybeEscape {
    let data = ctx.data_mut();
    assert!(!data.input_initialized());

    platform::dump_elf();

    Ok(())
}
