// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

mod pending;
mod value;

use crate::{
    pending::{PENDING_EVENT, STYLUS_RESULT},
    value::*,
};
use arbutil::{wavm, Color};
use fnv::FnvHashSet as HashSet;
use go_abi::*;
use rand::RngCore;
use rand_pcg::Pcg32;
use std::{collections::BinaryHeap, convert::TryFrom, io::Write};

unsafe fn read_value_slice(mut ptr: u64, len: u64) -> Vec<JsValue> {
    let mut values = Vec::new();
    for _ in 0..len {
        let p = usize::try_from(ptr).expect("Go pointer didn't fit in usize");
        values.push(JsValue::new(wavm::caller_load64(p)));
        ptr += 8;
    }
    values
}

#[no_mangle]
pub unsafe extern "C" fn go__debug(x: usize) {
    println!("go debug: {}", x);
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_resetMemoryDataView(_: usize) {}

/// Safety: λ(code int32)
#[no_mangle]
pub unsafe extern "C" fn go__runtime_wasmExit(sp: usize) {
    let mut sp = GoStack::new(sp);
    std::process::exit(sp.read_u32() as i32);
}

/// Safety: λ(fd uintptr, p pointer, len int32)
#[no_mangle]
pub unsafe extern "C" fn go__runtime_wasmWrite(sp: usize) {
    let mut sp = GoStack::new(sp);
    let fd = sp.read_u64();
    let ptr = sp.read_u64();
    let len = sp.read_u32();
    let buf = wavm::read_slice(ptr, len.into());
    if fd == 2 {
        let stderr = std::io::stderr();
        let mut stderr = stderr.lock();
        stderr.write_all(&buf).unwrap();
    } else {
        let stdout = std::io::stdout();
        let mut stdout = stdout.lock();
        stdout.write_all(&buf).unwrap();
    }
}

// An increasing clock used when Go asks for time, measured in nanoseconds.
static mut TIME: u64 = 0;
// The amount of TIME advanced each check. Currently 10 milliseconds.
static mut TIME_INTERVAL: u64 = 10_000_000;

/// Safety: λ() int64
#[no_mangle]
pub unsafe extern "C" fn go__runtime_nanotime1(sp: usize) {
    let mut sp = GoStack::new(sp);
    TIME += TIME_INTERVAL;
    sp.write_u64(TIME);
}

/// Safety: λ() (seconds int64, nanos int32)
#[no_mangle]
pub unsafe extern "C" fn go__runtime_walltime(sp: usize) {
    let mut sp = GoStack::new(sp);
    TIME += TIME_INTERVAL;
    sp.write_u64(TIME / 1_000_000_000);
    sp.write_u32((TIME % 1_000_000_000) as u32);
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_walltime1(sp: usize) {
    let mut sp = GoStack::new(sp);
    TIME += TIME_INTERVAL;
    sp.write_u64(TIME / 1_000_000_000);
    sp.write_u64(TIME % 1_000_000_000);
}

static mut RNG: Option<Pcg32> = None;

unsafe fn get_rng<'a>() -> &'a mut Pcg32 {
    RNG.get_or_insert_with(|| Pcg32::new(0xcafef00dd15ea5e5, 0xa02bdbf7bb3c0a7))
}

/// Safety: λ(dest []byte)
#[no_mangle]
pub unsafe extern "C" fn go__runtime_getRandomData(sp: usize) {
    let mut sp = GoStack::new(sp);
    let rng = get_rng();
    let mut ptr = usize::try_from(sp.read_u64()).expect("Go getRandomData pointer not a usize");
    let mut len = sp.read_u64();
    while len >= 4 {
        wavm::caller_store32(ptr, rng.next_u32());
        ptr += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = rng.next_u32();
        for _ in 0..len {
            wavm::caller_store8(ptr, rem as u8);
            ptr += 1;
            rem >>= 8;
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct TimeoutInfo {
    time: u64,
    id: u32,
}

impl Ord for TimeoutInfo {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        other
            .time
            .cmp(&self.time)
            .then_with(|| other.id.cmp(&self.id))
    }
}

impl PartialOrd for TimeoutInfo {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(&other))
    }
}

#[derive(Default, Debug)]
struct TimeoutState {
    /// Contains tuples of (time, id)
    times: BinaryHeap<TimeoutInfo>,
    pending_ids: HashSet<u32>,
    next_id: u32,
}

static mut TIMEOUT_STATE: Option<TimeoutState> = None;

/// Safety: λ() (delay int64) int32
#[no_mangle]
pub unsafe extern "C" fn go__runtime_scheduleTimeoutEvent(sp: usize) {
    let mut sp = GoStack::new(sp);
    let mut time = sp.read_u64();
    time = time.saturating_mul(1_000_000); // milliseconds to nanoseconds
    time = time.saturating_add(TIME); // add the current time to the delay

    let state = TIMEOUT_STATE.get_or_insert_with(Default::default);
    let id = state.next_id;
    state.next_id += 1;
    state.times.push(TimeoutInfo { time, id });
    state.pending_ids.insert(id);

    sp.write_u32(id);
}

/// Safety: λ(id int32)
#[no_mangle]
pub unsafe extern "C" fn go__runtime_clearTimeoutEvent(sp: usize) {
    let mut sp = GoStack::new(sp);
    let id = sp.read_u32();

    let state = TIMEOUT_STATE.get_or_insert_with(Default::default);
    if !state.pending_ids.remove(&id) {
        eprintln!("Go attempting to clear not pending timeout event {}", id);
    }
}

macro_rules! unimpl_js {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub unsafe extern "C" fn $f(_: usize) {
                unimplemented!("Go JS interface {} not supported", stringify!($f));
            }
        )*
    }
}

