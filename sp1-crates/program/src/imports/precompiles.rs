use crate::{Escape, MaybeEscape, Ptr, keccak, platform, read_slice, replay::CustomEnvData};
use wasmer::FunctionEnvMut;

use secp256k1::{
    Message,
    ecdsa::{RecoverableSignature, RecoveryId},
};

pub fn ecrecover(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    hash: Ptr,
    sig: Ptr,
    output: Ptr,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let hash = read_slice(hash, 32, &memory)?;
    let sig = read_slice(sig, 65, &memory)?;

    let message = Message::from_digest(hash.try_into().unwrap());
    let Ok(recovery_id) = RecoveryId::from_i32(sig[64] as i32) else {
        return Ok(1);
    };
    let Ok(signature) = RecoverableSignature::from_compact(&sig[0..64], recovery_id) else {
        return Ok(2);
    };

    let Ok(public_key) = signature.recover(&message) else {
        return Ok(3);
    };
    let serialized_pub_key = public_key.serialize_uncompressed();

    memory.write(output.offset() as u64, &serialized_pub_key)?;
    Ok(0)
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
