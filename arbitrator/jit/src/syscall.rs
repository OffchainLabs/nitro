// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
};

use arbutil::Color;
use rand::RngCore;
use wasmer::AsStoreMut;

use std::{collections::BTreeMap, io::Write};

const ZERO_ID: u32 = 1;
const NULL_ID: u32 = 2;
const GLOBAL_ID: u32 = 5;
const GO_ID: u32 = 6;

const OBJECT_ID: u32 = 100;
const ARRAY_ID: u32 = 101;
const PROCESS_ID: u32 = 102;
const FS_ID: u32 = 103;
const UINT8_ARRAY_ID: u32 = 104;
const CRYPTO_ID: u32 = 105;
const DATE_ID: u32 = 106;

const FS_CONSTANTS_ID: u32 = 200;

const DYNAMIC_OBJECT_ID_BASE: u32 = 10000;

#[derive(Default)]
pub struct JsRuntimeState {
    /// A collection of js objects
    pool: DynamicObjectPool,
    /// The event Go will execute next
    pub pending_event: Option<PendingEvent>,
}

#[derive(Clone, Default, Debug)]
struct DynamicObjectPool {
    objects: BTreeMap<u32, DynamicObject>,
    free_ids: Vec<u32>,
}

impl DynamicObjectPool {
    fn insert(&mut self, object: DynamicObject) -> u32 {
        let id = self
            .free_ids
            .pop()
            .unwrap_or(DYNAMIC_OBJECT_ID_BASE + self.objects.len() as u32);
        self.objects.insert(id, object);
        id
    }

    fn get(&self, id: u32) -> Option<&DynamicObject> {
        self.objects.get(&id)
    }

    fn get_mut(&mut self, id: u32) -> Option<&mut DynamicObject> {
        self.objects.get_mut(&id)
    }

    fn remove(&mut self, id: u32) -> Option<DynamicObject> {
        let res = self.objects.remove(&id);
        if res.is_some() {
            self.free_ids.push(id);
        }
        res
    }
}

#[derive(Debug, Clone)]
enum DynamicObject {
    Uint8Array(Vec<u8>),
    FunctionWrapper(JsValue, JsValue),
    PendingEvent(PendingEvent),
    ValueArray(Vec<GoValue>),
    Date,
}

#[derive(Clone, Debug)]
pub struct PendingEvent {
    pub id: JsValue,
    pub this: JsValue,
    pub args: Vec<GoValue>,
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum JsValue {
    Undefined,
    Number(f64),
    Ref(u32),
}

impl JsValue {
    fn assume_num_or_object(self) -> GoValue {
        match self {
            JsValue::Undefined => GoValue::Undefined,
            JsValue::Number(x) => GoValue::Number(x),
            JsValue::Ref(x) => GoValue::Object(x),
        }
    }

