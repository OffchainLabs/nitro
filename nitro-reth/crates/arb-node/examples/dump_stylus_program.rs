//! One-shot diagnostic for Stylus contract activation parity.
//!
//! Opens the existing reth chain DB read-only, fetches the contract bytecode
//! and the on-chain `Programs` storage word for a given (block, address),
//! decompresses the Stylus WASM, runs `arb_stylus::activate_program`, and
//! prints both the on-chain values and what we computed.
//!
//! Compare the printed `module_hash` lines: a divergence is the smoking gun
//! for the block-55755413-style Stylus bug.
//!
//! Usage:
//!     cargo run -p arb-node --example dump_stylus_program -- \
//!         --datadir /data/arbreth-data/db \
//!         --chain   /data/arbreth/genesis/arbitrum-sepolia.json \
//!         --block   55755413 \
//!         --addr    0x42108f617cc7a04b841db639431e1faa8b0cc3e9
//!
//! IMPORTANT: the running node opens the DB with `--db.exclusive=true`. Stop
//! the node before running this tool, or open will fail.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::{path::PathBuf, sync::Arc};

use alloy_primitives::{Address, B256, U256};
use clap::Parser;
use eyre::Context;
use reth_chainspec::ChainSpec;
use reth_provider::providers::{ProviderFactoryBuilder, ReadOnlyConfig};
use reth_storage_api::StateProvider;

use arb_precompiles::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, PROGRAMS_DATA_KEY,
    PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
};

#[derive(Parser, Debug)]
#[command(about = "Dump Stylus activation state at a given block")]
struct Args {
    /// Path to the reth datadir (the directory containing `db/`, `static_files/`, `rocksdb/`).
    #[arg(long, default_value = "/data/arbreth-data")]
    datadir: PathBuf,

    /// Path to the chainspec genesis JSON.
    #[arg(long, default_value = "/data/arbreth/genesis/arbitrum-sepolia.json")]
    chain: PathBuf,

    /// Block number at which to read state.
    #[arg(long)]
    block: u64,

    /// One or more contract addresses to inspect.
    #[arg(long = "addr", num_args = 1.., required = true)]
    addrs: Vec<String>,
}

fn parse_addr(s: &str) -> eyre::Result<Address> {
    let s = s.trim_start_matches("0x");
    let bytes = hex_decode(s).wrap_err("addr hex")?;
    if bytes.len() != 20 {
        eyre::bail!("addr must be 20 bytes, got {}", bytes.len());
    }
    let mut out = [0u8; 20];
    out.copy_from_slice(&bytes);
    Ok(Address::from(out))
}

fn hex_decode(s: &str) -> eyre::Result<Vec<u8>> {
    if !s.len().is_multiple_of(2) {
        eyre::bail!("odd hex length");
    }
    (0..s.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&s[i..i + 2], 16).wrap_err("hex byte"))
        .collect()
}

fn load_chainspec(path: &PathBuf) -> eyre::Result<Arc<ChainSpec>> {
    let raw = std::fs::read_to_string(path)
        .wrap_err_with(|| format!("read chainspec from {}", path.display()))?;
    let genesis: alloy_genesis::Genesis = serde_json::from_str(&raw).wrap_err("parse genesis")?;
    Ok(Arc::new(genesis.into()))
}

/// Slot of the `Programs[codehash]` data word at `ARBOS_STATE_ADDRESS`.
fn program_data_slot(codehash: B256) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    map_slot_b256(data_key.as_slice(), &codehash)
}

/// Slot of the per-program `module_hashes[codehash]` at `ARBOS_STATE_ADDRESS`.
fn module_hash_slot(codehash: B256) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let module_hashes_key = derive_subspace_key(programs_key.as_slice(), &[2]);
    map_slot_b256(module_hashes_key.as_slice(), &codehash)
}

/// Slot of the `Programs.params` 32-byte word.
fn programs_params_slot() -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    map_slot(params_key.as_slice(), 0)
}

#[derive(Debug)]
#[allow(dead_code)]
struct Program {
    version: u16,
    init_cost: u16,
    cached_cost: u16,
    footprint: u16,
    activated_at_hours: u32,
    asm_estimate_kb: u32,
    cached_flag: u8,
}

fn parse_program_word(w: &[u8; 32]) -> Program {
    Program {
        version: u16::from_be_bytes([w[0], w[1]]),
        init_cost: u16::from_be_bytes([w[2], w[3]]),
        cached_cost: u16::from_be_bytes([w[4], w[5]]),
        footprint: u16::from_be_bytes([w[6], w[7]]),
        activated_at_hours: ((w[8] as u32) << 16) | ((w[9] as u32) << 8) | w[10] as u32,
        asm_estimate_kb: ((w[11] as u32) << 16) | ((w[12] as u32) << 8) | w[13] as u32,
        cached_flag: w[14],
    }
}

