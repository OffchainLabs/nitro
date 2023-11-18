// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//! This crate implements a Js Runtime meant to back the functionality needed by the Go runtime in WASM.
//!
//! Official reference implementation
//!     [js.go](https://github.com/golang/go/blob/go1.21.4/src/syscall/js/js.go)
//!     [wasm_exec.js](https://github.com/golang/go/blob/go1.21.4/misc/wasm/wasm_exec.js)

mod evm_api;
mod js_core;
mod runtime;

pub use js_core::{JsEnv, JsValue, JsValueId};

use eyre::{bail, Result};
use js_core::{JsObject, GLOBAL_ID, NAN_ID, NULL_ID, ZERO_ID};
use parking_lot::Mutex;
use std::sync::Arc;

pub fn get_null() -> JsValueId {
    NULL_ID
}

pub fn get_number(f: f64) -> JsValueId {
    if f.is_nan() {
        NAN_ID
    } else if f == 0. {
        ZERO_ID
    } else {
        JsValueId(f.to_bits())
    }
}

pub struct JsState {
    values: js_core::JsValuePool,
}

impl JsState {
    pub fn new() -> Self {
        Self {
            values: js_core::JsValuePool::new(
                runtime::make_globals_object(),
                runtime::make_go_object(),
            ),
        }
    }

    pub fn finalize_ref(&self, id: JsValueId) {
        self.values.finalize(id)
    }

    pub fn value_get(&self, object: JsValueId, field: &str) -> JsValueId {
        let value = self
            .values
            .id_to_value(object)
            .assume_object("valueGet target")
            .get(field);
        self.values.assign_id(value)
    }

    pub fn value_set(&self, object: JsValueId, field: &str, new_value: JsValueId) {
        let new_value = self.values.id_to_value(new_value);
        self.values
            .id_to_value(object)
            .assume_object("valueSet target")
            .insert(field, new_value);
    }

    pub fn value_index(&self, source: JsValueId, index: usize) -> JsValueId {
        let source = self.values.id_to_value(source);
        let result = match &source {
            JsValue::Array(array) => array.lock().get(index).cloned(),
            JsValue::Uint8Array(array) => {
                array.lock().get(index).map(|x| JsValue::Number(*x as f64))
            }
            _ => {
                panic!("Go attempted to call valueIndex on invalid type: {source:?}");
            }
        };
        let result = result.unwrap_or_else(|| {
            eprintln!("Go attempted to index out-of-bounds index {index} on {source:?}");
            JsValue::Undefined
        });
        self.values.assign_id(result)
    }

    pub fn value_set_index(&self, source: JsValueId, index: usize, new_value: JsValueId) {
        let source = self.values.id_to_value(source);
        let new_value = self.values.id_to_value(new_value);
        match &source {
            JsValue::Array(array) => {
                let mut array = array.lock();
                if index >= array.len() {
                    array.resize(index + 1, JsValue::Undefined);
                }
                array[index] = new_value;
            }
            JsValue::Uint8Array(array) => {
                let mut array = array.lock();
                let new_value = match new_value {
                    JsValue::Number(x) => x as u8,
                    _ => {
                        eprintln!("Go is setting a Uint8Array index to {new_value:?}");
                        0
                    }
                };
                if index >= array.len() {
                    eprintln!("Go is setting out-of-range index {index} in Uint8Array of size {} to {new_value:?}", array.len());
                } else {
                    array[index] = new_value;
                }
            }
            _ => {
                panic!("Go attempted to call valueSetIndex on invalid type: {source:?}");
            }
        }
    }

