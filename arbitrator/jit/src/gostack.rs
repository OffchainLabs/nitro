// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    syscall::{JsRuntimeState, JsValue},
    wavmio::Bytes32,
    Opts,
};

use eyre::{bail, Result, WrapErr};
use parking_lot::{Mutex, MutexGuard};
use rand_pcg::Pcg32;
use sha3::{Digest, Keccak256};
use thiserror::Error;
use wasmer::{Memory, MemoryView, WasmPtr, WasmerEnv};

use std::{
    collections::{BTreeMap, BTreeSet, BinaryHeap},
    fs::File,
    io::{self, BufReader, ErrorKind, Read},
    net::TcpStream,
    ops::Deref,
    sync::Arc,
};

#[derive(Clone)]
pub struct GoStack {
    start: u32,
    memory: Memory,
}

#[allow(dead_code)]
impl GoStack {
    pub fn new(start: u32, env: &WasmEnvArc) -> (Self, MutexGuard<WasmEnv>) {
        let memory = env.lock().memory.clone().unwrap();
        let sp = Self { start, memory };
        let env = env.lock();
        (sp, env)
    }

    /// Returns the memory size, in bytes.
    /// note: wasmer measures memory in 65536-byte pages.
    pub fn memory_size(&self) -> u64 {
        self.memory.size().0 as u64 * 65536
    }

    fn offset(&self, arg: u32) -> u32 {
        self.start + (arg + 1) * 8
    }

    pub fn read_u8(&self, arg: u32) -> u8 {
        self.read_u8_ptr(self.offset(arg))
    }

    pub fn read_u32(&self, arg: u32) -> u32 {
        self.read_u32_ptr(self.offset(arg))
    }

    pub fn read_u64(&self, arg: u32) -> u64 {
        self.read_u64_ptr(self.offset(arg))
    }

    pub fn read_u8_ptr(&self, ptr: u32) -> u8 {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().get()
    }

    pub fn read_u32_ptr(&self, ptr: u32) -> u32 {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().get()
    }

    pub fn read_u64_ptr(&self, ptr: u32) -> u64 {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().get()
    }

    pub fn write_u8(&self, arg: u32, x: u8) {
        self.write_u8_ptr(self.offset(arg), x);
    }

    pub fn write_u32(&self, arg: u32, x: u32) {
        self.write_u32_ptr(self.offset(arg), x);
    }

    pub fn write_u64(&self, arg: u32, x: u64) {
        self.write_u64_ptr(self.offset(arg), x);
    }

    pub fn write_u8_ptr(&self, ptr: u32, x: u8) {
        let ptr: WasmPtr<u8> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().set(x);
    }

    pub fn write_u32_ptr(&self, ptr: u32, x: u32) {
        let ptr: WasmPtr<u32> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().set(x);
    }

    pub fn write_u64_ptr(&self, ptr: u32, x: u64) {
        let ptr: WasmPtr<u64> = WasmPtr::new(ptr);
        ptr.deref(&self.memory).unwrap().set(x);
    }

    pub fn read_slice(&self, ptr: u64, len: u64) -> Vec<u8> {
        let ptr = u32::try_from(ptr).expect("Go pointer not a u32") as usize;
        let len = u32::try_from(len).expect("length isn't a u32") as usize;
        unsafe { self.memory.data_unchecked()[ptr..ptr + len].to_vec() }
    }

    pub fn write_slice(&self, ptr: u64, src: &[u8]) {
        let ptr = u32::try_from(ptr).expect("Go pointer not a u32");
        let view: MemoryView<u8> = self.memory.view();
        let view = view.subarray(ptr, ptr + src.len() as u32);
        unsafe { view.copy_from(src) }
    }

    pub fn read_value_slice(&self, mut ptr: u64, len: u64) -> Vec<JsValue> {
        let mut values = Vec::new();
        for _ in 0..len {
            let p = u32::try_from(ptr).expect("Go pointer not a u32");
            values.push(JsValue::new(self.read_u64_ptr(p)));
            ptr += 8;
        }
        values
    }
}

#[derive(Error, Debug)]
pub enum Escape {
    #[error("program exited with status code `{0}`")]
    Exit(u32),
    #[error("jit failed with `{0}`")]
    Failure(String),
    #[error("hostio failed with `{0}`")]
    HostIO(String),
    #[error("hostio socket failed with `{0}`")]
    SocketError(#[from] io::Error),
}

pub type MaybeEscape = Result<(), Escape>;

impl Escape {
    pub fn exit(code: u32) -> MaybeEscape {
        Err(Self::Exit(code))
    }

