use caller_env::{GuestPtr, MemAccess};
use prover::value::Value;
use wasmer::FunctionEnvMut;

use crate::{
    Escape, MaybeEscape,
    stylus::{StylusCustomEnvData, stylus_env},
};

pub fn console_log_text(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    ptr: GuestPtr,
    len: u32,
) -> MaybeEscape {
    let (mem, _data) = stylus_env(&mut ctx);

    let text = mem.read_slice(ptr, len as usize);
    println!("Stylus says: {}", String::from_utf8_lossy(&text));
    Ok(())
}

pub fn console_log<T: Into<Value>>(
    _ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: T,
) -> MaybeEscape {
    let value = value.into();
    println!("Stylus says: {}", value);
    Ok(())
}

pub fn console_tee<T: Into<Value> + Copy>(
    _ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: T,
) -> Result<T, Escape> {
    println!("Stylus says: {}", value.into());
    Ok(value)
}

pub fn null_host(_ctx: FunctionEnvMut<StylusCustomEnvData>) {}

// Benchmarking records wall-clock time and ink for host-side debug tooling
// (jit returns it in its final cothread message). The zkVM has no wall clock
// and nothing consumes the data here, so these are deliberate no-ops.
pub fn start_benchmark(_ctx: FunctionEnvMut<StylusCustomEnvData>) -> MaybeEscape {
    Ok(())
}

pub fn end_benchmark(_ctx: FunctionEnvMut<StylusCustomEnvData>) -> MaybeEscape {
    Ok(())
}
