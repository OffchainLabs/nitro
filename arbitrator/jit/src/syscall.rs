// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
};

use arbutil::{Color, DebugColor};
use rand::RngCore;

use std::{collections::BTreeMap, io::Write};

const ZERO_ID: u32 = 1;
const NULL_ID: u32 = 2;
const GLOBAL_ID: u32 = 5;
const GO_ID: u32 = 6;
pub const STYLUS_ID: u32 = 7;

const OBJECT_ID: u32 = 100;
const ARRAY_ID: u32 = 101;
const PROCESS_ID: u32 = 102;
const FS_ID: u32 = 103;
const UINT8_ARRAY_ID: u32 = 104;
const CRYPTO_ID: u32 = 105;
const DATE_ID: u32 = 106;
const CONSOLE_ID: u32 = 107;

const FS_CONSTANTS_ID: u32 = 200;

const DYNAMIC_OBJECT_ID_BASE: u32 = 10000;

fn standard_id_name(id: u32) -> Option<&'static str> {
    Some(match id {
        STYLUS_ID => "stylus",
        OBJECT_ID => "Object",
        ARRAY_ID => "Array",
        PROCESS_ID => "process",
        FS_ID => "fs",
        CRYPTO_ID => "crypto",
        DATE_ID => "Date",
        CONSOLE_ID => "console",
        _ => return None,
    })
}

#[derive(Default)]
pub struct JsRuntimeState {
    /// A collection of js objects
    pub pool: DynamicObjectPool,
    /// The event Go will execute next
    pub pending_event: Option<PendingEvent>,
    /// The stylus return result
    pub stylus_result: Option<JsValue>,
}

impl JsRuntimeState {
    pub fn set_pending_event(&mut self, id: u32, this: JsValue, args: Vec<GoValue>) {
        let id = JsValue::Number(id as f64);
        self.pending_event = Some(PendingEvent { id, this, args });
    }

    fn free(&mut self, value: GoValue) {
        use GoValue::*;
        match value {
            Object(id) => drop(self.pool.remove(id)),
            Undefined | Null | Number(_) => {}
            _ => unimplemented!(),
        }
    }
}

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
            .unwrap_or(DYNAMIC_OBJECT_ID_BASE + self.objects.len() as u32);
        self.objects.insert(id, object);
        id
    }

    pub fn get(&self, id: u32) -> Option<&DynamicObject> {
        self.objects.get(&id)
    }

    fn get_mut(&mut self, id: u32) -> Option<&mut DynamicObject> {
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
    GoString(Vec<u8>),
    FunctionWrapper(u32), // the func_id
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

    pub fn assume_id(self) -> Result<u32, Escape> {
        match self {
            GoValue::Object(id) => Ok(id),
            x => Escape::failure(format!("not an id: {}", x.debug_red())),
        }
    }
}

fn get_field(env: &mut WasmEnv, source: u32, field: &[u8]) -> GoValue {
    use DynamicObject::*;
    let js = &mut env.js_state;

    if let Some(source) = js.pool.get(source) {
        return match (source, field) {
            (PendingEvent(event), b"id") => event.id.assume_num_or_object(),
            (PendingEvent(event), b"this") => event.this.assume_num_or_object(),
            (PendingEvent(event), b"args") => {
                let args = ValueArray(event.args.clone());
                let id = env.js_state.pool.insert(args);
                GoValue::Object(id)
            }
            _ => {
                let field = String::from_utf8_lossy(field);
                eprintln!("Go trying to access unimplemented JS value {source:?} field {field}",);
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
        (GLOBAL_ID, b"console") => GoValue::Object(CONSOLE_ID),
        (GLOBAL_ID, b"fetch") => GoValue::Undefined, // Triggers a code path in Go for a fake network impl
        (FS_ID, b"constants") => GoValue::Object(FS_CONSTANTS_ID),
        (
            FS_CONSTANTS_ID,
            b"O_WRONLY" | b"O_RDWR" | b"O_CREAT" | b"O_TRUNC" | b"O_APPEND" | b"O_EXCL",
        ) => GoValue::Number(-1.),
        (GO_ID, b"_pendingEvent") => match &mut js.pending_event {
            Some(event) => {
                let event = PendingEvent(event.clone());
                let id = js.pool.insert(event);
                GoValue::Object(id)
            }
            None => GoValue::Null,
        },
        (GLOBAL_ID, b"stylus") => GoValue::Object(STYLUS_ID),
        (STYLUS_ID, b"result") => match &mut js.stylus_result {
            Some(value) => value.assume_num_or_object(), // TODO: reference count
            None => GoValue::Null,
        },
        _ => {
            let field = String::from_utf8_lossy(field);
            eprintln!("Go trying to access unimplemented unknown JS value {source} field {field}");
            GoValue::Undefined
        }
    }
}

/// go side: λ(v value)
pub fn js_finalize_ref(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let pool = &mut env.js_state.pool;

    let val = JsValue::new(sp.read_u64());
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

/// go side: λ(v value, field string) value
pub fn js_value_get(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let source = JsValue::new(sp.read_u64());
    let field = sp.read_js_string();

    let value = match source {
        JsValue::Ref(id) => get_field(env, id, &field),
        val => {
            let field = String::from_utf8_lossy(&field);
            eprintln!("Go trying to read field {:?} . {field}", val);
            GoValue::Null
        }
    };
    sp.write_u64(value.encode());
}

/// go side: λ(v value, field string, x value)
pub fn js_value_set(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    use JsValue::*;

    let source = JsValue::new(sp.read_u64());
    let field = sp.read_js_string();
    let new_value = JsValue::new(sp.read_u64());

    if source == Ref(GO_ID) && &field == b"_pendingEvent" && new_value == Ref(NULL_ID) {
        env.js_state.pending_event = None;
        return;
    }
    match (source, field.as_slice()) {
        (Ref(STYLUS_ID), b"result") => {
            env.js_state.stylus_result = Some(new_value);
            return;
        }
        _ => {}
    }

    if let Ref(id) = source {
        let source = env.js_state.pool.get_mut(id);
        if let Some(DynamicObject::PendingEvent(_)) = source {
            if field == b"result" {
                return;
            }
        }
    }
    let field = String::from_utf8_lossy(&field).red();
    eprintln!("Go attempted to set unsupported value {source:?} field {field} to {new_value:?}");
}

/// go side: λ(v value, i int) value
pub fn js_value_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            sp.write_u64(GoValue::Null.encode());
            return
        }};
    }

    let source = match JsValue::new(sp.read_u64()) {
        JsValue::Ref(x) => env.js_state.pool.get(x),
        val => fail!("Go attempted to index into {val:?}"),
    };
    let index = sp.read_go_ptr() as usize;
    let value = match source {
        Some(DynamicObject::Uint8Array(x)) => x.get(index).map(|x| GoValue::Number(*x as f64)),
        Some(DynamicObject::ValueArray(x)) => x.get(index).cloned(),
        _ => fail!("Go attempted to index into unsupported value {source:?}"),
    };
    let Some(value) = value else {
        fail!("Go indexing out of bounds into {source:?} index {index}")
    };
    sp.write_u64(value.encode());
}

