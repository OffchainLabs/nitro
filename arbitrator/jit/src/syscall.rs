// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    gostack::{GoRuntimeState, GoStack},
    machine::{Escape, MaybeEscape, WasmEnvFuncs, WasmEnvMut},
};

use arbutil::Color;
use go_js::JsEnv;
use wasmer::TypedFunction;

/// go side: λ(v value)
pub fn js_finalize_ref(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let val = sp.read_js();
    env.js_state.finalize_ref(val);
}

/// go side: λ(v value, field string) value
pub fn js_value_get(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = sp.read_js();
    let field = sp.read_string();

    let result = env.js_state.value_get(source, &field);
    sp.write_js(result);
}

/// go side: λ(v value, field string, x value)
pub fn js_value_set(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = sp.read_js();
    let field = sp.read_string();
    let new_value = sp.read_js();

    env.js_state.value_set(source, &field, new_value);
}

/// go side: λ(v value, i int) value
pub fn js_value_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = sp.read_js();
    let index = sp.read_go_ptr() as usize;

    let result = env.js_state.value_index(source, index);
    sp.write_js(result);
}

/// go side: λ(array value, i int, v value)
pub fn js_value_set_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = sp.read_js();
    let index = sp.read_go_ptr() as usize;
    let value = sp.read_js();

    env.js_state.value_set_index(source, index, value);
}

pub struct WasmerJsEnv<'a, 'b> {
    rng: &'a mut rand_pcg::Pcg32,
    resume: &'a TypedFunction<(), ()>,
    get_stack_pointer: &'a TypedFunction<(), i32>,
    go_stack: &'a mut GoStack<'b>,
}

impl<'a, 'b> WasmerJsEnv<'a, 'b> {
    pub fn new(
        go_stack: &'a mut GoStack<'b>,
        exports: &'a mut WasmEnvFuncs,
        go_state: &'a mut GoRuntimeState,
    ) -> Result<Self, Escape> {
        let Some(resume) = &exports.resume else {
            return Escape::failure(format!("wasmer failed to bind {}", "resume".red()));
        };
        let Some(get_stack_pointer) = &exports.get_stack_pointer else {
            return Escape::failure(format!("wasmer failed to bind {}", "getsp".red()));
        };

        Ok(Self {
            rng: &mut go_state.rng,
            resume,
            get_stack_pointer,
            go_stack,
        })
    }
}

impl<'a, 'b> JsEnv for WasmerJsEnv<'a, 'b> {
    fn get_rng(&mut self) -> &mut dyn rand::RngCore {
        &mut self.rng
    }

    fn resume(&mut self) -> eyre::Result<()> {
        let go_stack = &mut *self.go_stack;
        let store = &mut go_stack.store;

        self.resume.call(store)?;

        // save our progress from the stack pointer
        let saved = go_stack.top - go_stack.sp;

        // recover the stack pointer
        let pointer = self.get_stack_pointer.call(store)? as u32;
        go_stack.sp = pointer;
        go_stack.top = pointer + saved;
        Ok(())
    }
}

/// go side: λ(v value, args []value) (value, bool)
pub fn js_value_new(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let constructor = sp.read_js();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_ids(args_ptr, args_len);

    let mut js_env = WasmerJsEnv::new(&mut sp, &mut env.exports, &mut env.go_state)?;
    let result = env.js_state.value_new(&mut js_env, constructor, &args);
    sp.write_call_result(result, || "constructor call".into())
}

/// go side: λ(v value, args []value) (value, bool)
pub fn js_value_invoke(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let object = sp.read_js();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_ids(args_ptr, args_len);

    let mut js_env = WasmerJsEnv::new(&mut sp, &mut env.exports, &mut env.go_state)?;
    let result = env.js_state.value_invoke(&mut js_env, object, &args);
    sp.write_call_result(result, || "invocation".into())
}

/// go side: λ(v value, method string, args []value) (value, bool)
pub fn js_value_call(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let object = sp.read_js();
    let method = sp.read_string();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_ids(args_ptr, args_len);

    let mut js_env = WasmerJsEnv::new(&mut sp, &mut env.exports, &mut env.go_state)?;
    let result = env.js_state.value_call(&mut js_env, object, &method, &args);
    sp.write_call_result(result, || format!("method call to {method}"))
}

/// go side: λ(v string) value
pub fn js_string_val(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let data = sp.read_string();
    let value = env.js_state.string_val(data);
    sp.write_js(value);
}

/// go side: λ(v value) int
pub fn js_value_length(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let source = sp.read_js();
    let length = env.js_state.value_length(source);

    sp.write_u64(length as u64);
}

/// go side: λ(str value) (array value, len int)
pub fn js_value_prepare_string(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let text = sp.read_js();

    let (data, len) = env.js_state.value_prepare_string(text);
    sp.write_js(data);
    sp.write_u64(len);
}

/// go side: λ(str value, dest []byte)
pub fn js_value_load_string(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let text = sp.read_js();
    let (dest_ptr, dest_len) = sp.read_go_slice();

    let write_bytes = |buf: &[_]| {
        let src_len = buf.len() as u64;
        if src_len != dest_len {
            eprintln!("Go copying bytes from JS src length {src_len} to Go dest length {dest_len}");
        }
        let len = src_len.min(dest_len) as usize;
        sp.write_slice(dest_ptr, &buf[..len]);
        len
    };
    if let Err(error) = env.js_state.copy_bytes_to_go(text, write_bytes) {
        eprintln!("failed to load string: {error:?}");
    }
}

/// go side: λ(dest []byte, src value) (int, bool)
pub fn js_copy_bytes_to_go(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let (dest_ptr, dest_len) = sp.read_go_slice();
    let src_val = sp.read_js();

    let write_bytes = |buf: &[_]| {
        let src_len = buf.len() as u64;
        if src_len != dest_len {
            eprintln!("Go copying bytes from JS src length {src_len} to Go dest length {dest_len}");
        }
        let len = src_len.min(dest_len) as usize;
        sp.write_slice(dest_ptr, &buf[..len]);
        len
    };

    let len = env.js_state.copy_bytes_to_go(src_val, write_bytes);
    sp.write_u64(len.as_ref().map(|x| *x).unwrap_or_default());
    sp.write_u8(len.map(|_| 1).unwrap_or_default());
}

/// go side: λ(dest value, src []byte) (int, bool)
pub fn js_copy_bytes_to_js(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let dest_val = sp.read_js();
    let (src_ptr, src_len) = sp.read_go_slice();

    let write_bytes = |buf: &mut [_]| {
        let dest_len = buf.len() as u64;
        if buf.len() as u64 != src_len {
            eprintln!("Go copying bytes from Go src length {src_len} to JS dest length {dest_len}");
        }
        let len = src_len.min(dest_len) as usize;

        // Slightly inefficient as this allocates a new temporary buffer
        let data = sp.read_slice(src_ptr, len as u64);
        buf[..len].copy_from_slice(&data);
        len
    };

    let len = env.js_state.copy_bytes_to_js(dest_val, write_bytes);
    sp.write_u64(len.as_ref().map(|x| *x).unwrap_or_default());
    sp.write_u8(len.map(|_| 1).unwrap_or_default());
}

/// go side: λ() u64
pub fn debug_pool_hash(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    sp.write_u64(env.js_state.pool_hash());
}

macro_rules! reject {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub fn $f(_: WasmEnvMut, _: u32) {
                unimplemented!("Go JS interface {} not supported", stringify!($f));
            }
        )*
    }
}

reject!(js_value_delete, js_value_instance_of);
