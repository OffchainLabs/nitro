// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::pending::{PendingEvent, PENDING_EVENT, STYLUS_RESULT};
use arbutil::DebugColor;
use eyre::{bail, Result};
use fnv::FnvHashMap as HashMap;

pub const ZERO_ID: u32 = 1;
pub const NULL_ID: u32 = 2;
pub const GLOBAL_ID: u32 = 5;
pub const GO_ID: u32 = 6;
pub const STYLUS_ID: u32 = 7;

pub const OBJECT_ID: u32 = 100;
pub const ARRAY_ID: u32 = 101;
pub const PROCESS_ID: u32 = 102;
pub const FS_ID: u32 = 103;
pub const UINT8_ARRAY_ID: u32 = 104;
pub const CRYPTO_ID: u32 = 105;
pub const DATE_ID: u32 = 106;
pub const CONSOLE_ID: u32 = 107;

pub const FS_CONSTANTS_ID: u32 = 200;

pub const DYNAMIC_OBJECT_ID_BASE: u32 = 10000;

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

    pub fn assume_id(self) -> Result<u32> {
        match self {
            GoValue::Object(id) => Ok(id),
            x => bail!("not an id: {}", x.debug_red()),
        }
    }

    pub unsafe fn free(self) {
        use GoValue::*;
        match self {
            Object(id) => drop(DynamicObjectPool::singleton().remove(id)),
            Undefined | Null | Number(_) => {}
            _ => unimplemented!(),
        }
    }
}

#[derive(Debug, Clone)]
pub(crate) enum DynamicObject {
    Uint8Array(Vec<u8>),
    GoString(Vec<u8>),
    FunctionWrapper(u32), // the func_id
    PendingEvent(PendingEvent),
    ValueArray(Vec<GoValue>),
    Date,
}

#[derive(Default, Debug)]
pub(crate) struct DynamicObjectPool {
    objects: HashMap<u32, DynamicObject>,
    free_ids: Vec<u32>,
}

pub(crate) static mut DYNAMIC_OBJECT_POOL: Option<DynamicObjectPool> = None;

impl DynamicObjectPool {
    pub unsafe fn singleton<'a>() -> &'a mut Self {
        DYNAMIC_OBJECT_POOL.get_or_insert_with(Default::default)
    }

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

pub unsafe fn get_field(source: u32, field: &[u8]) -> GoValue {
    use DynamicObject::*;
    let pool = DynamicObjectPool::singleton();

    if let Some(source) = pool.get(source) {
        return match (source, field) {
            (PendingEvent(event), b"id") => event.id.assume_num_or_object(),
            (PendingEvent(event), b"this") => event.this.assume_num_or_object(),
            (PendingEvent(event), b"args") => {
                let args = ValueArray(event.args.clone());
                let id = pool.insert(args);
                GoValue::Object(id)
            }
            _ => {
                let field = String::from_utf8_lossy(field);
                eprintln!(
                    "Go trying to access unimplemented unknown JS value {source:?} field {field}",
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
        (GLOBAL_ID, b"console") => GoValue::Object(CONSOLE_ID),
        (GLOBAL_ID, b"fetch") => GoValue::Undefined, // Triggers a code path in Go for a fake network impl
        (FS_ID, b"constants") => GoValue::Object(FS_CONSTANTS_ID),
        (
            FS_CONSTANTS_ID,
            b"O_WRONLY" | b"O_RDWR" | b"O_CREAT" | b"O_TRUNC" | b"O_APPEND" | b"O_EXCL",
        ) => GoValue::Number(-1.),
        (GO_ID, b"_pendingEvent") => match &PENDING_EVENT {
            Some(event) => {
                let event = PendingEvent(event.clone());
                let id = pool.insert(event);
                GoValue::Object(id)
            }
            None => GoValue::Null,
        },
        (GLOBAL_ID, b"stylus") => GoValue::Object(STYLUS_ID),
        (STYLUS_ID, b"result") => match &mut STYLUS_RESULT {
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
