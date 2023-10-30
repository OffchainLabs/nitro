// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use fnv::FnvHashMap;
use parking_lot::Mutex;
use std::{
    collections::hash_map,
    fmt,
    hash::{Hash, Hasher},
    sync::Arc,
};

const CANONICAL_NAN_BITS: u64 = 0x7FF8000000000000;

#[derive(Default, Debug, Clone)]
pub struct JsObject(Arc<Mutex<FnvHashMap<String, JsValue>>>);

pub trait JsEnv {
    fn get_rng(&mut self) -> &mut dyn rand::RngCore;
    fn resume(&mut self) -> eyre::Result<()>;
}

impl JsObject {
    pub fn insert(&self, key: impl Into<String>, value: impl Into<JsValue>) {
        self.0.lock().insert(key.into(), value.into());
    }

    /// Identical to `insert` but with better type inference
    pub fn insert_func(
        &self,
        key: impl Into<String>,
        value: impl Fn(&mut dyn JsEnv, JsValue, Vec<JsValue>) -> eyre::Result<JsValue>
            + Send
            + Sync
            + 'static,
    ) {
        self.insert(key, value);
    }

    /// Returns `&JsValue::Undefined` if the key is not present
    pub fn get(&self, key: &str) -> JsValue {
        self.0.lock().get(key).cloned().unwrap_or_default()
    }
}

pub trait JsFunction: Send + Sync + 'static {
    fn call(&self, env: &mut dyn JsEnv, this: JsValue, args: Vec<JsValue>)
        -> eyre::Result<JsValue>;
}

impl<F> JsFunction for F
where
    F: Fn(&mut dyn JsEnv, JsValue, Vec<JsValue>) -> eyre::Result<JsValue> + Send + Sync + 'static,
{
    fn call(
        &self,
        env: &mut dyn JsEnv,
        this: JsValue,
        args: Vec<JsValue>,
    ) -> eyre::Result<JsValue> {
        self(env, this, args)
    }
}

#[derive(Clone)]
pub enum JsValue {
    Undefined,
    Null,
    Bool(bool),
    Number(f64),
    String(Arc<String>),
    Object(JsObject),
    Uint8Array(Arc<Mutex<Box<[u8]>>>),
    Array(Arc<Mutex<Vec<JsValue>>>),
    Function(Arc<Box<dyn JsFunction>>),
}

impl JsValue {
    pub fn assume_object(self, name: &str) -> JsObject {
        match self {
            Self::Object(x) => x,
            _ => panic!("Expected JS Value {name} to be an object but got {self:?}"),
        }
    }
}

impl From<JsObject> for JsValue {
    fn from(value: JsObject) -> Self {
        Self::Object(value)
    }
}

impl From<Vec<JsValue>> for JsValue {
    fn from(value: Vec<JsValue>) -> Self {
        Self::Array(Arc::new(Mutex::new(value)))
    }
}

impl<F: JsFunction> From<F> for JsValue {
    fn from(value: F) -> Self {
        Self::Function(Arc::new(Box::new(value)))
    }
}

impl Default for JsValue {
    fn default() -> Self {
        Self::Undefined
    }
}

#[derive(Hash, PartialEq)]
enum JsValueEquality<'a> {
    AlwaysEqual,
    Bool(bool),
    Number(u64),
    String(&'a str),
    Pointer(usize),
}

