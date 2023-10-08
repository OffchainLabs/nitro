// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnvMut},
};

use arbutil::Color;
use go_js::{JsEnv, JsValueId};
use wasmer::{StoreMut, TypedFunction};

/// go side: λ(v value)
// TODO: implement ref counting
pub fn js_finalize_ref(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let val = JsValueId(sp.read_u64());
    env.js_state.finalize_ref(val);
}

/// go side: λ(v value, field string) value
pub fn js_value_get(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = JsValueId(sp.read_u64());
    let field = sp.read_string();

    let result = env.js_state.value_get(source, &field);

    sp.write_u64(result.0);
}

/// go side: λ(v value, field string, x value)
pub fn js_value_set(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = JsValueId(sp.read_u64());
    let field = sp.read_string();
    let new_value = JsValueId(sp.read_u64());

    env.js_state.value_set(source, &field, new_value);
}

/// go side: λ(v value, i int) value
pub fn js_value_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = JsValueId(sp.read_u64());
    let index = sp.read_go_ptr() as usize;

    let result = env.js_state.value_index(source, index);

    sp.write_u64(result.0);
}

/// go side: λ(array value, i int, v value)
pub fn js_value_set_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = JsValueId(sp.read_u64());
    let index = sp.read_go_ptr() as usize;
    let value = JsValueId(sp.read_u64());

    env.js_state.value_set_index(source, index, value);
}

struct WasmerJsEnv<'a, 'b> {
    rng: &'a mut rand_pcg::Pcg32,
    resume: &'a TypedFunction<(), ()>,
    store: &'a mut StoreMut<'b>,
}

impl<'a, 'b> JsEnv for WasmerJsEnv<'a, 'b> {
    fn get_rng(&mut self) -> &mut dyn rand::RngCore {
        &mut self.rng
    }

    fn resume(&mut self) -> eyre::Result<()> {
        self.resume.call(self.store)?;
        Ok(())
    }
}

/// go side: λ(v value, method string, args []value) (value, bool)
pub fn js_value_call(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env, mut store) = GoStack::new_with_store(sp, &mut env);

    let object = JsValueId(sp.read_u64());
    let method_name = sp.read_string();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_ids(args_ptr, args_len);

    let Some(resume) = &env.exports.resume else {
        return Escape::failure(format!("wasmer failed to bind {}", "resume".red()));
    };
    let mut js_env = WasmerJsEnv {
        rng: &mut env.go_state.rng,
        resume: resume,
        store: &mut store,
    };

    let result = env
        .js_state
        .value_call(&mut js_env, object, &method_name, &args);
    unsafe {
        sp.refresh(env, &mut store)?;
    }
    match result {
        Ok(result) => {
            sp.write_u64(result.0);
            sp.write_u8(1);
        }
        Err(err) => match err.downcast::<Escape>() {
            Ok(escape) => {
                return Err(escape);
            }
            Err(err) => {
                eprintln!("Go method call to {method_name} failed with error {err:#}");
                sp.write_u64(go_js::get_null().0);
                sp.write_u8(0);
            }
        },
    }

    Ok(())
}

/// go side: λ(v value, args []value) (value, bool)
pub fn js_value_new(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env, mut store) = GoStack::new_with_store(sp, &mut env);

    let Some(resume) = &env.exports.resume else {
        return Escape::failure(format!("wasmer failed to bind {}", "resume".red()));
    };
    let mut js_env = WasmerJsEnv {
        rng: &mut env.go_state.rng,
        resume: resume,
        store: &mut store,
    };

    let constructor = JsValueId(sp.read_u64());
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_ids(args_ptr, args_len);

    let result = env.js_state.value_new(&mut js_env, constructor, &args);
    unsafe {
        sp.refresh(env, &mut store)?;
    }
    match result {
        Ok(result) => {
            sp.write_u64(result.0);
            sp.write_u8(1);
        }
        Err(err) => match err.downcast::<Escape>() {
            Ok(escape) => {
                return Err(escape);
            }
            Err(err) => {
                eprintln!("Go constructor call failed with error {err:#}");
                sp.write_u64(go_js::get_null().0);
                sp.write_u8(0);
            }
        },
    }

    Ok(())
}

/// go side: λ(v string) value
pub fn js_string_val(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let data = sp.read_string();
    let value = env.js_state.string_val(data);
    sp.write_u64(value.0);
}

/// go side: λ(v value) int
pub fn js_value_length(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let source = JsValueId(sp.read_u64());
    let length = env.js_state.value_length(source);

    sp.write_u64(length as u64);
}

/// go side: λ(dest []byte, src value) (int, bool)
pub fn js_copy_bytes_to_go(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let (dest_ptr, dest_len) = sp.read_go_slice();
    let src_val = JsValueId(sp.read_u64());

    env.js_state.copy_bytes_to_go(src_val, |buf| {
        let src_len = buf.len() as u64;
        if src_len != dest_len {
            eprintln!(
                "Go copying bytes from JS source length {src_len} to Go dest length {dest_len}",
            );
        }
        let len = std::cmp::min(src_len, dest_len) as usize;
        sp.write_slice(dest_ptr, &buf[..len]);
        sp.write_u64(go_js::get_number(len as f64).0);
        sp.write_u8(1);
    });
}

/// go side: λ(dest value, src []byte) (int, bool)
pub fn js_copy_bytes_to_js(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let dest_val = JsValueId(sp.read_u64());
    let (src_ptr, src_len) = sp.read_go_slice();

    env.js_state.copy_bytes_to_js(dest_val, |buf| {
        let dest_len = buf.len() as u64;
        if buf.len() as u64 != src_len {
            eprintln!(
                "Go copying bytes from Go source length {src_len} to JS dest length {dest_len}",
            );
        }
        let len = std::cmp::min(src_len, dest_len) as usize;

        // Slightly inefficient as this allocates a new temporary buffer
        let data = sp.read_slice(src_ptr, len as u64);
        buf[..len].copy_from_slice(&data);
        sp.write_u64(go_js::get_number(len as f64).0);
        sp.write_u8(1);
    });
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

reject!(
    js_value_prepare_string,
    js_value_load_string,
    js_value_delete,
    js_value_invoke,
    js_value_instance_of,
);
