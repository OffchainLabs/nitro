// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{GoStack, WasmEnv, WasmEnvArc};

use parking_lot::MutexGuard;

use std::collections::BTreeMap;

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

#[derive(Clone, Default, Debug)]
pub struct DynamicObjectPool {
    objects: BTreeMap<u32, DynamicObject>,
    free_ids: Vec<u32>,
}

impl DynamicObjectPool {
    pub fn insert(&mut self, object: DynamicObject) -> u32 {
        let id = self
            .free_ids
            .pop()
            .unwrap_or_else(|| DYNAMIC_OBJECT_ID_BASE + self.objects.len() as u32);
        self.objects.insert(id, object);
        id
    }

    pub fn get(&self, id: u32) -> Option<&DynamicObject> {
        self.objects.get(&id)
    }

    pub fn get_mut(&mut self, id: u32) -> Option<&mut DynamicObject> {
        self.objects.get_mut(&id)
    }

    pub fn remove(&mut self, id: u32) -> Option<DynamicObject> {
        let res = self.objects.remove(&id);
        if res.is_some() {
            self.free_ids.push(id);
        }
        res
    }
}

#[derive(Debug, Clone)]
pub enum DynamicObject {
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
    pub fn assume_num_or_object(self) -> GoValue {
        match self {
            JsValue::Undefined => GoValue::Undefined,
            JsValue::Number(x) => GoValue::Number(x),
            JsValue::Ref(x) => GoValue::Object(x),
        }
    }
}

pub fn js_value(repr: u64) -> JsValue {
    if repr == 0 {
        return JsValue::Undefined;
    }
    let float = f64::from_bits(repr);
    if float.is_nan() && repr != f64::NAN.to_bits() {
        let id = repr as u32;
        if id == ZERO_ID {
            return JsValue::Number(0.);
        }
        return JsValue::Ref(id);
    }
    JsValue::Number(float)
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
    pub fn encode(self) -> u64 {
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

fn get_field(env: &mut MutexGuard<WasmEnv>, source: u32, field: &[u8]) -> GoValue {
    use DynamicObject::*;

    if let Some(source) = env.js_object_pool.get(source) {
        return match (source, field) {
            (PendingEvent(event), b"id" | b"this") => event.id.assume_num_or_object(),
            (PendingEvent(event), b"args") => {
                let args = ValueArray(event.args.clone());
                let id = env.js_object_pool.insert(args);
                GoValue::Object(id)
            }
            _ => {
                let field = String::from_utf8_lossy(field);
                eprintln!(
                    "Go attempting to access unimplemented unknown JS value {:?} field {field}",
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
        (GO_ID, b"_pendingEvent") => match &mut env.js_pending_event {
            Some(event) => {
                let event = PendingEvent(event.clone());
                let id = env.js_object_pool.insert(event);
                GoValue::Object(id)
            }
            None => GoValue::Null,
        },
        _ => {
            let field = String::from_utf8_lossy(field);
            eprintln!(
                "Go attempting to access unimplemented unknown JS value {source} field {field}",
            );
            GoValue::Undefined
        }
    }
}

pub fn js_finalize_ref(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
    let pool = &mut env.js_object_pool;

    let val = js_value(sp.read_u64(0));
    match val {
        JsValue::Ref(x) if x < DYNAMIC_OBJECT_ID_BASE => {}
        JsValue::Ref(x) => {
            if pool.remove(x).is_none() {
                eprintln!("Go attempting to finalize unknown ref {}", x);
            }
        }
        val => eprintln!("Go attempting to finalize {:?}", val),
    }
}

pub fn js_value_get(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
    let source = js_value(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let field = sp.read_slice(field_ptr, field_len.into());
    let value = match source {
        JsValue::Ref(id) => get_field(&mut env, id, &field),
        val => {
            let field = String::from_utf8_lossy(&field);
            eprintln!("Go attempting to read field {:?} . {field}", val);
            GoValue::Null
        }
    };
    sp.write_u64(3, value.encode());
}

pub fn js_value_set(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
    use JsValue::*;

    let source = js_value(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let new_value = js_value(sp.read_u64(3));
    let field = sp.read_slice(field_ptr, field_len);
    if source == Ref(GO_ID) && &field == b"_pendingEvent" && new_value == Ref(NULL_ID) {
        env.js_pending_event = None;
        return;
    }
    if let Ref(id) = source {
        let source = env.js_object_pool.get(id);
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

pub fn js_value_index(env: &WasmEnvArc, sp: u32) {
    let (sp, env) = GoStack::new(sp, env);

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            return sp.write_u64(2, GoValue::Null.encode());
        }};
    }

    let source = match js_value(sp.read_u64(0)) {
        JsValue::Ref(x) => env.js_object_pool.get(x),
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
    let value = match value {
        Some(value) => value,
        None => fail!("Go indexing out of bounds into {:?} index {index}", source),
    };
    sp.write_u64(2, value.encode());
}

pub fn js_value_call(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
}

pub fn js_value_new(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
    let pool = &mut env.js_object_pool;

    let class = sp.read_u32(0);
    let args_ptr = sp.read_u64(1);
    let args_len = sp.read_u64(2);
    let args = sp.read_value_slice(args_ptr, args_len);
    match class {
        UINT8_ARRAY_ID => match args.get(0) {
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
        _ => eprintln!("Go attempting to construct unimplemented JS value {class}"),
    }
    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

pub fn js_value_length(env: &WasmEnvArc, sp: u32) {
    let (sp, env) = GoStack::new(sp, env);

    let source = match js_value(sp.read_u64(0)) {
        JsValue::Ref(x) => env.js_object_pool.get(x),
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

pub fn js_copy_bytes_to_go(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);
    let dest_ptr = sp.read_u64(0);
    let dest_len = sp.read_u64(1);
    let src_val = js_value(sp.read_u64(3));

    match src_val {
        JsValue::Ref(src_id) => match env.js_object_pool.get_mut(src_id) {
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
                    "Go attempting to copy bytes from unsupported source {:?}",
                    source,
                );
            }
        },
        _ => eprintln!("Go attempting to copy bytes from {:?}", src_val),
    }

    sp.write_u8(5, 0);
}

pub fn js_copy_bytes_to_js(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, env);

    match js_value(sp.read_u64(0)) {
        JsValue::Ref(dest_id) => {
            let src_ptr = sp.read_u64(1);
            let src_len = sp.read_u64(2);

            match env.js_object_pool.get_mut(dest_id) {
                Some(DynamicObject::Uint8Array(buf)) => {
                    let dest_len = buf.len() as u64;
                    if buf.len() as u64 != src_len {
                        eprintln!(
                            "Go copying bytes from Go source length {src_len} to JS dest length {dest_len}",
                        );
                    }
                    let len = std::cmp::min(src_len, dest_len) as usize;

                    // Slightly inefficient as this allocates a new temporary buffer
                    buf[..len].copy_from_slice(&sp.read_slice(src_ptr, len as u64));
                    sp.write_u64(4, GoValue::Number(len as f64).encode());
                    sp.write_u8(5, 1);
                    return;
                }
                dest => eprintln!(
                    "Go attempting to copy bytes into unsupported target {:?}",
                    dest,
                ),
            }
        }
        value => eprintln!("Go attempting to copy bytes into {:?}", value),
    }

    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

macro_rules! unimpl_js {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub fn $f(_: &WasmEnvArc, _: u32) {
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