/// go side: λ(array value, i int, v value)
pub fn js_value_set_index(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            return
        }};
    }

    let source = match JsValue::new(sp.read_u64()) {
        JsValue::Ref(x) => env.js_state.pool.get_mut(x),
        val => fail!("Go attempted to index into {val:?}"),
    };
    let index = sp.read_go_ptr() as usize;
    let value = JsValue::new(sp.read_u64()).assume_num_or_object();

    match source {
        Some(DynamicObject::ValueArray(vec)) => {
            if index >= vec.len() {
                vec.resize(index + 1, GoValue::Undefined);
            }
            let prior = std::mem::replace(&mut vec[index], value);
            env.js_state.free(prior);
        }
        _ => fail!("Go attempted to index into unsupported value {source:?} {index}"),
    }
}

/// go side: λ(v value, method string, args []value) (value, bool)
pub fn js_value_call(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (mut sp, env, mut store) = GoStack::new_with_store(sp, &mut env);
    let rng = &mut env.go_state.rng;
    let pool = &mut env.js_state.pool;

    let object = JsValue::new(sp.read_u64());
    let method_name = sp.read_js_string();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = sp.read_value_slice(args_ptr, args_len);
    let name = String::from_utf8_lossy(&method_name);
    use JsValue::*;

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            sp.write_u64(GoValue::Null.encode());
            sp.write_u8(1);
            return Ok(())
        }};
    }

    let value = match (object, method_name.as_slice()) {
        (Ref(GO_ID), b"_makeFuncWrapper") => {
            let Some(JsValue::Number(func_id)) = args.get(0) else {
                fail!("Go trying to call Go._makeFuncWrapper with bad args {args:?}")
            };
            let ref_id = pool.insert(DynamicObject::FunctionWrapper(*func_id as u32));
            GoValue::Function(ref_id)
        }
        (Ref(STYLUS_ID), b"setCallbacks") => {
            let mut ids = vec![];
            for arg in args {
                let Ref(id) = arg else {
                    fail!("Stylus callback not a function {arg:?}")
                };
                ids.push(GoValue::Number(id as f64));
            }
            let value = pool.insert(DynamicObject::ValueArray(ids));
            GoValue::Object(value)
        }
        (Ref(FS_ID), b"write") => {
            // ignore any args after the 6th, and slice no more than than the number of args we have
            let args_len = std::cmp::min(6, args.len());

            match &args.as_slice()[..args_len] {
                &[Number(fd), Ref(buf_id), Number(offset), Number(length), Ref(NULL_ID), Ref(callback_id)] =>
                {
                    let buf = match pool.get(buf_id) {
                        Some(DynamicObject::Uint8Array(x)) => x,
                        x => fail!("Go trying to call fs.write with bad buffer {x:?}"),
                    };
                    let &func_id = match pool.get(callback_id) {
                        Some(DynamicObject::FunctionWrapper(func_id)) => func_id,
                        x => fail!("Go trying to call fs.write with bad buffer {x:?}"),
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
                        eprintln!("Go trying to write to unknown FD {fd}");
                    }

                    env.js_state.set_pending_event(
                        func_id,
                        object,
                        vec![
                            GoValue::Null,                  // no error
                            GoValue::Number(length as f64), // amount written
                        ],
                    );

                    // SAFETY: only sp is live after this
                    unsafe { sp.resume(env, &mut store)? };
                    GoValue::Null
                }
                _ => fail!("Go trying to call fs.write with bad args {args:?}"),
            }
        }
        (Ref(CRYPTO_ID), b"getRandomValues") => {
            let name = "crypto.getRandomValues";

            let id = match args.get(0) {
                Some(Ref(x)) => x,
                _ => fail!("Go trying to call {name} with bad args {args:?}"),
            };

            let buf = match pool.get_mut(*id) {
                Some(DynamicObject::Uint8Array(buf)) => buf,
                Some(x) => fail!("Go trying to call {name} on bad object {x:?}"),
                None => fail!("Go trying to call {name} on unknown reference {id}"),
            };

            rng.fill_bytes(buf.as_mut_slice());
            GoValue::Undefined
        }
        (Ref(CONSOLE_ID), b"error") => {
            print!("{}", "console error:".red());
            for arg in args {
                match arg {
                    JsValue::Undefined => print!(" undefined"),
                    JsValue::Number(x) => print!(" num {x}"),
                    JsValue::Ref(id) => match pool.get(id) {
                        Some(DynamicObject::GoString(data)) => {
                            print!(" {}", String::from_utf8_lossy(&data))
                        }
                        Some(DynamicObject::Uint8Array(data)) => {
                            print!(" 0x{}", hex::encode(data))
                        }
                        Some(other) => print!(" {other:?}"),
                        None => print!(" unknown"),
                    },
                }
            }
            println!("");
            GoValue::Undefined
        }
        (Ref(obj_id), _) => {
            let obj_name = standard_id_name(obj_id).unwrap_or("unknown object").red();
            let value = match pool.get(obj_id) {
                Some(value) => value,
                None => fail!("Go trying to call method {name} for {obj_name} - id {obj_id}"),
            };
            match value {
                DynamicObject::Date => GoValue::Number(0.0),
                _ => fail!("Go trying to call unknown method {name} for date object"),
            }
        }
        _ => fail!("Go trying to call unknown method {object:?} . {name}"),
    };

    sp.write_u64(value.encode());
    sp.write_u8(1);
    Ok(())
}

