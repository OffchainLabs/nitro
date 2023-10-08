use crate::wavm;

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
    /*
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

    // Important: the current reference to the pool shouldn't be used after this resume call
    wavm_guest_call__resume();

    let pool = DynamicObjectPool::singleton();
    for id in ids {
        pool.remove(id);
    }
    stylus_result(func).1.len()
    */
    todo!("go_stub__run_stylus_closure")
}

/// Copies the current closure results' lengths.
///
/// # Safety
///
/// Panics if no result exists.
/// Non-reentrant.
#[no_mangle]
pub unsafe extern "C" fn go_stub__read_closure_lens(func: u32, lens: *mut usize) {
    /*
    let outs = stylus_result(func).1;
    let pool = DynamicObjectPool::singleton();

    for (index, out) in outs.iter().enumerate() {
        let id = out.assume_id().unwrap();
        let Some(DynamicObject::Uint8Array(out)) = pool.get(id) else {
            panic!("bad inner return value for func {}", func.red())
        };
        wavm::caller_store32(lens.add(index) as usize, out.len() as u32);
    }
    */
    todo!("go_stub__read_closure_lens")
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
    /*
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
    */
    todo!("go_stub__drop_closure_outs")
}