impl JsValue {
    /// We follow the JS [SameValueZero](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Equality_comparisons_and_sameness#same-value-zero_equality) rule of equality.
    fn equality(&self) -> JsValueEquality<'_> {
        match self {
            JsValue::Undefined => JsValueEquality::AlwaysEqual,
            JsValue::Null => JsValueEquality::AlwaysEqual,
            JsValue::Bool(x) => JsValueEquality::Bool(*x),
            // Treat all NaN values as equal
            JsValue::Number(x) if x.is_nan() => JsValueEquality::Number(CANONICAL_NAN_BITS),
            // Treat all zero values as equal
            JsValue::Number(x) if *x == 0. => JsValueEquality::Number(0_f64.to_bits()),
            JsValue::Number(x) => JsValueEquality::Number(x.to_bits()),
            JsValue::String(x) => JsValueEquality::String(x.as_str()),
            JsValue::Object(x) => JsValueEquality::Pointer(Arc::as_ptr(&x.0) as usize),
            JsValue::Uint8Array(x) => JsValueEquality::Pointer(Arc::as_ptr(x) as usize),
            JsValue::Array(x) => JsValueEquality::Pointer(Arc::as_ptr(x) as usize),
            JsValue::Function(x) => JsValueEquality::Pointer(Arc::as_ptr(x) as usize),
        }
    }

    fn go_typecode(&self) -> u8 {
        match self {
            JsValue::Undefined => 0,
            JsValue::Null => 0,
            JsValue::Bool(_) => 0,
            JsValue::Number(_) => 0,
            JsValue::Object(_) => 1,
            JsValue::Uint8Array(_) => 1,
            JsValue::Array(_) => 1,
            JsValue::String(_) => 2,
            // Symbols are 3 but we don't support them
            JsValue::Function(_) => 4,
        }
    }
}

impl PartialEq for JsValue {
    fn eq(&self, other: &Self) -> bool {
        if std::mem::discriminant(self) != std::mem::discriminant(other) {
            return false;
        }
        self.equality() == other.equality()
    }
}

impl Eq for JsValue {}

impl Hash for JsValue {
    fn hash<H: Hasher>(&self, state: &mut H) {
        std::mem::discriminant(self).hash(state);
        self.equality().hash(state);
    }
}

impl fmt::Debug for JsValue {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            JsValue::Undefined => write!(f, "undefined"),
            JsValue::Null => write!(f, "null"),
            JsValue::Bool(x) => write!(f, "{x}"),
            JsValue::Number(x) => write!(f, "{x}"),
            JsValue::String(x) => write!(f, "{x:?}"),
            JsValue::Object(x) => write!(f, "{x:?}"),
            JsValue::Uint8Array(x) => write!(f, "{x:?}"),
            JsValue::Array(x) => write!(f, "{x:?}"),
            JsValue::Function(x) => write!(f, "<function {:?}>", Arc::as_ptr(x)),
        }
    }
}

enum ValueOrPoolId {
    Value(JsValue),
    PoolId(u32),
}

/// Represents the bits of a float for a JS Value ID in Go.
/// Warning: Equality does not treat equal but different floats as equal.
#[derive(Clone, Copy, PartialEq)]
pub struct JsValueId(pub u64);

pub const NAN_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS);
pub const ZERO_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | 1);
pub const NULL_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | 2);
pub const TRUE_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | 3);
pub const FALSE_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | 4);
pub const GLOBAL_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | (1 << 32) | 5);
pub const GO_OBJECT_ID: JsValueId = JsValueId(CANONICAL_NAN_BITS | (1 << 32) | 6);

impl JsValueId {
    /// This method is only for non-number values (pool IDs)
    fn new(go_typecode: u8, pool_id: u32) -> Self {
        Self(CANONICAL_NAN_BITS | (u64::from(go_typecode) << 32) | u64::from(pool_id))
    }

    fn as_value_or_pool_id(self) -> ValueOrPoolId {
        let id_float = f64::from_bits(self.0);
        if id_float == 0. {
            return ValueOrPoolId::Value(JsValue::Undefined);
        }
        if !id_float.is_nan() {
            return ValueOrPoolId::Value(JsValue::Number(id_float));
        }
        ValueOrPoolId::PoolId(self.0 as u32)
    }
}

impl fmt::Debug for JsValueId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "JsValueId(0x{:016x})", self.0)
    }
}

/// A reference count of None means infinity (never freeable)
struct ReferenceCount(Option<usize>);

impl ReferenceCount {
    pub fn one() -> Self {
        ReferenceCount(Some(1))
    }

    pub fn infinity() -> Self {
        ReferenceCount(None)
    }

    pub fn increment(&mut self) {
        if let Some(count) = &mut self.0 {
            *count += 1;
        }
    }

    /// Returns true if the reference count has reached zero
    pub fn decrement(&mut self) -> bool {
        let Some(count) = &mut self.0 else {
            return false;
        };
        if *count == 0 {
            panic!("Attempted to decrement reference count of zero")
        }
        *count -= 1;
        *count == 0
    }
}

