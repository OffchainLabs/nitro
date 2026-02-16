use bytes::{Bytes, BytesMut};
use std::{
    collections::{BTreeMap, HashMap},
    io::Read,
};
use validation::{UserWasm, ValidationInput};

/// This groups components in Arbitrum's WasmEnv that come from FileData.
/// It helps us maintain a clear separation between Arbitrum inputs, and other
/// WASM required data.
/// For details of each field other than FileData, please refer to:
/// https://github.com/OffchainLabs/nitro/blob/11255c6177d50c1ebc43dbcd4bb5f6e9fae5383a/arbitrator/jit/src/machine.rs#L193-L214
#[derive(rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)]
pub struct Input {
    pub small_globals: [u64; 2],
    pub large_globals: [[u8; 32]; 2],
    pub preimages: Preimages,
    pub module_asms: HashMap<[u8; 32], ModuleAsm>,
    pub sequencer_messages: Inbox,
    pub delayed_messages: Inbox,
}

// SP1 has additional alignment requirements, we have to decompress the data
// into aligned bytes
pub fn decompress_aligned(user_wasm: &UserWasm) -> ModuleAsm {
    let data = user_wasm.as_ref();
    // This is less ideal but until one of the following happens, we
    // will have to stick with it:
    // * Allocator allocates aligned memory
    // * Bytes add alignment options
    // * Wasmer's Module does not simply accept `IntoBytes` trait.
    let mut buffer = BytesMut::zeroed(data.len() + 7);
    let p = buffer.as_ptr() as usize;
    let aligned_p = (p + 7) / 8 * 8;
    let offset = aligned_p - p;
    buffer[offset..offset + data.len()].copy_from_slice(&data);
    let bytes = buffer.freeze();
    bytes.slice(offset..offset + data.len())
}

impl Input {
    /// This takes hint from `arbitrator/jit/src/prepare.rs`
    pub fn from_file_data(data: ValidationInput) -> eyre::Result<Self> {
        let large_globals = [data.start_state.block_hash.0, data.start_state.send_root.0];
        let small_globals = [data.start_state.batch, data.start_state.pos_in_batch];

        let mut sequencer_messages = Inbox::default();
        for batch_info in data.batch_info.iter() {
            sequencer_messages.insert(batch_info.number, batch_info.data.clone());
        }

        let mut delayed_messages = Inbox::default();
        if data.delayed_msg_nr != 0 && !data.delayed_msg.is_empty() {
            delayed_messages.insert(data.delayed_msg_nr, data.delayed_msg.clone());
        }

        let mut preimages = Preimages::default();
        for (preimage_ty, inner_map) in data.preimages {
            let map = preimages.entry(preimage_ty as u8).or_default();
            for (hash, preimage) in inner_map {
                map.insert(*hash, preimage);
            }
        }

        let mut module_asms = HashMap::default();
        if let Some(user_wasms) = data.user_wasms.get(&local_target()) {
            for (module_hash, module_asm) in user_wasms.iter() {
                module_asms.insert(**module_hash, decompress_aligned(&module_asm));
            }
        }

        Ok(Self {
            small_globals,
            large_globals,
            preimages,
            module_asms,
            sequencer_messages,
            delayed_messages,
        })
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

fn local_target() -> String {
    "rv64".to_string()
}

pub type Inbox = BTreeMap<u64, Vec<u8>>;
pub type Preimages = BTreeMap<u8, BTreeMap<[u8; 32], Vec<u8>>>;
pub type ModuleAsm = Bytes;