    /// Creates a JS runtime value from its native 64-bit floating point representation.
    /// The JS runtime stores handles to references in the NaN bits.
    /// Native 0 is the value called "undefined", and actual 0 is a special-cased NaN.
    /// Anything else that's not a NaN is the Number class.
    pub fn new(repr: u64) -> Self {
        if repr == 0 {
            return Self::Undefined;
        }
        let float = f64::from_bits(repr);
        if float.is_nan() && repr != f64::NAN.to_bits() {
            let id = repr as u32;
            if id == ZERO_ID {
                return Self::Number(0.);
            }
            return Self::Ref(id);
        }
        Self::Number(float)
    }
}

#[derive(Clone, Copy, Debug)]
#[allow(dead_code)]
pub enum GoValue {
    Undefined,
    Number(f64),
    Null,
    Object(u32),
    String(u32),
    Symbol(u32),
    Function(u32),
}

impl GoValue {
    fn encode(self) -> u64 {
        let (ty, id): (u32, u32) = match self {
            GoValue::Undefined => return 0,
            GoValue::Number(mut f) => {
                // Canonicalize NaNs so they don't collide with other value types
                if f.is_nan() {
                    f = f64::NAN;
                }
                if f == 0. {
                    // Zeroes are encoded differently for some reason
                    (0, ZERO_ID)
                } else {
                    return f.to_bits();
                }
            }
            GoValue::Null => (0, NULL_ID),
            GoValue::Object(x) => (1, x),
            GoValue::String(x) => (2, x),
            GoValue::Symbol(x) => (3, x),
            GoValue::Function(x) => (4, x),
        };
        // Must not be all zeroes, otherwise it'd collide with a real NaN
        assert!(ty != 0 || id != 0, "GoValue must not be empty");
        f64::NAN.to_bits() | (u64::from(ty) << 32) | u64::from(id)
    }
}

fn get_field(env: &mut WasmEnv, source: u32, field: &[u8]) -> GoValue {
    use DynamicObject::*;

    if let Some(source) = env.js_state.pool.get(source) {
        return match (source, field) {
            (PendingEvent(event), b"id" | b"this") => event.id.assume_num_or_object(),
            (PendingEvent(event), b"args") => {
                let args = ValueArray(event.args.clone());
                let id = env.js_state.pool.insert(args);
                GoValue::Object(id)
            }
            _ => {
                let field = String::from_utf8_lossy(field);
                eprintln!(
                    "Go trying to access unimplemented unknown JS value {:?} field {field}",
                    source
                );
                GoValue::Undefined
            }
        };
    }

    match (source, field) {
        (GLOBAL_ID, b"Object") => GoValue::Function(OBJECT_ID),
        (GLOBAL_ID, b"Array") => GoValue::Function(ARRAY_ID),
        (GLOBAL_ID, b"process") => GoValue::Object(PROCESS_ID),
        (GLOBAL_ID, b"fs") => GoValue::Object(FS_ID),
        (GLOBAL_ID, b"Uint8Array") => GoValue::Function(UINT8_ARRAY_ID),
        (GLOBAL_ID, b"crypto") => GoValue::Object(CRYPTO_ID),
        (GLOBAL_ID, b"Date") => GoValue::Object(DATE_ID),
        (GLOBAL_ID, b"fetch") => GoValue::Undefined, // Triggers a code path in Go for a fake network impl
        (FS_ID, b"constants") => GoValue::Object(FS_CONSTANTS_ID),
        (
            FS_CONSTANTS_ID,
            b"O_WRONLY" | b"O_RDWR" | b"O_CREAT" | b"O_TRUNC" | b"O_APPEND" | b"O_EXCL",
        ) => GoValue::Number(-1.),
        (GO_ID, b"_pendingEvent") => match &mut env.js_state.pending_event {
            Some(event) => {
                let event = PendingEvent(event.clone());
                let id = env.js_state.pool.insert(event);
                GoValue::Object(id)
            }
            None => GoValue::Null,
        },
        (PROCESS_ID, b"pid") => GoValue::Number(1.),
        _ => {
            let field = String::from_utf8_lossy(field);
            eprintln!("Go trying to access unimplemented unknown JS value {source} field {field}");
            GoValue::Undefined
        }
    }
}

pub fn js_finalize_ref(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    let pool = &mut env.js_state.pool;

    let val = JsValue::new(sp.read_u64(0));
    match val {
        JsValue::Ref(x) if x < DYNAMIC_OBJECT_ID_BASE => {}
        JsValue::Ref(x) => {
            if pool.remove(x).is_none() {
                eprintln!("Go trying to finalize unknown ref {}", x);
            }
        }
        val => eprintln!("Go trying to finalize {:?}", val),
    }
}

pub fn js_value_get(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    let source = JsValue::new(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let field = sp.read_slice(field_ptr, field_len);
    let value = match source {
        JsValue::Ref(id) => get_field(env, id, &field),
        val => {
            let field = String::from_utf8_lossy(&field);
            eprintln!("Go trying to read field {:?} . {field}", val);
            GoValue::Null
        }
    };
    sp.write_u64(3, value.encode());
}

pub fn js_value_set(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    use JsValue::*;

    let source = JsValue::new(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let new_value = JsValue::new(sp.read_u64(3));
    let field = sp.read_slice(field_ptr, field_len);
    if source == Ref(GO_ID) && &field == b"_pendingEvent" && new_value == Ref(NULL_ID) {
        env.js_state.pending_event = None;
        return;
    }
    if let Ref(id) = source {
        let source = env.js_state.pool.get(id);
        if let Some(DynamicObject::PendingEvent(_)) = source {
            if field == b"result" {
                return;
            }
        }
    }
    let field = String::from_utf8_lossy(&field);
    eprintln!(
        "Go attempted to set unsupported value {:?} field {field} to {:?}",
        source, new_value,
    );
}

pub fn js_value_index(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            return sp.write_u64(2, GoValue::Null.encode());
        }};
    }

    let source = match JsValue::new(sp.read_u64(0)) {
        JsValue::Ref(x) => env.js_state.pool.get(x),
        val => fail!("Go attempted to index into {:?}", val),
    };
    let index = match u32::try_from(sp.read_u64(1)) {
        Ok(index) => index as usize,
        Err(err) => fail!("{:?}", err),
    };
    let value = match source {
        Some(DynamicObject::Uint8Array(x)) => x.get(index).map(|x| GoValue::Number(*x as f64)),
        Some(DynamicObject::ValueArray(x)) => x.get(index).cloned(),
        _ => fail!("Go attempted to index into unsupported value {:?}", source),
    };
    let Some(value) = value else {
        fail!("Go indexing out of bounds into {:?} index {index}", source)
    };
    sp.write_u64(2, value.encode());
}