    pub fn value_invoke<'a>(
        &self,
        env: &'a mut (dyn JsEnv + 'a),
        func: JsValueId,
        args: &[JsValueId],
    ) -> Result<JsValueId> {
        let this = self.values.id_to_value(func);
        let JsValue::Function(func) = this else {
            bail!("Go attempted to invoke non-function {this:?}");
        };
        let args = args.iter().map(|x| self.values.id_to_value(*x)).collect();
        let result = func.call(env, JsValue::Undefined, args)?;
        Ok(self.values.assign_id(result))
    }

    pub fn value_call<'a>(
        &self,
        env: &'a mut (dyn JsEnv + 'a),
        object: JsValueId,
        method: &str,
        args: &[JsValueId],
    ) -> Result<JsValueId> {
        let this = self.values.id_to_value(object);
        let object = this.clone().assume_object("valueCall target");
        let JsValue::Function(function) = object.get(method) else {
            panic!("Go attempted to call {object:?} non-function field {method}");
        };
        let args = args.iter().map(|x| self.values.id_to_value(*x)).collect();
        let result = function.call(env, this, args)?;
        Ok(self.values.assign_id(result))
    }

    pub fn value_new<'a>(
        &self,
        env: &'a mut (dyn JsEnv + 'a),
        constructor: JsValueId,
        args: &[JsValueId],
    ) -> Result<JsValueId> {
        // All of our constructors are normal functions that work via a call
        let function = match self.values.id_to_value(constructor) {
            JsValue::Function(function) => function,
            x => panic!("Go tried to construct non-function {x:?}"),
        };
        let args = args.iter().map(|x| self.values.id_to_value(*x)).collect();
        let result = function.call(env, JsValue::Undefined, args)?;
        Ok(self.values.assign_id(result))
    }

    pub fn string_val(&self, s: String) -> JsValueId {
        self.values.assign_id(JsValue::String(Arc::new(s)))
    }

    pub fn value_length(&self, array: JsValueId) -> usize {
        let len = match self.values.id_to_value(array) {
            JsValue::Array(array) => array.lock().len(),
            JsValue::Uint8Array(array) => array.lock().len(),
            JsValue::String(data) => data.encode_utf16().count(),
            x => panic!("Go tried to call valueLength on unsupported type: {x:?}"),
        };
        len
    }

    /// Creates a uint8 array from the contents of a value coercible string.
    pub fn value_prepare_string(&self, text: JsValueId) -> (JsValueId, u64) {
        let text = match self.values.id_to_value(text) {
            JsValue::String(text) => text,
            JsValue::Bool(x) => Arc::new(format!("{x}")),
            JsValue::Number(x) => Arc::new(format!("{x}")),
            x => panic!("Go tried to call prepareString on unsupported type: {x:?}"),
        };
        let len = text.len() as u64;
        let text = JsValue::new_uint8_array(text.as_bytes().to_vec());
        let id = self.values.assign_id(text);
        (id, len)
    }

    /// Gets the contents of a uint8 array.
    pub fn get_uint8_array(&self, array: JsValueId) -> Arc<Mutex<Box<[u8]>>> {
        match self.values.id_to_value(array) {
            JsValue::Uint8Array(text) => text,
            x => panic!("value {array:?} not a uint8 array: {x:?}"),
        }
    }

    /// Copies bytes from a uint8 array, returning the number of bytes written.
    pub fn copy_bytes_to_go(
        &self,
        src: JsValueId,
        write_bytes: impl FnOnce(&[u8]) -> usize, // returns number of bytes written
    ) -> Result<u64> {
        let len = match self.values.id_to_value(src) {
            JsValue::Uint8Array(array) => write_bytes(&array.lock()),
            x => bail!("Go tried to call copyBytesToGo on invalid type: {x:?}"),
        };
        Ok(len as u64)
    }

    /// Copies bytes into a uint8 array, returning the number of bytes written.
    pub fn copy_bytes_to_js(
        &self,
        dest: JsValueId,
        write_bytes: impl FnOnce(&mut [u8]) -> usize, // returns number of bytes written
    ) -> Result<u64> {
        let len = match self.values.id_to_value(dest) {
            JsValue::Uint8Array(array) => write_bytes(&mut array.lock()),
            x => bail!("Go tried to call copyBytesToJs on invalid type: {x:?}"),
        };
        Ok(len as u64)
    }

    /// Gets the globals object for use in Rust
    pub fn get_globals(&self) -> JsObject {
        match self.values.id_to_value(GLOBAL_ID) {
            JsValue::Object(object) => object,
            _ => unreachable!(),
        }
    }
}

impl Default for JsState {
    fn default() -> Self {
        Self::new()
    }
}
