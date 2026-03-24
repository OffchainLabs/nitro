use crate::{Escape, MaybeEscape, Ptr, read_slice, stylus::StylusCustomEnvData};
use prover::value::Value;
use wasmer::FunctionEnvMut;

pub fn console_log_text(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    ptr: Ptr,
    len: u32,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let text = read_slice(ptr, len as usize, &memory)?;
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

pub fn start_benchmark(_ctx: FunctionEnvMut<StylusCustomEnvData>) -> MaybeEscape {
    unimplemented!()
}

pub fn end_benchmark(_ctx: FunctionEnvMut<StylusCustomEnvData>) -> MaybeEscape {
    unimplemented!()
}