unimpl_js!(
    go__syscall_js_valuePrepareString,
    go__syscall_js_valueLoadString,
    go__syscall_js_valueDelete,
    go__syscall_js_valueInvoke,
    go__syscall_js_valueInstanceOf,
);

/// Safety: λ(v value, field string) value
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueGet(sp: usize) {
    let mut sp = GoStack::new(sp);
    let source = JsValue::new(sp.read_u64());
    let field = sp.read_js_string();

    let value = match source {
        JsValue::Ref(id) => get_field(id, &field),
        val => {
            eprintln!(
                "Go attempting to read field {:?} . {}",
                val,
                String::from_utf8_lossy(&field),
            );
            GoValue::Null
        }
    };
    sp.write_u64(value.encode());
}

/// Safety: λ(v value, args []value) (value, bool)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueNew(sp: usize) {
    let mut sp = GoStack::new(sp);
    let pool = DynamicObjectPool::singleton();

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
    let args = read_value_slice(args_ptr, args_len);
    let value = match class {
        UINT8_ARRAY_ID => match args.get(0) {
            Some(JsValue::Number(size)) => DynamicObject::Uint8Array(vec![0; *size as usize]),
            _ => fail!("Go attempted to construct Uint8Array with bad args: {args:?}"),
        },
        DATE_ID => DynamicObject::Date,
        ARRAY_ID => {
            // Note: assumes values are only numbers and objects
            let values = args.into_iter().map(JsValue::assume_num_or_object);
            DynamicObject::ValueArray(values.collect())
        }
        _ => fail!("Go trying to construct unimplemented JS value {class}"),
    };
    let id = pool.insert(value);
    sp.write_u64(GoValue::Object(id).encode());
    sp.write_u8(1);
}

/// Safety: λ(dest value, src []byte) (int, bool)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_copyBytesToJS(sp: usize) {
    let mut sp = GoStack::new(sp);
    let dest_val = JsValue::new(sp.read_u64());
    let (src_ptr, src_len) = sp.read_go_slice();

    if let JsValue::Ref(dest_id) = dest_val {
        let dest = DynamicObjectPool::singleton().get_mut(dest_id);
        if let Some(DynamicObject::Uint8Array(buf)) = dest {
            if buf.len() as u64 != src_len {
                eprintln!(
                    "Go copying bytes from Go source length {} to JS dest length {}",
                    src_len,
                    buf.len(),
                );
            }
            let len = std::cmp::min(src_len, buf.len() as u64) as usize;
            // Slightly inefficient as this allocates a new temporary buffer
            buf[..len].copy_from_slice(&wavm::read_slice(src_ptr, len as u64));
            sp.write_u64(GoValue::Number(len as f64).encode());
            sp.write_u8(1);
            return;
        } else {
            eprintln!(
                "Go attempting to copy bytes into unsupported target {:?}",
                dest,
            );
        }
    } else {
        eprintln!("Go attempting to copy bytes into {:?}", dest_val);
    }
    sp.write_u64(GoValue::Null.encode());
    sp.write_u8(0);
}

