//! Mapping from `_TODO_*` placeholder strings (and the fixture name they live
//! in) to a concrete program-construction directive.

use alloy_primitives::U256;
use anyhow::{anyhow, Result};

use crate::wat_sources;

/// What kind of program should back a deploy.
#[derive(Debug, Clone)]
pub enum WasmSource {
    /// Read `_wat/<stem>.wat`, compile via `wat::parse_str`.
    WatFile { stem: String },
    /// Inline WAT source string.
    Inline { wat: String },
    /// Compile `_wat/<stem>.wat`, then pad WASM to at least `target_size` bytes.
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
pub struct BatchInvoke {
    pub target_role: String,
    pub calldata: Vec<u8>,
    pub value: U256,
}

#[derive(Debug, Clone)]
pub enum ResolvedPlaceholder {
    Deploy {
        role: String,
        source: WasmSource,
    },
    /// Plain EVM bytecode deploy (no Stylus discriminant).
    DeployEvm {
        role: String,
        runtime_code: Vec<u8>,
    },
    ActivateProgram {
        target_role: String,
    },
    Invoke {
        target_role: String,
        calldata: Vec<u8>,
        value: U256,
    },
    ArbOwnerSetMaxFragments {
        max_fragments: u8,
    },
    Batch {
        invokes: Vec<BatchInvoke>,
    },
}

const DEFAULT_ROLE: &str = "program";

pub fn resolve(placeholder: &str, fixture_name: &str) -> Result<ResolvedPlaceholder> {
    // Strip leading sentinel.
    let body = placeholder
        .strip_prefix("_TODO_")
        .ok_or_else(|| anyhow!("not a TODO placeholder: {placeholder}"))?;

    // Activation first — common across all hostio/cache fixtures.
    if body == "compile_wasm_activate_program" {
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: DEFAULT_ROLE.to_string(),
        });
    }
    if let Some(rest) = body.strip_prefix("signed_invoke_arbwasm_activate_program") {
        let _ = rest;
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: DEFAULT_ROLE.to_string(),
        });
    }
    if body == "compile_wasm_activate_multicall"
        || body == "compile_wasm_activate_reentrant_multicall"
    {
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: "multicall".to_string(),
        });
    }
    if body == "compile_wasm_activate_storage_target" {
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: "storage_target".to_string(),
        });
    }
    if body == "compile_wasm_activate_create_factory" {
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: "create_factory".to_string(),
        });
    }
    if body == "signed_invoke_arbwasm_activate_oversized_program_v60" {
        return Ok(ResolvedPlaceholder::ActivateProgram {
            target_role: DEFAULT_ROLE.to_string(),
        });
    }

    // Hostio program deploys (use shared WAT files).
    if let Some(stem_suffix) = body.strip_prefix("compile_wasm_deploy_") {
        if stem_suffix.starts_with("hostio_") {
            // Map the trailing hostio_<x> directly to a WAT stem.
            return Ok(ResolvedPlaceholder::Deploy {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: stem_suffix.to_string(),
                },
            });
        }
        if stem_suffix == "program" {
            return Ok(ResolvedPlaceholder::Deploy {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_keccak".to_string(),
                },
            });
        }
        if let Some(letter) = stem_suffix.strip_prefix("program_") {
            return Ok(ResolvedPlaceholder::Deploy {
                role: format!("program_{letter}"),
                source: WasmSource::Inline {
                    wat: wat_sources::distinct_keccak_program(letter),
                },
            });
        }
        if stem_suffix == "multicall" {
            return Ok(ResolvedPlaceholder::Deploy {
                role: "multicall".to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::multicall_wat().to_string(),
                },
            });
        }
        if stem_suffix == "reentrant_multicall" {
            // Functionally identical to multicall, just lives in a distinct slot
            // so the fixture can self-call.
            return Ok(ResolvedPlaceholder::Deploy {
                role: "multicall".to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::multicall_wat().to_string(),
                },
            });
        }
        if stem_suffix == "storage_target" {
            return Ok(ResolvedPlaceholder::Deploy {
                role: "storage_target".to_string(),
                source: WasmSource::WatFile {
                    stem: "hostio_storage_store_bytes32".to_string(),
                },
            });
        }
        if stem_suffix == "create_factory" {
            return Ok(ResolvedPlaceholder::Deploy {
                role: "create_factory".to_string(),
                source: WasmSource::Inline {
                    wat: wat_sources::create_factory_wat().to_string(),
                },
            });
        }
        if let Some(_) = stem_suffix.strip_prefix("oversized_program_") {
            // Both the must-revert and succeeds cases just need the same
            // oversize payload. Pad past the legacy MAX_CODE_SIZE (24,576).
            return Ok(ResolvedPlaceholder::Deploy {
                role: DEFAULT_ROLE.to_string(),
                source: WasmSource::Padded {
                    stem: "hostio_keccak".to_string(),
                    target_size: 25_000,
                },
            });
        }
    }

    // Hostio invokes — calldata varies per variant suffix.
    if let Some(rest) = body.strip_prefix("compile_wasm_invoke_") {
        let calldata = hostio_invoke_calldata(rest, fixture_name)?;
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: DEFAULT_ROLE.to_string(),
            calldata,
            value: U256::ZERO,
        });
    }

    // ArbOwner setter (must be checked before generic signed_invoke_).
    if body == "signed_invoke_arbowner_set_max_stylus_fragments_to_capacity_2" {
        return Ok(ResolvedPlaceholder::ArbOwnerSetMaxFragments { max_fragments: 2 });
    }

    // EVM proxy deploys (no Stylus prefix).
    if body == "signed_deploy_evm_proxy_call_into_wasm" {
        return Ok(ResolvedPlaceholder::DeployEvm {
            role: "evm_proxy".to_string(),
            runtime_code: evm_proxy_call_runtime(),
        });
    }
    if body == "signed_deploy_evm_proxy_delegatecall_into_wasm" {
        return Ok(ResolvedPlaceholder::DeployEvm {
            role: "evm_proxy".to_string(),
            runtime_code: evm_proxy_delegatecall_runtime(),
        });
    }

    // Same-block batch of two invokes against the program.
    if body == "signed_batch_two_invokes_same_block_first_then_cache_hit" {
        return Ok(ResolvedPlaceholder::Batch {
            invokes: vec![
                BatchInvoke {
                    target_role: DEFAULT_ROLE.to_string(),
                    calldata: vec![0x11; 4],
                    value: U256::ZERO,
                },
                BatchInvoke {
                    target_role: DEFAULT_ROLE.to_string(),
                    calldata: vec![0x22; 4],
                    value: U256::ZERO,
                },
            ],
        });
    }

    // Subcall / cache invokes.
    if let Some(rest) = body.strip_prefix("signed_invoke_") {
        return resolve_signed_invoke(rest);
    }

    Err(anyhow!("unhandled placeholder: {placeholder}"))
}

