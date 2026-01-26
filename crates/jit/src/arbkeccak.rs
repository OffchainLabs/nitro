use crate::caller_env::{JitEnv, JitExecEnv};
use crate::machine::{MaybeEscape, WasmEnvMut};
use caller_env::GuestPtr;

pub fn keccak256(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, wenv) = src.jit_env();

    caller_env::arbkeccak::keccak256(
        &mut mem,
        &mut JitExecEnv { wenv },
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
    );
    Ok(())
}