fn dump_one(state: &dyn StateProvider, addr: Address) -> eyre::Result<()> {
    println!("\n══ {addr} ══");

    let acct = state
        .basic_account(&addr)
        .wrap_err("basic_account")?
        .ok_or_else(|| eyre::eyre!("account does not exist at this block"))?;

    let codehash = acct
        .bytecode_hash
        .ok_or_else(|| eyre::eyre!("account has no codehash"))?;
    println!("  codehash:      {codehash}");
    println!("  nonce:         {}", acct.nonce);
    println!("  balance:       {}", acct.balance);

    let bytecode = state
        .bytecode_by_hash(&codehash)
        .wrap_err("bytecode_by_hash")?
        .ok_or_else(|| eyre::eyre!("no bytecode for codehash"))?;
    let raw = bytecode.original_byte_slice();
    println!("  bytecode:      {} bytes", raw.len());

    if !arb_stylus::is_stylus_program(raw) {
        println!("  NOT a Stylus program — skipping activation");
        return Ok(());
    }

    let wasm = arb_stylus::decompress_wasm(raw).wrap_err("decompress wasm")?;
    println!("  decompressed:  {} bytes", wasm.len());
    println!("  wasm_hash:     {}", alloy_primitives::keccak256(&wasm));

    // Read on-chain Programs data + module hash + params at this block.
    let params = state
        .storage(ARBOS_STATE_ADDRESS, B256::from(programs_params_slot()))?
        .unwrap_or_default();
    let params_word = params.to_be_bytes::<32>();
    let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
    let page_limit = u16::from_be_bytes([params_word[13], params_word[14]]);
    println!("  params.version: {params_version}");
    println!("  params.page_limit: {page_limit}");

    let onchain_word = state
        .storage(ARBOS_STATE_ADDRESS, B256::from(program_data_slot(codehash)))?
        .unwrap_or_default();
    let onchain = parse_program_word(&onchain_word.to_be_bytes::<32>());
    println!("  on-chain Program: {onchain:?}");

    let onchain_module_hash = state
        .storage(ARBOS_STATE_ADDRESS, B256::from(module_hash_slot(codehash)))?
        .unwrap_or_default();
    let onchain_module_hash_b32 = B256::from(onchain_module_hash.to_be_bytes::<32>());
    println!("  on-chain module_hash: {onchain_module_hash_b32}");

    if onchain.version == 0 {
        println!("  Program word is empty — not activated at this block.");
        return Ok(());
    }

    // Re-activate with the SAME (params_version, arbos_version, page_limit) the
    // chain saw at this block. We don't know the exact arbos_version stored
    // for this block from CLI; pass 30 as a Stylus-era default. Try the most
    // common values if the first fails.
    let arbos_versions: &[u64] = &[30, 31, 32, 11];
    let mut last_err = None;
    let mut activation = None;
    for &av in arbos_versions {
        let mut gas = u64::MAX;
        match arb_stylus::activate_program(
            &wasm,
            codehash.as_ref(),
            params_version,
            av,
            page_limit,
            false,
            &mut gas,
        ) {
            Ok(info) => {
                println!("  activated at arbos_version={av}");
                activation = Some(info);
                break;
            }
            Err(e) => {
                last_err = Some(format!("arbos_version={av}: {e}"));
            }
        }
    }
    let info = match activation {
        Some(i) => i,
        None => {
            println!("  ACTIVATION FAILED across tried arbos versions:");
            if let Some(e) = last_err {
                println!("    {e}");
            }
            return Ok(());
        }
    };

    println!("  computed module_hash: {}", info.module_hash);
    println!("  computed init_gas:    {}", info.init_gas);
    println!("  computed cached_gas:  {}", info.cached_init_gas);
    println!("  computed footprint:   {}", info.footprint);
    println!("  computed asm_estimate:{}", info.asm_estimate);

    let match_hash = info.module_hash == onchain_module_hash_b32;
    let match_footprint = info.footprint == onchain.footprint;
    let match_init = info.init_gas == onchain.init_cost;
    let match_cached = info.cached_init_gas == onchain.cached_cost;
    let match_asm = info.asm_estimate / 1024 == onchain.asm_estimate_kb;

    println!(
        "  PARITY: hash={} footprint={} init={} cached={} asm_kb={}",
        ok(match_hash),
        ok(match_footprint),
        ok(match_init),
        ok(match_cached),
        ok(match_asm),
    );

    if !match_hash {
        println!("\n  >>> MODULE HASH MISMATCH — root cause confirmed for this contract.");
        println!("      computed:  {}", info.module_hash);
        println!("      on-chain:  {onchain_module_hash_b32}");
    }

    Ok(())
}

fn ok(b: bool) -> &'static str {
    if b {
        "MATCH"
    } else {
        "MISMATCH"
    }
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();
    println!("datadir: {}", args.datadir.display());
    println!("chain:   {}", args.chain.display());
    println!("block:   {}", args.block);

    let chainspec = load_chainspec(&args.chain)?;
    println!(
        "loaded chainspec: chain_id={} hash={:?}",
        chainspec.chain.id(),
        chainspec.genesis_hash()
    );

    // mdbx may need WAL recovery if the previous shutdown was unclean. Recovery
    // requires write access, so open RW once briefly to let mdbx run the
    // recovery, then drop and re-open read-only.
    {
        let db_dir = args.datadir.join("db");
        let _recovery =
            reth_db::open_db(&db_dir, Default::default()).wrap_err("open_db RW for recovery")?;
        println!("mdbx recovery pass complete");
    }

    let runtime = reth_tasks::Runtime::test();
    let config = ReadOnlyConfig::from_datadir(&args.datadir).no_watch();
    let pf = ProviderFactoryBuilder::<arb_node::ArbNode>::default()
        .open_read_only(chainspec, config, runtime)
        .wrap_err("open_read_only")?;

    let state = pf
        .history_by_block_number(args.block)
        .wrap_err("history_by_block_number")?;

    for s in &args.addrs {
        let addr = parse_addr(s)?;
        if let Err(e) = dump_one(state.as_ref(), addr) {
            println!("  ERROR: {e:?}");
        }
    }

    Ok(())
}