/// Safety: λ(dest []byte, src value) (int, bool)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_copyBytesToGo(sp: usize) {
    let mut sp = GoStack::new(sp);
    let (dest_ptr, dest_len) = sp.read_go_slice();
    let src_val = JsValue::new(sp.read_u64());

    if let JsValue::Ref(src_id) = src_val {
        let source = DynamicObjectPool::singleton().get_mut(src_id);
        if let Some(DynamicObject::Uint8Array(buf)) = source {
            if buf.len() as u64 != dest_len {
                eprintln!(
                    "Go copying bytes from JS source length {} to Go dest length {}",
                    buf.len(),
                    dest_len,
                );
            }
            let len = std::cmp::min(buf.len() as u64, dest_len) as usize;
            wavm::write_slice(&buf[..len], dest_ptr);

            sp.write_u64(GoValue::Number(len as f64).encode());
            sp.write_u8(1);
            return;
        } else {
            eprintln!(
                "Go attempting to copy bytes from unsupported source {:?}",
                source,
            );
        }
    } else {
        eprintln!("Go attempting to copy bytes from {:?}", src_val);
    }
    sp.skip_u64().write_u8(0);
}

/// Safety: λ(array value, i int, v value)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueSetIndex(sp: usize) {
    let mut sp = GoStack::new(sp);
    let pool = DynamicObjectPool::singleton();

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            return
        }};
    }

    let source = match JsValue::new(sp.read_u64()) {
        JsValue::Ref(x) => pool.get_mut(x),
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
            prior.free();
        }
        _ => fail!("Go attempted to index into unsupported value {source:?} {index}"),
    }
}

/// Safety: λ(v value, method string, args []value) (value, bool)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueCall(sp: usize) {
    let mut sp = GoStack::new(sp);
    let object = JsValue::new(sp.read_u64());
    let method_name = sp.read_js_string();
    let (args_ptr, args_len) = sp.read_go_slice();
    let args = read_value_slice(args_ptr, args_len);
    let name = String::from_utf8_lossy(&method_name);
    let pool = DynamicObjectPool::singleton();
    use JsValue::*;

    macro_rules! fail {
        ($text:expr $(,$args:expr)*) => {{
            eprintln!($text $(,$args)*);
            sp.write_u64(GoValue::Null.encode());
            sp.write_u8(1);
            return
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
                        eprintln!("Go trying to write to unknown FD {}", fd);
                    }

                    pending::set_event(
                        func_id,
                        object,
                        vec![
                            GoValue::Null,                  // no error
                            GoValue::Number(length as f64), // amount written
                        ],
                    );
                    sp.resume();
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

            get_rng().fill_bytes(buf.as_mut_slice());
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
                            print!(" {}", String::from_utf8_lossy(data))
                        }
                        Some(DynamicObject::Uint8Array(data)) => {
                            print!(" 0x{}", hex::encode(data))
                        }
                        Some(other) => print!(" {other:?}"),
                        None => print!(" unknown"),
                    },
                }
            }
            println!();
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
        _ => fail!("Go trying to call unknown method {object:?} . {name}"),
    };

    sp.write_u64(value.encode());
    sp.write_u8(1);
}

