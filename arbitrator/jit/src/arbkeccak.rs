use crate::caller_env::JitEnv;
use crate::machine::{MaybeEscape, WasmEnvMut};
use caller_env::{GuestPtr, MemAccess};
use core::mem::MaybeUninit;
use tiny_keccak::{Hasher, Keccak};

#[allow(clippy::too_many_arguments)]
pub fn keccak256(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, wenv) = src.jit_env();
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);

    let mut output = MaybeUninit::<[u8; 32]>::uninit();
    let mut hasher = Keccak::v256();
    hasher.update(input.as_ref());

    // SAFETY: finalize() writes 32 bytes
    unsafe {
        hasher.finalize(&mut *output.as_mut_ptr());
        mem.write_slice(out_buf_ptr, output.assume_init().as_slice());
    }
    Ok(())
}
