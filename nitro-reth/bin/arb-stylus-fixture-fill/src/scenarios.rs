//! Per-fixture scenario definitions: which deploy / activate / invoke
//! operations a given Stylus fixture exercises.

use alloy_primitives::U256;

use crate::wat_sources;

#[derive(Debug, Clone)]
pub enum WasmSource {
    /// Read `_wat/<stem>.wat`, compile via `wat::parse_str`.
    WatFile { stem: String },
    /// Inline WAT source.
    Inline { wat: String },
    /// Read `_wat/<stem>.wat`, then pad WASM to at least `target_size` bytes
    /// (post-compression should still exceed the contract-size threshold).
    Padded { stem: String, target_size: usize },
}

impl WasmSource {
    pub fn cache_key(&self) -> String {
        match self {
            WasmSource::WatFile { stem } => format!("file:{stem}"),
            WasmSource::Inline { wat } => format!("inline:{}", wat.len()),
            WasmSource::Padded { stem, target_size } => format!("pad:{stem}:{target_size}"),
        }
    }
}

#[derive(Debug, Clone)]
pub enum Operation {
    /// Deploy a Stylus program; `role` keys lookup for later invokes.
    DeployStylus { role: String, source: WasmSource },
    /// Deploy plain EVM bytecode (e.g. proxy contract for delegatecall tests).
    DeployEvm { role: String, runtime_code: Vec<u8> },
    /// Call ArbWasm.activateProgram(target).
    Activate { target_role: String },
    /// SignedL2Tx call to a previously deployed program / EOA.
    Invoke {
        target_role: String,
        calldata: Vec<u8>,
        value: U256,
    },
    /// ArbOwner.setMaxStylusContractFragments(maxFragments).
    ArbOwnerSetMaxStylusFragments { max_fragments: u8 },
}

pub struct Scenario {
    pub eip1559: bool,
    pub ops: Vec<Operation>,
}

const DEFAULT_ROLE: &str = "program";
const MULTICALL_ROLE: &str = "multicall";
const STORAGE_TARGET_ROLE: &str = "storage_target";
const CREATE_FACTORY_ROLE: &str = "create_factory";
const EVM_PROXY_ROLE: &str = "evm_proxy";

pub fn for_fixture(name: &str) -> Option<Scenario> {
    match name {
        "cache_activate_invoke_arbos32" => Some(activate_invoke(DEFAULT_ROLE, "hostio_keccak", true)),
        "cache_recent_wasms_block_local_arbos60" => {
            Some(activate_invoke(DEFAULT_ROLE, "hostio_keccak", true))
        }
        "cache_lru_eviction_arbos60" => Some(lru_eviction()),

        "contract_limit_v59_max_size" => Some(oversized_program(true)),
        "contract_limit_v60_increased" => Some(oversized_program_v60()),

        // hostio_emit_log_*: deploy emit_log + activate + invoke with topic+payload calldata.
        "hostio_emit_log_zero_topics" => Some(hostio_emit_log(0, 0, true)),
        "hostio_emit_log_two_topics" => Some(hostio_emit_log(2, 0, true)),
        "hostio_emit_log_four_topics" => Some(hostio_emit_log(4, 0, true)),

        "hostio_keccak_short" => Some(hostio_keccak(32, true)),
        "hostio_keccak_aligned" => Some(hostio_keccak(256, true)),
        "hostio_keccak_long" => Some(hostio_keccak(2048, true)),

        "hostio_storage_load_bytes32_arbos32" => Some(hostio_storage_load(true)),
        "hostio_storage_load_bytes32_arbos60" => Some(hostio_storage_load(true)),
        "hostio_storage_load_bytes32_warm" => Some(hostio_storage_load_warm()),

        "hostio_storage_store_bytes32_arbos32" => Some(hostio_storage_store(0x07, 1, true)),
        "hostio_storage_store_bytes32_arbos60" => Some(hostio_storage_store(0x07, 2, true)),
        "hostio_storage_store_bytes32_clear" => Some(hostio_storage_store(0x07, 0, true)),

        "hostio_msg_sender_eoa" => Some(hostio_msg_sender_eoa()),
        "hostio_msg_sender_call" => Some(hostio_msg_sender_via_proxy(false)),
        "hostio_msg_sender_delegatecall" => Some(hostio_msg_sender_via_proxy(true)),

        "subcall_wasm_call_wasm" => Some(subcall_wasm_pair(0)),
        "subcall_wasm_delegatecall_wasm" => Some(subcall_wasm_pair(1)),
        "subcall_wasm_staticcall_wasm" => Some(subcall_wasm_pair(2)),
        "subcall_wasm_call_evm" => Some(subcall_wasm_call_evm()),
        "subcall_wasm_create_wasm" => Some(subcall_create()),
        "subcall_wasm_create2_wasm" => Some(subcall_create2()),
        "subcall_reentrancy" => Some(subcall_reentrancy()),

        _ => None,
    }
}