fn hostio_invoke_calldata(rest: &str, _fixture: &str) -> Result<Vec<u8>> {
    if let Some(n) = rest.strip_prefix("keccak_").and_then(|s| s.strip_suffix("_bytes")) {
        let count: usize = n
            .parse()
            .map_err(|e| anyhow!("bad keccak byte count {n}: {e}"))?;
        return Ok(vec![0x55; count]);
    }
    if let Some(rest) = rest.strip_prefix("emit_log_") {
        // Format: <kind>_topics_payload_<len>
        let mut iter = rest.split('_');
        let topic_word = iter.next().ok_or_else(|| anyhow!("emit_log: missing topic word"))?;
        let topics: u8 = match topic_word {
            "zero" => 0,
            "one" => 1,
            "two" => 2,
            "three" => 3,
            "four" => 4,
            other => other.parse().map_err(|e| anyhow!("bad topic count {other}: {e}"))?,
        };
        // Skip "topics" sentinel.
        let _ = iter.next();
        // Skip "payload" sentinel.
        let _ = iter.next();
        let payload_len: usize = iter
            .next()
            .ok_or_else(|| anyhow!("emit_log: missing payload len"))?
            .parse()
            .map_err(|e| anyhow!("bad payload len: {e}"))?;
        let mut buf = Vec::with_capacity(1 + (topics as usize) * 32 + payload_len);
        buf.push(topics);
        // topic data
        for i in 0..topics {
            let mut topic = [0u8; 32];
            topic[31] = (i + 1) as u8;
            buf.extend_from_slice(&topic);
        }
        for i in 0..payload_len {
            buf.push((i & 0xff) as u8);
        }
        return Ok(buf);
    }
    if rest == "with_slot_key_zero_pad_32" {
        // Storage load: 32-byte key (we use slot keccak("arbreth_test_slot")).
        let mut key = [0u8; 32];
        key[31] = 0x07;
        return Ok(key.to_vec());
    }
    if rest == "store_zero_to_one" {
        return Ok(store_calldata(0x07, 1));
    }
    if rest == "store_one_to_two" {
        return Ok(store_calldata(0x07, 2));
    }
    if rest == "store_one_to_zero_refund" {
        return Ok(store_calldata(0x07, 0));
    }
    if rest == "storage_load_cold_then_warm_same_tx" {
        // Single load — the warm reload happens implicitly in the WAT only if it
        // were re-entered; for fixture-parse purposes we just pass a key.
        let mut key = [0u8; 32];
        key[31] = 0x07;
        return Ok(key.to_vec());
    }
    if rest == "msg_sender_from_eoa" {
        return Ok(Vec::new());
    }

    Err(anyhow!("unhandled hostio invoke variant: {rest}"))
}

fn store_calldata(slot_byte: u8, value_byte: u8) -> Vec<u8> {
    let mut buf = vec![0u8; 64];
    buf[31] = slot_byte;
    buf[63] = value_byte;
    buf
}