struct ValueAndRefCount {
    value: JsValue,
    ref_count: ReferenceCount,
}

#[derive(Default)]
pub struct JsValuePoolInner {
    value_by_id: FnvHashMap<u32, ValueAndRefCount>,
    id_by_value: FnvHashMap<JsValue, u32>,
    next_id: u32,
}

impl JsValuePoolInner {
    fn insert_static(&mut self, value: JsValue) -> JsValueId {
        let id = self.next_id;
        self.next_id += 1;
        self.value_by_id.insert(
            id,
            ValueAndRefCount {
                value: value.clone(),
                ref_count: ReferenceCount::infinity(),
            },
        );
        let go_typecode = value.go_typecode();
        self.id_by_value.insert(value, id);
        JsValueId::new(go_typecode, id)
    }
}

#[derive(Clone)]
pub struct JsValuePool(Arc<Mutex<JsValuePoolInner>>);

impl JsValuePool {
    pub fn new(globals: JsValue, go_object: JsValue) -> Self {
        let mut this = JsValuePoolInner::default();
        assert_eq!(
            this.insert_static(JsValue::Number(f64::from_bits(CANONICAL_NAN_BITS))),
            NAN_ID,
        );
        assert_eq!(this.insert_static(JsValue::Number(0.)), ZERO_ID);
        assert_eq!(this.insert_static(JsValue::Null), NULL_ID);
        assert_eq!(this.insert_static(JsValue::Bool(true)), TRUE_ID);
        assert_eq!(this.insert_static(JsValue::Bool(false)), FALSE_ID);
        assert_eq!(this.insert_static(globals), GLOBAL_ID);
        assert_eq!(this.insert_static(go_object), GO_OBJECT_ID);
        Self(Arc::new(Mutex::new(this)))
    }

    pub fn id_to_value(&self, id: JsValueId) -> JsValue {
        let pool_id = match id.as_value_or_pool_id() {
            ValueOrPoolId::Value(value) => return value,
            ValueOrPoolId::PoolId(id) => id,
        };
        let inner = self.0.lock();
        let Some(ValueAndRefCount { value, .. }) = inner.value_by_id.get(&pool_id) else {
            panic!("JsValuePool missing {id:?}");
        };
        let expected_id = JsValueId::new(value.go_typecode(), pool_id);
        if id.0 != expected_id.0 {
            panic!("Got non-canonical JS ValueID {id:?} but expected {expected_id:?}");
        }
        value.clone()
    }

    /// Warning: this increments the reference count for the returned id
    pub fn value_to_id(&self, value: JsValue) -> JsValueId {
        if let JsValue::Number(n) = value {
            if n != 0. && !n.is_nan() {
                return JsValueId(n.to_bits());
            }
        }
        let mut inner = self.0.lock();
        let go_ty = value.go_typecode();
        let pool_id = if let Some(id) = inner.id_by_value.get(&value).cloned() {
            inner
                .value_by_id
                .get_mut(&id)
                .unwrap()
                .ref_count
                .increment();
            id
        } else {
            let id = inner.next_id;
            inner.next_id += 1;
            inner.value_by_id.insert(
                id,
                ValueAndRefCount {
                    value: value.clone(),
                    ref_count: ReferenceCount::one(),
                },
            );
            inner.id_by_value.insert(value, id);
            id
        };
        JsValueId::new(go_ty, pool_id)
    }

    pub fn finalize(&self, id: JsValueId) {
        let pool_id = match id.as_value_or_pool_id() {
            ValueOrPoolId::Value(_) => return,
            ValueOrPoolId::PoolId(id) => id,
        };
        let mut inner = self.0.lock();
        let hash_map::Entry::Occupied(mut entry) = inner.value_by_id.entry(pool_id) else {
            panic!("Attempted to finalize unknown {id:?}");
        };
        if entry.get_mut().ref_count.decrement() {
            let value = entry.remove().value;
            let removed = inner.id_by_value.remove(&value);
            if removed != Some(pool_id) {
                panic!("Removing {id:?} but corresponding value {value:?} mapped to {removed:?} in id_by_value");
            }
        }
    }
}
