use rand::RngCore;
use rand_pcg::Pcg32;
use std::{convert::TryFrom, io::Write};

#[allow(dead_code)]
extern "C" {
    fn wavm_caller_module_memory_load8(ptr: usize) -> u8;
    fn wavm_caller_module_memory_load32(ptr: usize) -> u32;
    fn wavm_caller_module_memory_store8(ptr: usize, val: u8);
    fn wavm_caller_module_memory_store32(ptr: usize, val: u32);
    fn wavm_guest_resume();
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
        let lower = wavm_caller_module_memory_load32(self.0 + (offset + 1) * 8);
        let upper = wavm_caller_module_memory_load32(self.0 + (offset + 1) * 8 + 4);
        lower as u64 | ((upper as u64) << 32)
    }

    unsafe fn write_u64(self, offset: usize, x: u64) {
        wavm_caller_module_memory_store32(self.0 + (offset + 1) * 8, x as u32);
        wavm_caller_module_memory_store32(self.0 + (offset + 1) * 8 + 4, (x >> 32) as u32);
    }
}

unsafe fn read_slice(ptr: u64, mut len: u64) -> Vec<u8> {
    let mut data = Vec::new();
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
    go__syscall_js_valueSet,
    go__syscall_js_valueIndex,
    go__syscall_js_valueSetIndex,
    go__syscall_js_valueCall,
    go__syscall_js_valueNew,
    go__syscall_js_valueLength,
    go__syscall_js_valuePrepareString,
    go__syscall_js_valueLoadString,
    go__syscall_js_copyBytesToJS,
);

const NULL_ID: u32 = 2;
const GLOBAL_ID: u32 = 5;

#[derive(Clone, Copy, Debug)]
#[allow(dead_code)]
enum GoValue {
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
            GoValue::Number(mut f) => {
                // Canonicalize NaNs so they don't collide with other value types
                if f.is_nan() {
                    f = f64::NAN;
                }
                return f.to_bits();
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

const OBJECT_ID: u32 = 100;
const ARRAY_ID: u32 = 101;
const PROCESS_ID: u32 = 102;
const FS_ID: u32 = 103;
const UINT8_ARRAY_ID: u32 = 104;

const FS_CONSTANTS_ID: u32 = 200;

fn get_value(source: u32, field: &[u8]) -> GoValue {
    if source == GLOBAL_ID {
        if field == b"Object" {
            return GoValue::Function(OBJECT_ID);
        } else if field == b"Array" {
            return GoValue::Function(ARRAY_ID);
        } else if field == b"process" {
            return GoValue::Object(PROCESS_ID);
        } else if field == b"fs" {
            return GoValue::Object(FS_ID);
        } else if field == b"Uint8Array" {
            return GoValue::Function(UINT8_ARRAY_ID);
        }
    } else if source == FS_ID {
        if field == b"constants" {
            return GoValue::Object(FS_CONSTANTS_ID);
        }
    } else if source == FS_CONSTANTS_ID {
        if matches!(
            field,
            b"O_WRONLY" | b"O_RDWR" | b"O_CREAT" | b"O_TRUNC" | b"O_APPEND" | b"O_EXCL"
        ) {
            return GoValue::Number(-1.);
        }
    }
    let s = String::from_utf8_lossy(&field);
    unimplemented!("Go attempting to access JS value {} field {}", source, s)
}

#[no_mangle]
pub unsafe extern "C" fn go__syscall_js_valueGet(sp: GoStack) {
    let source = sp.read_u32(0);
    let field_ptr = sp.read_u64(1);
    let field_len = sp.read_u64(2);
    let field = read_slice(field_ptr, field_len);
    let value = get_value(source, &field);
    sp.write_u64(3, value.encode());
}

#[no_mangle]
pub unsafe extern "C" fn wavm__go_after_run() {
    // TODO
    while let Some(_pending_event) = None::<u32> {
        wavm_guest_resume();
    }
}
