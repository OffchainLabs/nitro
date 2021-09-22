mod value;

use crate::value::*;
use rand::RngCore;
use rand_pcg::Pcg32;
use std::{convert::TryFrom, io::Write};

#[allow(dead_code)]
extern "C" {
    fn wavm_caller_module_memory_load8(ptr: usize) -> u8;
    fn wavm_caller_module_memory_load32(ptr: usize) -> u32;
    fn wavm_caller_module_memory_store8(ptr: usize, val: u8);
    fn wavm_caller_module_memory_store32(ptr: usize, val: u32);

    fn wavm_guest_call__getsp() -> usize;
    fn wavm_guest_call__resume();
}

unsafe fn wavm_caller_module_memory_load64(ptr: usize) -> u64 {
    let lower = wavm_caller_module_memory_load32(ptr);
    let upper = wavm_caller_module_memory_load32(ptr + 4);
    lower as u64 | ((upper as u64) << 32)
}

unsafe fn wavm_caller_module_memory_store64(ptr: usize, val: u64) {
    wavm_caller_module_memory_store32(ptr, val as u32);
    wavm_caller_module_memory_store32(ptr + 4, (val >> 32) as u32);
}

#[derive(Clone, Copy)]
#[repr(transparent)]
pub struct GoStack(usize);

impl GoStack {
    unsafe fn read_u32(self, offset: usize) -> u32 {
        wavm_caller_module_memory_load32(self.0 + (offset + 1) * 8)
    }

    unsafe fn read_i32(self, offset: usize) -> i32 {
        self.read_u32(offset) as i32
    }

    unsafe fn read_u64(self, offset: usize) -> u64 {
        wavm_caller_module_memory_load64(self.0 + (offset + 1) * 8)
    }

    unsafe fn write_u8(self, offset: usize, x: u8) {
        wavm_caller_module_memory_store8(self.0 + (offset + 1) * 8, x);
    }

    unsafe fn write_u64(self, offset: usize, x: u64) {
        wavm_caller_module_memory_store64(self.0 + (offset + 1) * 8, x);
    }
}

fn interpret_value(repr: u64) -> InterpValue {
    let float = f64::from_bits(repr);
    if float.is_nan() && repr != f64::NAN.to_bits() {
        let id = repr as u32;
        if id == ZERO_ID {
            return InterpValue::Number(0.);
        }
        return InterpValue::Ref(id);
    }
    InterpValue::Number(float)
}

unsafe fn read_value_slice(mut ptr: u64, len: u64) -> Vec<InterpValue> {
    let mut values = Vec::new();
    for _ in 0..len {
        let p = usize::try_from(ptr).expect("Go pointer didn't fit in usize");
        values.push(interpret_value(wavm_caller_module_memory_load64(p)));
        ptr += 8;
    }
    values
}

unsafe fn read_slice(ptr: u64, mut len: u64) -> Vec<u8> {
    let mut data = Vec::with_capacity(len as usize);
    if len == 0 {
        return data;
    }
    let mut ptr = usize::try_from(ptr).expect("Go pointer didn't fit in usize");
    while len >= 4 {
        data.extend(wavm_caller_module_memory_load32(ptr).to_le_bytes());
        ptr += 4;
        len -= 4;
    }
    for _ in 0..len {
        data.push(wavm_caller_module_memory_load8(ptr));
        ptr += 1;
    }
    data
}