fn activate_invoke(role: &str, stem: &str, eip1559: bool) -> Scenario {
    Scenario {
        eip1559,
        ops: vec![
            Operation::DeployStylus {
                role: role.to_string(),
                source: WasmSource::WatFile {
                    stem: stem.to_string(),
                },
            },
            Operation::Activate {
                target_role: role.to_string(),
            },
            Operation::Invoke {
                target_role: role.to_string(),
                calldata: vec![0xab; 8],
                value: U256::ZERO,
            },
            Operation::Invoke {
                target_role: role.to_string(),
                calldata: vec![0xab; 8],
                value: U256::ZERO,
            },
        ],
    }
}

fn lru_eviction() -> Scenario {
    let mut ops = Vec::new();
    ops.push(Operation::ArbOwnerSetMaxStylusFragments { max_fragments: 2 });
    for letter in ["a", "b", "c"] {
        let role = format!("program_{letter}");
        ops.push(Operation::DeployStylus {
            role: role.clone(),
            source: WasmSource::Inline {
                wat: wat_sources::distinct_keccak_program(letter),
            },
        });
        ops.push(Operation::Activate {
            target_role: role.clone(),
        });
        ops.push(Operation::Invoke {
            target_role: role,
            calldata: vec![0xab; 8],
            value: U256::ZERO,
        });
    }
    // Reinvoke A — cache miss after B/C displaced it.
    ops.push(Operation::Invoke {
        target_role: "program_a".to_string(),
        calldata: vec![0xab; 8],
        value: U256::ZERO,
    });
    Scenario { eip1559: true, ops }
}

fn oversized_program(must_revert: bool) -> Scenario {
    let _ = must_revert;
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::Padded {
                    stem: "hostio_keccak".to_string(),
                    target_size: 100_000,
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
        ],
    }
}