pub fn js_value_call(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let Some(resume) = env.data().exports.resume.clone() else {
        return Escape::failure(format!("wasmer failed to bind {}", "resume".red()));
    };
    let Some(get_stack_pointer) = env.data().exports.get_stack_pointer.clone() else {
        return Escape::failure(format!("wasmer failed to bind {}", "getsp".red()));
    };
    let sp = GoStack::simple(sp, &env);
    let data = env.data_mut();
    let rng = &mut data.go_state.rng;
    let pool = &mut data.js_state.pool;
    use JsValue::*;

    let object = JsValue::new(sp.read_u64(0));
    let method_name_ptr = sp.read_u64(1);
    let method_name_len = sp.read_u64(2);
    let method_name = sp.read_slice(method_name_ptr, method_name_len);
    let args_ptr = sp.read_u64(3);
    let args_len = sp.read_u64(4);
    let args = sp.read_value_slice(args_ptr, args_len);
    let name = String::from_utf8_lossy(&method_name);

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            sp.write_u64(6, GoValue::Null.encode());
            sp.write_u8(7, 1);
            return Ok(())
        }};
    }

    let value = match (object, method_name.as_slice()) {
        (Ref(GO_ID), b"_makeFuncWrapper") => {
            let arg = match args.first() {
                Some(arg) => arg,
                None => fail!(
                    "Go trying to call Go._makeFuncWrapper with bad args {:?}",
                    args
                ),
            };
            let ref_id = pool.insert(DynamicObject::FunctionWrapper(*arg, object));
            GoValue::Function(ref_id)
        }
        (Ref(FS_ID), b"write") => {
            // ignore any args after the 6th, and slice no more than than the number of args we have
            let args_len = std::cmp::min(6, args.len());

            match &args.as_slice()[..args_len] {
                &[Number(fd), Ref(buf_id), Number(offset), Number(length), Ref(NULL_ID), Ref(callback_id)] =>
                {
                    let buf = match pool.get(buf_id) {
                        Some(DynamicObject::Uint8Array(x)) => x,
                        x => fail!("Go trying to call fs.write with bad buffer {:?}", x),
                    };
                    let (func_id, this) = match pool.get(callback_id) {
                        Some(DynamicObject::FunctionWrapper(f, t)) => (f, t),
                        x => fail!("Go trying to call fs.write with bad buffer {:?}", x),
                    };

                    let mut offset = offset as usize;
                    let mut length = length as usize;
                    if offset > buf.len() {
                        eprintln!(
                            "Go trying to call fs.write with offset {offset} >= buf.len() {length}"
                        );
                        offset = buf.len();
                    }
                    if offset + length > buf.len() {
                        eprintln!(
                            "Go trying to call fs.write with offset {offset} + length {length} >= buf.len() {}",
                            buf.len(),
                        );
                        length = buf.len() - offset;
                    }
                    if fd == 1. {
                        let stdout = std::io::stdout();
                        let mut stdout = stdout.lock();
                        stdout.write_all(&buf[offset..(offset + length)]).unwrap();
                    } else if fd == 2. {
                        let stderr = std::io::stderr();
                        let mut stderr = stderr.lock();
                        stderr.write_all(&buf[offset..(offset + length)]).unwrap();
                    } else {
                        eprintln!("Go trying to write to unknown FD {}", fd);
                    }

                    data.js_state.pending_event = Some(PendingEvent {
                        id: *func_id,
                        this: *this,
                        args: vec![
                            GoValue::Null,                  // no error
                            GoValue::Number(length as f64), // amount written
                        ],
                    });

                    // recursively call into wasmer
                    let mut store = env.as_store_mut();
                    resume.call(&mut store)?;

                    // the stack pointer has changed, so we'll need to write our return results elsewhere
                    let pointer = get_stack_pointer.call(&mut store)? as u32;
                    sp.write_u64_ptr(pointer + sp.relative_offset(6), GoValue::Null.encode());
                    sp.write_u8_ptr(pointer + sp.relative_offset(7), 1);
                    return Ok(());
                }
                _ => fail!("Go trying to call fs.write with bad args {:?}", args),
            }
        }
        (Ref(CRYPTO_ID), b"getRandomValues") => {
            let name = "crypto.getRandomValues";

            let id = match args.first() {
                Some(Ref(x)) => x,
                _ => fail!("Go trying to call {name} with bad args {:?}", args),
            };

            let buf = match pool.get_mut(*id) {
                Some(DynamicObject::Uint8Array(buf)) => buf,
                Some(x) => fail!("Go trying to call {name} on bad object {:?}", x),
                None => fail!("Go trying to call {name} on unknown reference {id}"),
            };

            rng.fill_bytes(buf.as_mut_slice());
            GoValue::Undefined
        }
        (Ref(obj_id), _) => {
            let value = match pool.get(obj_id) {
                Some(value) => value,
                None => fail!("Go trying to call method {name} for unknown object - id {obj_id}"),
            };
            match value {
                DynamicObject::Date => GoValue::Number(0.0),
                _ => fail!("Go trying to call unknown method {name} for date object"),
            }
        }
        _ => fail!("Go trying to call unknown method {:?} . {name}", object),
    };

    sp.write_u64(6, value.encode());
    sp.write_u8(7, 1);
    Ok(())
}