/// Safety: λ(v value, field string, x value)
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueSet(sp: usize) {
    let mut sp = GoStack::new(sp);
    use JsValue::*;

    let source = JsValue::new(sp.read_u64());
    let field = sp.read_js_string();
    let new_value = JsValue::new(sp.read_u64());

    if source == Ref(GO_ID) && &field == b"_pendingEvent" && new_value == Ref(NULL_ID) {
        PENDING_EVENT = None;
        return;
    }

    let pool = DynamicObjectPool::singleton();
    if let (Ref(STYLUS_ID), b"result") = (source, field.as_slice()) {
        STYLUS_RESULT = Some(new_value);
        return;
    }
    if let Ref(id) = source {
        let source = pool.get(id);
        if let Some(DynamicObject::PendingEvent(_)) = source {
            if field == b"result" {
                return;
            }
        }
    }
    let field = String::from_utf8_lossy(&field).red();
    eprintln!("Go attempted to set unsupported value {source:?} field {field} to {new_value:?}",);
}

/// Safety: λ(v string) value
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_stringVal(sp: usize) {
    let mut sp = GoStack::new(sp);
    let pool = DynamicObjectPool::singleton();
    let data = sp.read_js_string();
    let id = pool.insert(DynamicObject::GoString(data));
    sp.write_u64(GoValue::Object(id).encode());
}

/// Safety: λ(v value) int
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueLength(sp: usize) {
    let mut sp = GoStack::new(sp);
    let source = JsValue::new(sp.read_u64());
    let pool = DynamicObjectPool::singleton();
    let source = match source {
        JsValue::Ref(x) => pool.get(x),
        _ => None,
    };
    let len = match source {
        Some(DynamicObject::Uint8Array(x)) => Some(x.len()),
        Some(DynamicObject::ValueArray(x)) => Some(x.len()),
        _ => None,
    };
    if let Some(len) = len {
        sp.write_u64(len as u64);
    } else {
        eprintln!(
            "Go attempted to get length of unsupported value {:?}",
            source,
        );
        sp.write_u64(0);
    }
}

/// Safety: λ(v value, i int) value
unsafe fn value_index_impl(sp: &mut GoStack) -> Result<GoValue, String> {
    let pool = DynamicObjectPool::singleton();
    let source = match JsValue::new(sp.read_u64()) {
        JsValue::Ref(x) => pool.get(x),
        val => return Err(format!("Go attempted to index into {:?}", val)),
    };
    let index = usize::try_from(sp.read_u64()).map_err(|e| format!("{:?}", e))?;
    let val = match source {
        Some(DynamicObject::Uint8Array(x)) => {
            Some(x.get(index).map(|x| GoValue::Number(*x as f64)))
        }
        Some(DynamicObject::ValueArray(x)) => Some(x.get(index).cloned()),
        _ => None,
    };
    match val {
        Some(Some(val)) => Ok(val),
        Some(None) => Err(format!(
            "Go attempted to index out of bounds into value {:?} index {}",
            source, index,
        )),
        None => Err(format!(
            "Go attempted to index into unsupported value {:?}",
            source
        )),
    }
}

/// Safety: λ(v value, i int) value
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueIndex(sp: usize) {
    let mut sp = GoStack::new(sp);
    match value_index_impl(&mut sp) {
        Ok(v) => sp.write_u64(v.encode()),
        Err(e) => {
            eprintln!("{}", e);
            sp.write_u64(GoValue::Null.encode())
        }
    };
}

/// Safety: λ(v value)
/// TODO: reference counting
#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_finalizeRef(sp: usize) {
    let mut sp = GoStack::new(sp);
    let val = JsValue::new(sp.read_u64());
    match val {
        JsValue::Ref(_)  => {}
        val => eprintln!("Go attempting to finalize {:?}", val),
    }
}

#[no_mangle]
pub unsafe extern "C" fn wavm__go_after_run() {
    let mut state = TIMEOUT_STATE.get_or_insert_with(Default::default);
    while let Some(info) = state.times.pop() {
        while state.pending_ids.contains(&info.id) {
            TIME = std::cmp::max(TIME, info.time);
            // Important: the current reference to state shouldn't be used after this resume call,
            // as it might during the resume call the reference might be invalidated.
            // That's why immediately after this resume call, we replace the reference
            // with a new reference to TIMEOUT_STATE.
            wavm_guest_call__resume();
            state = TIMEOUT_STATE.get_or_insert_with(Default::default);
        }
    }
}