#[no_mangle]
pub unsafe extern "C" fn go__debug(x: usize) {
    println!("go debug: {}", x);
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_resetMemoryDataView(_: GoStack) {}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_wasmExit(sp: GoStack) {
    std::process::exit(sp.read_i32(0));
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_wasmWrite(sp: GoStack) {
    let fd = sp.read_u64(0);
    let ptr = sp.read_u64(1);
    let len = sp.read_u32(2);
    let buf = read_slice(ptr, len.into());
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

#[no_mangle]
pub unsafe extern "C" fn go__runtime_nanotime1(sp: GoStack) {
    TIME += TIME_INTERVAL;
    sp.write_u64(0, TIME);
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_walltime1(sp: GoStack) {
    TIME += TIME_INTERVAL;
    sp.write_u64(0, TIME / 1_000_000_000);
    sp.write_u64(1, TIME % 1_000_000_000);
}

static mut RNG: Option<Pcg32> = None;

#[no_mangle]
pub unsafe extern "C" fn go__runtime_getRandomData(sp: GoStack) {
    let rng = RNG.get_or_insert_with(|| Pcg32::new(0xcafef00dd15ea5e5, 0xa02bdbf7bb3c0a7));
    let mut ptr =
        usize::try_from(sp.read_u64(0)).expect("Go getRandomData pointer didn't fit in usize");
    let mut len = sp.read_u64(1);
    while len >= 4 {
        wavm_caller_module_memory_store32(ptr, rng.next_u32());
        ptr += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = rng.next_u32();
        for _ in 0..len {
            wavm_caller_module_memory_store8(ptr, rem as u8);
            ptr += 1;
            rem >>= 8;
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_scheduleTimeoutEvent(_: GoStack) {
    todo!()
}

#[no_mangle]
pub unsafe extern "C" fn go__runtime_clearTimeoutEvent(_: GoStack) {
    todo!()
}

macro_rules! unimpl_js {
    ($($f:ident),* $(,)?) => {
        $(
            #[no_mangle]
            pub unsafe extern "C" fn $f(_: GoStack) {
                unimplemented!("Go JS interface {} not supported", stringify!($f));
            }
        )*
    }
}

unimpl_js!(
    go__syscall_js_finalizeRef,
    go__syscall_js_stringVal,
    go__syscall_js_valueSetIndex,
    go__syscall_js_valuePrepareString,
    go__syscall_js_valueLoadString,
);

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueGet(sp: GoStack) {
    let source = interpret_value(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let field = read_slice(field_ptr, field_len);
    let value = match source {
        InterpValue::Ref(id) => get_field(id, &field),
        InterpValue::Number(n) => {
            eprintln!(
                "Go attempting to read field of float: {} . {}",
                n,
                String::from_utf8_lossy(&field),
            );
            GoValue::Null
        }
    };
    sp.write_u64(3, value.encode());
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueNew(sp: GoStack) {
    let class = sp.read_u32(0);
    let args_ptr = sp.read_u64(1);
    let args_len = sp.read_u64(2);
    let args = read_value_slice(args_ptr, args_len);
    if class == UINT8_ARRAY_ID {
        if let Some(InterpValue::Number(size)) = args.get(0) {
            let id = DynamicObjectPool::singleton()
                .insert(DynamicObject::Uint8Array(vec![0; *size as usize]));
            sp.write_u64(4, GoValue::Object(id).encode());
            sp.write_u8(5, 1);
            return;
        } else {
            eprintln!(
                "Go attempted to construct Uint8Array with bad args: {:?}",
                args,
            );
        }
    } else {
        eprintln!(
            "Go attempting to construct unimplemented JS value {}",
            class,
        );
    }
    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_copyBytesToJS(sp: GoStack) {
    let dest_val = interpret_value(sp.read_u64(0));
    if let InterpValue::Ref(dest_id) = dest_val {
        let src_ptr = sp.read_u64(1);
        let src_len = sp.read_u64(2);
        let dest = DynamicObjectPool::singleton().get_mut(dest_id);
        if let Some(DynamicObject::Uint8Array(buf)) = dest {
            if buf.len() as u64 != src_len {
                eprintln!(
                    "Go copying bytes from source length {} to dest length {}",
                    src_len,
                    buf.len(),
                );
            }
            let len = std::cmp::min(src_len, buf.len() as u64) as usize;
            // Slightly inefficient as this allocates a new temporary buffer
            buf[..len].copy_from_slice(&read_slice(src_ptr, len as u64));
            sp.write_u64(4, GoValue::Number(len as f64).encode());
            sp.write_u8(5, 1);
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
    sp.write_u64(4, GoValue::Null.encode());
    sp.write_u8(5, 0);
}

unsafe fn value_call_impl(sp: &mut GoStack) -> Result<GoValue, String> {
    let object = interpret_value(sp.read_u64(0));
    let method_name_ptr = sp.read_u64(1);
    let method_name_len = sp.read_u64(2);
    let method_name = read_slice(method_name_ptr, method_name_len);
    let args_ptr = sp.read_u64(3);
    let args_len = sp.read_u64(4);
    let args = read_value_slice(args_ptr, args_len);
    if object == InterpValue::Ref(GO_ID) && &method_name == b"_makeFuncWrapper" {
        let id = args.get(0).ok_or_else(|| {
            format!(
                "Go attempting to call Go._makeFuncWrapper with bad args {:?}",
                args,
            )
        })?;
        let ref_id =
            DynamicObjectPool::singleton().insert(DynamicObject::FunctionWrapper(*id, object));
        Ok(GoValue::Function(ref_id))
    } else if object == InterpValue::Ref(FS_ID) && &method_name == b"write" {
        let args_len = std::cmp::min(6, args.len());
        if let &[InterpValue::Number(fd), InterpValue::Ref(buf_id), InterpValue::Number(offset), InterpValue::Number(length), InterpValue::Ref(NULL_ID), InterpValue::Ref(callback_id)] =
            &args.as_slice()[..args_len]
        {
            let object_pool = DynamicObjectPool::singleton();
            let buf = match object_pool.get(buf_id) {
                Some(DynamicObject::Uint8Array(x)) => x,
                x => {
                    return Err(format!(
                        "Go attempting to call fs.write with bad buffer {:?}",
                        x,
                    ))
                }
            };
            let (func_id, this) = match object_pool.get(callback_id) {
                Some(DynamicObject::FunctionWrapper(f, t)) => (f, t),
                x => {
                    return Err(format!(
                        "Go attempting to call fs.write with bad buffer {:?}",
                        x,
                    ))
                }
            };
            let mut offset = offset as usize;
            let mut length = length as usize;
            if offset > buf.len() {
                eprintln!(
                    "Go attempting to call fs.write with offset {} >= buf.len() {}",
                    offset,
                    buf.len(),
                );
                offset = buf.len();
            }
            if offset + length > buf.len() {
                eprintln!(
                    "Go attempting to call fs.write with offset {} + length {} >= buf.len() {}",
                    offset,
                    length,
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
                eprintln!("Go attempting to write to unknown FD {}", fd);
            }

            PENDING_EVENT = Some(PendingEvent {
                id: *func_id,
                this: *this,
                args: vec![
                    GoValue::Null,                  // no error
                    GoValue::Number(length as f64), // amount written
                ],
            });
            wavm_guest_call__resume();

            *sp = GoStack(wavm_guest_call__getsp());
            Ok(GoValue::Null)
        } else {
            Err(format!(
                "Go attempting to call fs.write with bad args {:?}",
                args
            ))
        }
    } else {
        Err(format!(
            "Go attempting to call unknown method {:?} . {}",
            object,
            String::from_utf8_lossy(&method_name),
        ))
    }
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueCall(mut sp: GoStack) {
    match value_call_impl(&mut sp) {
        Ok(val) => {
            sp.write_u64(6, val.encode());
            sp.write_u8(7, 1);
        }
        Err(err) => {
            eprintln!("{}", err);
            sp.write_u64(6, GoValue::Null.encode());
            sp.write_u8(7, 0);
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueSet(sp: GoStack) {
    let source = interpret_value(sp.read_u64(0));
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let new_value = interpret_value(sp.read_u64(3));
    let field = read_slice(field_ptr, field_len);
    if source == InterpValue::Ref(GO_ID)
        && &field == b"_pendingEvent"
        && new_value == InterpValue::Ref(NULL_ID)
    {
        PENDING_EVENT = None;
        return;
    }
    let pool = DynamicObjectPool::singleton();
    if let InterpValue::Ref(id) = source {
        let source = pool.get(id);
        if let Some(DynamicObject::PendingEvent(_)) = source {
            if field == b"result" {
                return;
            }
        }
    }
    eprintln!(
        "Go attempted to set unsupported value {:?} field {} to {:?}",
        source,
        String::from_utf8_lossy(&field),
        new_value,
    );
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueLength(sp: GoStack) {
    let source = interpret_value(sp.read_u64(0));
    let pool = DynamicObjectPool::singleton();
    let source = match source {
        InterpValue::Ref(x) => pool.get(x),
        InterpValue::Number(_) => None,
    };
    let len = match source {
        Some(DynamicObject::Uint8Array(x)) => Some(x.len()),
        Some(DynamicObject::ValueArray(x)) => Some(x.len()),
        _ => None,
    };
    if let Some(len) = len {
        sp.write_u64(1, len as u64);
    } else {
        eprintln!(
            "Go attempted to get length of unsupported value {:?}",
            source,
        );
        sp.write_u64(1, 0);
    }
}

unsafe fn value_index_impl(sp: GoStack) -> Result<GoValue, String> {
    let pool = DynamicObjectPool::singleton();
    let source = match interpret_value(sp.read_u64(0)) {
        InterpValue::Ref(x) => pool.get(x),
        InterpValue::Number(x) => return Err(format!("Go attempted to index into number {}", x)),
    };
    let index = usize::try_from(sp.read_u64(1)).map_err(|e| format!("{:?}", e))?;
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

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueIndex(sp: GoStack) {
    match value_index_impl(sp) {
        Ok(v) => sp.write_u64(2, v.encode()),
        Err(e) => {
            eprintln!("{}", e);
            sp.write_u64(2, GoValue::Null.encode());
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn wavm__go_after_run() {
    // TODO
    while let Some(_pending_event) = None::<u32> {
        wavm_guest_call__resume();
    }
}