pub fn js_value_new(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    let pool = &mut env.js_state.pool;

    let class = sp.read_u32(0);
    let args_ptr = sp.read_u64(1);
    let args_len = sp.read_u64(2);
    let args = sp.read_value_slice(args_ptr, args_len);
    match class {
        UINT8_ARRAY_ID => match args.first() {
            Some(JsValue::Number(size)) => {
                let id = pool.insert(DynamicObject::Uint8Array(vec![0; *size as usize]));
                sp.write_u64(4, GoValue::Object(id).encode());
                sp.write_u8(5, 1);
                return;
            }
            _ => eprintln!(
                "Go attempted to construct Uint8Array with bad args: {:?}",
                args,
            ),
        },
        DATE_ID => {
            let id = pool.insert(DynamicObject::Date);
            sp.write_u64(4, GoValue::Object(id).encode());
            sp.write_u8(5, 1);
            return;
        }
        _ => eprintln!("Go trying to construct unimplemented JS value {class}"),
    }
    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

pub fn js_value_length(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);

    let source = match JsValue::new(sp.read_u64(0)) {
        JsValue::Ref(x) => env.js_state.pool.get(x),
        _ => None,
    };
    let length = match source {
        Some(DynamicObject::Uint8Array(x)) => x.len(),
        Some(DynamicObject::ValueArray(x)) => x.len(),
        _ => {
            eprintln!(
                "Go attempted to get length of unsupported value {:?}",
                source,
            );
            0
        }
    };
    sp.write_u64(1, length as u64);
}

pub fn js_copy_bytes_to_go(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    let dest_ptr = sp.read_u64(0);
    let dest_len = sp.read_u64(1);
    let src_val = JsValue::new(sp.read_u64(3));

    match src_val {
        JsValue::Ref(src_id) => match env.js_state.pool.get_mut(src_id) {
            Some(DynamicObject::Uint8Array(buf)) => {
                let src_len = buf.len() as u64;
                if src_len != dest_len {
                    eprintln!(
                        "Go copying bytes from JS source length {src_len} to Go dest length {dest_len}",
                    );
                }
                let len = std::cmp::min(src_len, dest_len) as usize;
                sp.write_slice(dest_ptr, &buf[..len]);
                sp.write_u64(4, GoValue::Number(len as f64).encode());
                sp.write_u8(5, 1);
                return;
            }
            source => {
                eprintln!(
                    "Go trying to copy bytes from unsupported source {:?}",
                    source,
                );
            }
        },
        _ => eprintln!("Go trying to copy bytes from {:?}", src_val),
    }

    sp.write_u8(5, 0);
}

pub fn js_copy_bytes_to_js(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);

    match JsValue::new(sp.read_u64(0)) {
        JsValue::Ref(dest_id) => {
            let src_ptr = sp.read_u64(1);
            let src_len = sp.read_u64(2);

            match env.js_state.pool.get_mut(dest_id) {
                Some(DynamicObject::Uint8Array(buf)) => {
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
                    sp.write_u64(4, GoValue::Number(len as f64).encode());
                    sp.write_u8(5, 1);
                    return;
                }
                dest => eprintln!("Go trying to copy bytes into unsupported target {:?}", dest),
            }
        }
        value => eprintln!("Go trying to copy bytes into {:?}", value),
    }

    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

macro_rules! unimpl_js {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub fn $f(_: WasmEnvMut, _: u32) {
                unimplemented!("Go JS interface {} not supported", stringify!($f));
            }
        )*
    }
}

unimpl_js!(
    js_string_val,
    js_value_set_index,
    js_value_prepare_string,
    js_value_load_string,
    js_value_delete,
    js_value_invoke,
    js_value_instance_of,
);
