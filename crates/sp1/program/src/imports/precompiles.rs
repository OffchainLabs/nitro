use wasmer::FunctionEnvMut;

use crate::{
    Escape, MaybeEscape, Ptr, keccak, platform, read_slice,
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
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let input = read_slice(input, input_length as usize, &memory)?;
    let hash = keccak(input);
    memory.write(output.offset() as u64, &hash)?;

    Ok(())
}

pub fn dump_elf(mut ctx: FunctionEnvMut<CustomEnvData>) -> MaybeEscape {
    let data = ctx.data_mut();
    assert!(!data.input_initialized());

    platform::dump_elf();

    Ok(())
}
