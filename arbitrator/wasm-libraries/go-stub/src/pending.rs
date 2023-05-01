// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::{wavm, Color};
use go_abi::wavm_guest_call__resume;
use std::mem;

use crate::value::{DynamicObject, DynamicObjectPool, GoValue, JsValue, STYLUS_ID};

/// The event Go will execute next
pub(crate) static mut PENDING_EVENT: Option<PendingEvent> = None;

/// The stylus return result
pub(crate) static mut STYLUS_RESULT: Option<JsValue> = None;

#[derive(Clone, Debug)]
pub(crate) struct PendingEvent {
    pub id: JsValue,
    pub this: JsValue,
    pub args: Vec<GoValue>,
}

/// Sets the Go runtime's pending event.
///
/// # Safety
///
/// Non-reentrant.
pub(crate) unsafe fn set_event(id: u32, this: JsValue, args: Vec<GoValue>) {
    let id = JsValue::Number(id as f64);
    PENDING_EVENT = Some(PendingEvent { id, this, args });
}

/// Executes a Stylus closure, calling back into go via `resume`.
/// Returns the number of outputs, which are stored in the `STYLUS_RESULT` singleton.
///
/// # Safety
///
/// Corrupts the stack pointer. No `GoStack` functions may be ran until `sp.reset()`.
/// Leaks object ids unless `go_stub__drop_closure_outs` is called.
#[no_mangle]
pub unsafe extern "C" fn go_stub__run_stylus_closure(
    func: u32,
    data: *const *const u8,
    lens: *const usize,
    count: usize,
) -> usize {
    let this = JsValue::Ref(STYLUS_ID);
    let pool = DynamicObjectPool::singleton();

    let mut args = vec![];
    let mut ids = vec![];
    for i in 0..count {
        let data = wavm::caller_load32(data.add(i) as usize);
        let len = wavm::caller_load32(lens.add(i) as usize);
        let arg = wavm::read_slice_usize(data as usize, len as usize);

        let id = pool.insert(DynamicObject::Uint8Array(arg));
        args.push(GoValue::Object(id));
        ids.push(id);
    }
    let Some(DynamicObject::FunctionWrapper(func)) = pool.get(func).cloned() else {
        panic!("missing func {}", func.red())
    };
    set_event(func, this, args);

    #[allow(clippy::drop_ref)]
    mem::drop(pool);
    wavm_guest_call__resume();

    let pool = DynamicObjectPool::singleton();
    for id in ids {
        pool.remove(id);
    }
    stylus_result(func).1.len()
}

/// Copies the current closure results' lengths.
///
/// # Safety
///
/// Panics if no result exists.
/// Non-reentrant.
#[no_mangle]
pub unsafe extern "C" fn go_stub__read_closure_lens(func: u32, lens: *mut usize) {
    let outs = stylus_result(func).1;
    let pool = DynamicObjectPool::singleton();

    for (index, out) in outs.iter().enumerate() {
        let id = out.assume_id().unwrap();
        let Some(DynamicObject::Uint8Array(out)) = pool.get(id) else {
            panic!("bad inner return value for func {}", func.red())
        };
        wavm::caller_store32(lens.add(index) as usize, out.len() as u32);
    }
}

/// Copies the bytes of the current closure results, releasing the objects.
///
/// # Safety
///
/// Panics if no result exists.
/// Unsound if `data` cannot store the bytes.
/// Non-reentrant.
#[no_mangle]
pub unsafe extern "C" fn go_stub__drop_closure_outs(func: u32, data: *const *mut u8) {
    let (object, outs) = stylus_result(func);
    let pool = DynamicObjectPool::singleton();

    for (index, out) in outs.iter().enumerate() {
        let id = out.assume_id().unwrap();
        let Some(DynamicObject::Uint8Array(out)) = pool.remove(id) else {
            panic!("bad inner return value for func {}", func.red())
        };
        let ptr = wavm::caller_load32(data.add(index) as usize);
        wavm::write_slice_usize(&out, ptr as usize)
    }
    pool.remove(object);
}

/// Retrieves the id and value of the current closure result.
///
/// # Safety
///
/// Panics if no result exists.
/// Non-reentrant.
unsafe fn stylus_result(func: u32) -> (u32, &'static [GoValue]) {
    let stylus_result = STYLUS_RESULT.as_ref().unwrap();
    let pool = DynamicObjectPool::singleton();

    let JsValue::Ref(id) = stylus_result else {
        panic!("bad return value for func {}", func.red())
    };
    let Some(DynamicObject::ValueArray(output)) = pool.get(*id) else {
        panic!("bad return value for func {}", func.red())
    };
    (*id, output)
}