    pub fn hostio(message: &str) -> MaybeEscape {
        Err(Self::HostIO(message.to_owned()))
    }
}

pub type Inbox = BTreeMap<u64, Vec<u8>>;
pub type Oracle = BTreeMap<[u8; 32], Vec<u8>>;

#[derive(Default)]
pub struct WasmEnv {
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Go's general runtime state
    pub go_state: GoRuntimeState,
    /// The state of Go's js runtime
    pub js_state: JsRuntimeState,
    /// An ordered list of the 8-byte globals
    pub small_globals: Vec<u64>,
    /// An ordered list of the 32-byte globals
    pub large_globals: Vec<Bytes32>,
    /// An oracle allowing the prover to reverse keccak256
    pub preimages: Oracle,
    /// The sequencer inbox's messages
    pub sequencer_messages: Inbox,
    /// The delayed inbox's messages
    pub delayed_messages: Inbox,
    /// The first inbox message number knowably out of bounds
    pub first_too_far: u64,
    /// The purpose and connections of this process
    pub process: ProcessEnv,
}

#[derive(Clone, Default, WasmerEnv)]
pub struct WasmEnvArc(Arc<Mutex<WasmEnv>>);

impl Deref for WasmEnvArc {
    type Target = Mutex<WasmEnv>;
    fn deref(&self) -> &Self::Target {
        &*self.0
    }
}

impl WasmEnvArc {
    pub fn cli(opts: &Opts) -> Result<Self> {
        let mut env = WasmEnv::default();

        let mut inbox_position = opts.inbox_position;
        let mut delayed_position = opts.delayed_inbox_position;

        for path in &opts.inbox {
            let mut msg = vec![];
            File::open(path)?.read_to_end(&mut msg)?;
            env.sequencer_messages.insert(inbox_position, msg);
            inbox_position += 1;
        }
        for path in &opts.delayed_inbox {
            let mut msg = vec![];
            File::open(path)?.read_to_end(&mut msg)?;
            env.delayed_messages.insert(delayed_position, msg);
            delayed_position += 1;
        }

        if let Some(path) = &opts.preimages {
            let mut file = BufReader::new(File::open(path)?);
            let mut preimages = Vec::new();
            let filename = path.to_string_lossy();
            loop {
                let mut size_buf = [0u8; 8];
                match file.read_exact(&mut size_buf) {
                    Ok(()) => {}
                    Err(err) if err.kind() == ErrorKind::UnexpectedEof => break,
                    Err(err) => bail!("Failed to parse {filename}: {}", err),
                }
                let size = u64::from_le_bytes(size_buf) as usize;
                let mut buf = vec![0u8; size];
                file.read_exact(&mut buf)?;
                preimages.push(buf);
            }
            for preimage in preimages {
                let mut hasher = Keccak256::new();
                hasher.update(&preimage);
                let hash = hasher.finalize().into();
                env.preimages.insert(hash, preimage);
            }
        }

        fn parse_hex(arg: &Option<String>, name: &str) -> Result<Bytes32> {
            match arg {
                Some(arg) => {
                    let mut arg = arg.as_str();
                    if arg.starts_with("0x") {
                        arg = &arg[2..];
                    }
                    let mut bytes32 = Bytes32::default();
                    hex::decode_to_slice(arg, &mut bytes32)
                        .wrap_err_with(|| format!("failed to parse {} contents", name))?;
                    Ok(bytes32)
                }
                None => Ok(Bytes32::default()),
            }
        }

        let last_block_hash = parse_hex(&opts.last_block_hash, "--last-block-hash")?;
        let last_send_root = parse_hex(&opts.last_send_root, "--last-send-root")?;
        env.small_globals = vec![opts.inbox_position, opts.position_within_message];
        env.large_globals = vec![last_block_hash, last_send_root];
        Ok(Self(Arc::new(Mutex::new(env))))
    }
}

#[derive(Clone)]
pub struct GoRuntimeState {
    /// An increasing clock used when Go asks for time, measured in nanoseconds
    pub time: u64,
    /// The amount of time advanced each check. Currently 10 milliseconds
    pub time_interval: u64,
    /// The state of Go's timeouts
    pub timeouts: TimeoutState,
    /// Deterministic source of random data
    pub rng: Pcg32,
}

impl Default for GoRuntimeState {
    fn default() -> Self {
        Self {
            time: 0,
            time_interval: 10_000_000,
            timeouts: TimeoutState::default(),
            rng: Pcg32::new(0xcafef00dd15ea5e5, 0xa02bdbf7bb3c0a7),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TimeoutInfo {
    pub time: u64,
    pub id: u32,
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

#[derive(Default, Clone, Debug)]
pub struct TimeoutState {
    /// Contains tuples of (time, id)
    pub times: BinaryHeap<TimeoutInfo>,
    pub pending_ids: BTreeSet<u32>,
    pub next_id: u32,
}

#[derive(Default)]
pub struct ProcessEnv {
    /// Whether to create child processes to handle execution
    pub forks: bool,
    /// Mechanism for asking for preimages and returning results
    pub socket: Option<(TcpStream, BufReader<TcpStream>)>,
}