fn oversized_program_v60() -> Scenario {
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::Padded {
                    stem: "hostio_keccak".to_string(),
                    target_size: 100_000,
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: vec![0xab; 8],
                value: U256::ZERO,
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: vec![0xab; 8],
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_emit_log(topics: u8, payload_len: usize, eip1559: bool) -> Scenario {
    let mut calldata = Vec::with_capacity(1 + (topics as usize) * 32 + payload_len);
    calldata.push(topics);
    for i in 0..topics {
        let mut topic = [0u8; 32];
        topic[31] = (i + 1) as u8;
        calldata.extend_from_slice(&topic);
    }
    for i in 0..payload_len {
        calldata.push((i & 0xff) as u8);
    }
    Scenario {
        eip1559,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_emit_log".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_keccak(byte_count: usize, eip1559: bool) -> Scenario {
    let calldata = vec![0x55; byte_count];
    Scenario {
        eip1559,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_keccak".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_storage_load(eip1559: bool) -> Scenario {
    let mut key = [0u8; 32];
    key[31] = 0x07;
    Scenario {
        eip1559,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_storage_load_bytes32".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: key.to_vec(),
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_storage_load_warm() -> Scenario {
    let mut key = [0u8; 32];
    key[31] = 0x07;
    // Two consecutive invokes to exercise the warm path on the second call.
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_storage_load_bytes32".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: key.to_vec(),
                value: U256::ZERO,
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: key.to_vec(),
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_storage_store(slot_byte: u8, value_byte: u8, eip1559: bool) -> Scenario {
    let mut buf = vec![0u8; 64];
    buf[31] = slot_byte;
    buf[63] = value_byte;
    Scenario {
        eip1559,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_storage_store_bytes32".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: buf,
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_msg_sender_eoa() -> Scenario {
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_msg_sender".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: DEFAULT_ROLE.to_string(),
                calldata: Vec::new(),
                value: U256::ZERO,
            },
        ],
    }
}

fn hostio_msg_sender_via_proxy(delegate: bool) -> Scenario {
    let runtime = if delegate {
        wat_sources::evm_proxy_delegatecall_runtime()
    } else {
        wat_sources::evm_proxy_call_runtime()
    };
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_msg_sender".to_string(),
                },
            },
            Operation::Activate {
                target_role: DEFAULT_ROLE.to_string(),
            },
            Operation::DeployEvm {
                role: EVM_PROXY_ROLE.to_string(),
                runtime_code: runtime,
            },
            Operation::Invoke {
                target_role: EVM_PROXY_ROLE.to_string(),
                calldata: Vec::new(),
                value: U256::ZERO,
            },
        ],
    }
}

fn subcall_wasm_pair(op_byte: u8) -> Scenario {
    let mut calldata = vec![op_byte];
    calldata.extend_from_slice(&[0u8; 19]);
    calldata.push(0xAA);
    calldata.extend_from_slice(&[0u8; 8]);
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: STORAGE_TARGET_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_storage_store_bytes32".to_string(),
                },
            },
            Operation::Activate {
                target_role: STORAGE_TARGET_ROLE.to_string(),
            },
            Operation::DeployStylus {
                role: MULTICALL_ROLE.to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::multicall_wat().to_string(),
                },
            },
            Operation::Activate {
                target_role: MULTICALL_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: MULTICALL_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn subcall_wasm_call_evm() -> Scenario {
    let mut calldata = vec![0u8];
    calldata.extend_from_slice(&[0u8; 19]);
    calldata.push(0x69);
    calldata.extend_from_slice(&[0u8; 8]);
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: MULTICALL_ROLE.to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::multicall_wat().to_string(),
                },
            },
            Operation::Activate {
                target_role: MULTICALL_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: MULTICALL_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn subcall_create() -> Scenario {
    let mut calldata = vec![3u8];
    calldata.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    calldata.extend_from_slice(&[0xEF, 0xF0, 0x00, 0x00]);
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: CREATE_FACTORY_ROLE.to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::create_factory_wat().to_string(),
                },
            },
            Operation::Activate {
                target_role: CREATE_FACTORY_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: CREATE_FACTORY_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn subcall_create2() -> Scenario {
    let mut calldata = vec![4u8];
    let mut salt = [0u8; 32];
    salt[31] = 0x42;
    calldata.extend_from_slice(&salt);
    calldata.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    calldata.extend_from_slice(&[0xEF, 0xF0, 0x00, 0x00]);
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: CREATE_FACTORY_ROLE.to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::create_factory_wat().to_string(),
                },
            },
            Operation::Activate {
                target_role: CREATE_FACTORY_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: CREATE_FACTORY_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}

fn subcall_reentrancy() -> Scenario {
    let mut calldata = vec![0u8];
    let mut addr = [0u8; 20];
    addr[19] = 0x42;
    calldata.extend_from_slice(&addr);
    calldata.extend_from_slice(&[0u8; 8]);
    Scenario {
        eip1559: true,
        ops: vec![
            Operation::DeployStylus {
                role: MULTICALL_ROLE.to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::multicall_wat().to_string(),
                },
            },
            Operation::Activate {
                target_role: MULTICALL_ROLE.to_string(),
            },
            Operation::Invoke {
                target_role: MULTICALL_ROLE.to_string(),
                calldata,
                value: U256::ZERO,
            },
        ],
    }
}