/// go side: λ(v value, args []value) (value, bool)
pub fn js_value_new(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let pool = &mut env.js_state.pool;

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            sp.write_u64(GoValue::Null.encode());
            sp.write_u8(0);
            return
        }};
    }

    let class = sp.read_u32();
    let (args_ptr, args_len) = sp.skip_space().read_go_slice();
    let args = sp.read_value_slice(args_ptr, args_len);
    let value = match class {
        UINT8_ARRAY_ID => match args.get(0) {
            Some(JsValue::Number(size)) => DynamicObject::Uint8Array(vec![0; *size as usize]),
            _ => fail!("Go attempted to construct Uint8Array with bad args: {args:?}"),
        },
        DATE_ID => DynamicObject::Date,
        ARRAY_ID => {
            // Note: assumes values are only numbers and objects
            let values = args
                .into_iter()
                .map(JsValue::assume_num_or_object)
                .collect();
            DynamicObject::ValueArray(values)
        }
        _ => fail!("Go trying to construct unimplemented JS value {class}"),
    };
    let id = pool.insert(value);
    sp.write_u64(GoValue::Object(id).encode());
    sp.write_u8(1);
}

/// go side: λ(v string) value
pub fn js_string_val(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let data = sp.read_js_string();
    let id = env.js_state.pool.insert(DynamicObject::GoString(data));
    sp.write_u64(GoValue::Object(id).encode());
}

/// go side: λ(v value) int
pub fn js_value_length(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let source = match JsValue::new(sp.read_u64()) {
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
    sp.write_u64(length as u64);
}

/// go side: λ(dest []byte, src value) (int, bool)
pub fn js_copy_bytes_to_go(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let (dest_ptr, dest_len) = sp.read_go_slice();
    let src_val = JsValue::new(sp.read_u64());

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
                sp.write_u64(GoValue::Number(len as f64).encode());
                sp.write_u8(1);
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
    sp.skip_u64().write_u8(0);
}

/// go side: λ(dest value, src []byte) (int, bool)
pub fn js_copy_bytes_to_js(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let dest_val = JsValue::new(sp.read_u64());
    let (src_ptr, src_len) = sp.read_go_slice();

    match dest_val {
        JsValue::Ref(dest_id) => {
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
                    sp.write_u64(GoValue::Number(len as f64).encode());
                    sp.write_u8(1);
                    return;
                }
                dest => eprintln!("Go trying to copy bytes into unsupported target {:?}", dest),
            }
        }
        value => eprintln!("Go trying to copy bytes into {:?}", value),
    }
    sp.write_u64(GoValue::Null.encode());
    sp.write_u8(0);
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
