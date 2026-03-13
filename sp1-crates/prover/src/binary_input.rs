use bytes::{Bytes, BytesMut};
use std::{collections::HashMap, io::Read};
use validation::{UserWasm, ValidationInput, ValidationRequest};

#[derive(rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
pub struct Input {
    pub small_globals: [u64; 2],
    pub large_globals: [[u8; 32]; 2],
    pub preimages: validation::Preimages,
    pub module_asms: HashMap<[u8; 32], ModuleAsm>,
    pub sequencer_messages: validation::Inbox,
    pub delayed_messages: validation::Inbox,
}

// SP1 has additional alignment requirements, we have to decompress the data
// into aligned bytes
pub fn decompress_aligned(user_wasm: &UserWasm) -> ModuleAsm {
    // This is less ideal but until one of the following happens, we
    // will have to stick with it:
    // * Allocator allocates aligned memory
    // * Bytes add alignment options
    // * Wasmer's Module does not simply accept `IntoBytes` trait.
    let data = user_wasm.as_vec();
    let mut buffer = BytesMut::zeroed(data.len() + 7);
    let p = buffer.as_ptr() as usize;
    let aligned_p = (p + 7) / 8 * 8;
    let offset = aligned_p - p;
    buffer[offset..offset + data.len()].copy_from_slice(&data);
    let bytes = buffer.freeze();
    bytes.slice(offset..offset + data.len())
}

pub fn decompress_aligned_from_vec(data: Vec<u8>) -> ModuleAsm {
    let mut buffer = BytesMut::zeroed(data.len() + 7);
    let p = buffer.as_ptr() as usize;
    let aligned_p = (p + 7) / 8 * 8;
    let offset = aligned_p - p;
    buffer[offset..offset + data.len()].copy_from_slice(&data);
    let bytes = buffer.freeze();
    bytes.slice(offset..offset + data.len())
}

impl Input {
    pub fn from_request(req: &ValidationRequest) -> Self {
        let base = ValidationInput::from_request(req, "rv64");
        let module_asms = base
            .module_asms
            .into_iter()
            .map(|(hash, data)| (hash, decompress_aligned_from_vec(data)))
            .collect();
        Self {
            small_globals: base.small_globals,
            large_globals: base.large_globals,
            preimages: base.preimages,
            sequencer_messages: base.sequencer_messages,
            delayed_messages: base.delayed_messages,
            module_asms,
        }
    }

    // This utilizes binary format from rykv
    pub fn from_reader<R: Read>(mut reader: R) -> Result<Self, String> {
        let mut s = Vec::new();
        reader
            .read_to_end(&mut s)
            .map_err(|e| format!("IO Error: {e:?}"))?;
        let archived = rkyv::access::<ArchivedInput, rkyv::rancor::Error>(&s[..])
            .map_err(|e| format!("rkyv access error: {e:?}"))?;
        rkyv::deserialize::<Input, rkyv::rancor::Error>(archived)
            .map_err(|e| format!("rkyv deserialize error: {e:?}"))
    }
}

pub type ModuleAsm = Bytes;