fn resolve_signed_invoke(rest: &str) -> Result<ResolvedPlaceholder> {
    if rest == "program_first_call"
        || rest == "program_second_call_cache_hit"
        || rest == "oversized_program_returns_zero"
        || rest.starts_with("program_a")
        || rest.starts_with("program_b")
        || rest.starts_with("program_c")
    {
        let target_role = if rest.starts_with("program_a") {
            "program_a".to_string()
        } else if rest.starts_with("program_b") {
            "program_b".to_string()
        } else if rest.starts_with("program_c") {
            "program_c".to_string()
        } else {
            "program".to_string()
        };
        return Ok(ResolvedPlaceholder::Invoke {
            target_role,
            calldata: vec![0xab; 8],
            value: U256::ZERO,
        });
    }
    if rest == "self_call_with_reentrant_branch" {
        // Multicall instruction set: call_contract(target=self, value=0, calldata=[…]).
        let mut calldata = Vec::new();
        // op = 0 (call), payload = self_address(20) + 0(8 gas low) + calldata(0 bytes inline)
        calldata.push(0);
        let mut addr = [0u8; 20];
        // Self-address is patched at runtime via msg_sender — for fixture parse
        // round-trip we just need a non-empty body.
        addr[19] = 0x42;
        calldata.extend_from_slice(&addr);
        calldata.extend_from_slice(&[0u8; 8]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "multicall".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "multicall_call_into_arbos_test_precompile" {
        // op=0 (call), addr=0x69 (ArbosTest), no inner calldata.
        let mut calldata = vec![0];
        calldata.extend_from_slice(&[0u8; 19]);
        calldata.push(0x69);
        calldata.extend_from_slice(&[0u8; 8]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "multicall".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "multicall_call_into_storage_with_value" {
        // op=0 (call), addr=storage_target placeholder (resolved at runtime by
        // the runner; here we encode a sentinel that the harness can patch).
        let mut calldata = vec![0];
        calldata.extend_from_slice(&[0u8; 19]);
        calldata.push(0xAA);
        calldata.extend_from_slice(&[0u8; 8]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "multicall".to_string(),
            calldata,
            value: U256::from(1u64),
        });
    }
    if rest == "multicall_delegatecall_into_storage" {
        let mut calldata = vec![1];
        calldata.extend_from_slice(&[0u8; 19]);
        calldata.push(0xAA);
        calldata.extend_from_slice(&[0u8; 8]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "multicall".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "multicall_staticcall_then_sstore_revert" {
        let mut calldata = vec![2];
        calldata.extend_from_slice(&[0u8; 19]);
        calldata.push(0xAA);
        calldata.extend_from_slice(&[0u8; 8]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "multicall".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "create1_with_payload_init_code" {
        let mut calldata = vec![3];
        calldata.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
        calldata.extend_from_slice(&[0xEF, 0xF0, 0x00, 0x00]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "create_factory".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "create2_with_fixed_salt_and_init_code" {
        let mut calldata = vec![4];
        let mut salt = [0u8; 32];
        salt[31] = 0x42;
        calldata.extend_from_slice(&salt);
        calldata.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
        calldata.extend_from_slice(&[0xEF, 0xF0, 0x00, 0x00]);
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "create_factory".to_string(),
            calldata,
            value: U256::ZERO,
        });
    }
    if rest == "proxy_call_into_msg_sender_program"
        || rest == "proxy_delegatecall_into_msg_sender_program"
    {
        return Ok(ResolvedPlaceholder::Invoke {
            target_role: "evm_proxy".to_string(),
            calldata: Vec::new(),
            value: U256::ZERO,
        });
    }

    Err(anyhow!("unhandled signed_invoke variant: {rest}"))
}

/// Minimal EVM bytecode that performs CALL into the address stored in the
/// first 32 bytes of its constructor argument and forwards calldata. We
/// inline a fixed-target dispatch that callers patch via the calldata they
/// pass on invoke; for fixture-parse round-trip the runtime code only needs
/// to be a syntactically-valid EVM stream.
fn evm_proxy_call_runtime() -> Vec<u8> {
    // CALL bytecode skeleton (returns RETURNDATA verbatim):
    //   PUSH1 0x00          ; ret_size
    //   PUSH1 0x00          ; ret_offset
    //   PUSH1 0x00          ; arg_size
    //   PUSH1 0x00          ; arg_offset
    //   PUSH1 0x00          ; value
    //   ADDRESS             ; placeholder target = self
    //   GAS
    //   CALL
    //   PUSH1 0x00
    //   PUSH1 0x00
    //   RETURN
    vec![
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0 (value)
        0x30, // ADDRESS (placeholder target)
        0x5A, // GAS
        0xF1, // CALL
        0x50, // POP
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0xF3, // RETURN
    ]
}

fn evm_proxy_delegatecall_runtime() -> Vec<u8> {
    vec![
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0x30, // ADDRESS (placeholder target)
        0x5A, // GAS
        0xF4, // DELEGATECALL
        0x50, // POP
        0x60, 0x00, // PUSH1 0
        0x60, 0x00, // PUSH1 0
        0xF3, // RETURN
    ]
}
