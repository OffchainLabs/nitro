//! Deterministic synthetic workload generators.

use std::collections::BTreeMap;

use alloy_primitives::{Address, Bytes, TxKind, U256};
use arb_executor_tests::helpers::{derive_address, sign_1559, sign_legacy, ONE_ETH, ONE_GWEI};
use rand::{rngs::StdRng, RngCore, SeedableRng};
use rand_chacha::ChaCha20Rng;
use serde::Deserialize;

use crate::runner::{BlockInput, DeployedContract, Workload};

const RECIPIENT: Address = alloy_primitives::address!("11111111111111111111111111111111111111ff");

/// Dispatch a generator by name.
pub fn generate(
    manifest_name: &str,
    chain_id: u64,
    arbos_version: u64,
    generator: &str,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let workload = match generator {
        "transfer_train" => transfer_train(manifest_name, chain_id, arbos_version, params)?,
        "thousand_tx_block" => thousand_tx_block(manifest_name, chain_id, arbos_version, params)?,
        "max_calldata" => max_calldata(manifest_name, chain_id, arbos_version, params)?,
        "precompile_fanout" => precompile_fanout(manifest_name, chain_id, arbos_version, params)?,
        "contract_deploy_swarm" => {
            contract_deploy_swarm(manifest_name, chain_id, arbos_version, params)?
        }
        "revert_storm" => revert_storm(manifest_name, chain_id, arbos_version, params)?,
        "stylus_deep_call_stack" => {
            stylus_deep_call_stack(manifest_name, chain_id, arbos_version, params)?
        }
        "stylus_cold_cache" => stylus_cold_cache(manifest_name, chain_id, arbos_version, params)?,
        "retryable_timeout_sweep" => {
            retryable_timeout_sweep(manifest_name, chain_id, arbos_version, params)?
        }
        "deposit_burst" => deposit_burst(manifest_name, chain_id, arbos_version, params)?,
        "fee_escalation" => fee_escalation(manifest_name, chain_id, arbos_version, params)?,
        "mixed_realistic" => mixed_realistic(manifest_name, chain_id, arbos_version, params)?,
        "stylus_storage_churn" => stylus_module_workload(
            manifest_name,
            chain_id,
            arbos_version,
            params,
            super::stylus_modules::StylusModule::StorageChurn,
        )?,
        "stylus_memory_grow" => stylus_module_workload(
            manifest_name,
            chain_id,
            arbos_version,
            params,
            super::stylus_modules::StylusModule::MemoryGrow,
        )?,
        "stylus_compute_loop" => stylus_module_workload(
            manifest_name,
            chain_id,
            arbos_version,
            params,
            super::stylus_modules::StylusModule::ComputeLoop,
        )?,
        "stylus_log_emit" => stylus_module_workload(
            manifest_name,
            chain_id,
            arbos_version,
            params,
            super::stylus_modules::StylusModule::LogEmit,
        )?,
        "stylus_host_fanout" => stylus_module_workload(
            manifest_name,
            chain_id,
            arbos_version,
            params,
            super::stylus_modules::StylusModule::HostFanout,
        )?,
        "mega_block" => mega_block(manifest_name, chain_id, arbos_version, params)?,
        "storage_churn_block" => {
            storage_churn_block(manifest_name, chain_id, arbos_version, params)?
        }
        "state_growth" => state_growth(manifest_name, chain_id, arbos_version, params)?,
        other => eyre::bail!("unknown synthetic generator: {other}"),
    };
    Ok(workload)
}

#[derive(Debug, Deserialize)]
struct CommonShape {
    #[serde(default = "default_block_count")]
    block_count: usize,
    #[serde(default = "default_txs_per_block")]
    txs_per_block: usize,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_block_count() -> usize {
    50
}
fn default_txs_per_block() -> usize {
    10
}
fn default_seed() -> u64 {
    0xA1B2_C3D4_E5F6_0789
}

/// Derive `n` deterministic test accounts.
fn derive_keys(seed: u64, n: usize) -> Vec<([u8; 32], Address)> {
    let mut rng = ChaCha20Rng::seed_from_u64(seed);
    (0..n)
        .map(|_| {
            let mut k = [0u8; 32];
            rng.fill_bytes(&mut k);
            k[0] |= 1;
            let addr = derive_address(k);
            (k, addr)
        })
        .collect()
}

/// Steady-state ETH transfers.
fn transfer_train(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: default_block_count(),
        txs_per_block: default_txs_per_block(),
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 16);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(1_000_000u128 * ONE_ETH)))
        .collect();

    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0u64)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    let mut rng = StdRng::seed_from_u64(p.seed ^ 0xDEAD_BEEF);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for _ in 0..p.txs_per_block {
            let idx = (rng.next_u32() as usize) % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            let tx = sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                U256::from(1_000u64),
                Bytes::new(),
                sk,
            );
            txs.push(tx);
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 30_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Dense ~1000-tx blocks.
fn thousand_tx_block(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 5,
        txs_per_block: 1000,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 64);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(10_000_000u128 * ONE_ETH)))
        .collect();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                U256::from(1u64),
                Bytes::new(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 200_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

#[derive(Debug, Deserialize)]
struct MaxCalldataParams {
    #[serde(default = "default_block_count")]
    block_count: usize,
    #[serde(default = "default_calldata_size")]
    calldata_bytes: usize,
    #[serde(default = "default_txs_calldata")]
    txs_per_block: usize,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_calldata_size() -> usize {
    32 * 1024
}
fn default_txs_calldata() -> usize {
    8
}

/// Near-max calldata per tx.
fn max_calldata(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: MaxCalldataParams =
        serde_json::from_value(params.clone()).unwrap_or(MaxCalldataParams {
            block_count: 20,
            calldata_bytes: default_calldata_size(),
            txs_per_block: default_txs_calldata(),
            seed: default_seed(),
        });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(100_000_000u128 * ONE_ETH)))
        .collect();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let payload: Bytes = vec![0xABu8; p.calldata_bytes].into();
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            // Generous gas limit covers L2 execution + ArbOS v60+ poster-gas
            // deduction, which scales with calldata size and L1 base fee.
            let intrinsic = 21_000 + 16 * p.calldata_bytes as u64;
            let gas = intrinsic.saturating_mul(8).max(5_000_000);
            txs.push(sign_1559(
                chain_id,
                nonce,
                ONE_GWEI * 10,
                ONE_GWEI,
                gas,
                TxKind::Call(RECIPIENT),
                U256::ZERO,
                payload.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 2_000_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Each tx hits a different ArbOS precompile.
fn precompile_fanout(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 20,
        txs_per_block: 64,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(10_000_000u128 * ONE_ETH)))
        .collect();
    // Pre-known Arbitrum precompile addresses (low-byte coded).
    let precompiles: [Address; 6] = [
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x64;
            b
        }), // ArbSys
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x65;
            b
        }), // ArbInfo
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x6c;
            b
        }), // ArbGasInfo
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x6e;
            b
        }), // ArbAddressTable
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x6f;
            b
        }), // ArbBLS / ArbStatistics
        Address::from_slice(&{
            let mut b = [0u8; 20];
            b[19] = 0x71;
            b
        }), // ArbAggregator
    ];
    // 4-byte selectors with no args; each precompile's first method is a no-arg
    // getter. (`getStorageGasAvailable()`, `getNetworkFeeAccount()`, etc.)
    // We just call with empty calldata; the precompile dispatch path is the hot
    // code; revert-on-bad-selector is acceptable since gas is still spent.
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            let target = precompiles[i % precompiles.len()];
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                100_000,
                TxKind::Call(target),
                U256::ZERO,
                Bytes::new(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 50_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

#[derive(Debug, Deserialize)]
struct DeployParams {
    #[serde(default = "default_block_count")]
    block_count: usize,
    #[serde(default = "default_deploys_per_block")]
    deploys_per_block: usize,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_deploys_per_block() -> usize {
    8
}

/// Trivial CREATE deploys per block.
fn contract_deploy_swarm(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: DeployParams = serde_json::from_value(params.clone()).unwrap_or(DeployParams {
        block_count: 20,
        deploys_per_block: 8,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(100_000_000u128 * ONE_ETH)))
        .collect();
    // Trivial creation bytecode that returns a 1-byte runtime ("STOP").
    // Init: PUSH1 0x01 PUSH1 0x00 MSTORE8  PUSH1 0x01 PUSH1 0x00 RETURN
    let init: Bytes = vec![0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xF3].into();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.deploys_per_block);
        for i in 0..p.deploys_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                250_000,
                TxKind::Create,
                U256::ZERO,
                init.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 50_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Every tx reverts on a known revert contract.
fn revert_storm(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 20,
        txs_per_block: 50,
        seed: default_seed(),
    });
    let revert_addr = Address::repeat_byte(0xCA);
    let revert_runtime: Vec<u8> = vec![0x60, 0x00, 0x60, 0x00, 0xFD];
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(10_000_000u128 * ONE_ETH)))
        .collect();
    let deployed = vec![DeployedContract {
        address: revert_addr,
        runtime_code: revert_runtime,
        balance: U256::ZERO,
    }];
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                100_000,
                TxKind::Call(revert_addr),
                U256::ZERO,
                Bytes::new(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 50_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: deployed,
        blocks,
        prewarm_alloc: None,
    })
}

/// Stylus workload: deploys via genesis, activates in block 1, calls per tx after.
fn stylus_call_workload(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
    _description: &str,
) -> eyre::Result<Workload> {
    use super::stylus_fixture::{
        activate_program_calldata, activate_program_value, stylus_call_selector, stylus_fixture,
        ARB_WASM_ADDRESS, STYLUS_FIXTURE_ADDRESS,
    };
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 20,
        txs_per_block: 16,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(10_000_000u128 * ONE_ETH)))
        .collect();
    let fixture = stylus_fixture()?;
    let selector = stylus_call_selector();
    let call_calldata = Bytes::from(selector.to_vec());
    let activate_calldata = activate_program_calldata(STYLUS_FIXTURE_ADDRESS);
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);

    let (activator_sk, activator_addr) = senders[0];

    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block + 1);

        // Block 1 starts with the activateProgram tx.
        if b == 0 {
            let nonce = *nonces.get(&activator_addr).unwrap();
            *nonces.get_mut(&activator_addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                5_000_000,
                TxKind::Call(ARB_WASM_ADDRESS),
                activate_program_value(),
                activate_calldata.clone(),
                activator_sk,
            ));
        }

        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                500_000,
                TxKind::Call(STYLUS_FIXTURE_ADDRESS),
                U256::ZERO,
                call_calldata.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 60_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: vec![fixture],
        blocks,
        prewarm_alloc: None,
    })
}

/// Stylus workload targeting a specific WASM module.
fn stylus_module_workload(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
    module: super::stylus_modules::StylusModule,
) -> eyre::Result<Workload> {
    use super::stylus_fixture::{
        activate_program_calldata, activate_program_value, stylus_call_selector, stylus_fixture_for,
    };
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 200,
        txs_per_block: 8,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(10_000_000u128 * ONE_ETH)))
        .collect();
    let fixture = stylus_fixture_for(module)?;
    let target_addr = module.deploy_address();
    let selector = stylus_call_selector();
    let call_calldata = Bytes::from(selector.to_vec());
    let activate_calldata = activate_program_calldata(target_addr);
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);

    let (activator_sk, activator_addr) = senders[0];

    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block + 1);
        if b == 0 {
            let nonce = *nonces.get(&activator_addr).unwrap();
            *nonces.get_mut(&activator_addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                10_000_000,
                TxKind::Call(super::stylus_fixture::ARB_WASM_ADDRESS),
                activate_program_value(),
                activate_calldata.clone(),
                activator_sk,
            ));
        }
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            // Per-call gas budget needs to cover the WASM's actual work.
            // 5M is comfortable for storage_churn + memory_grow; cheaper
            // modules use less.
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                5_000_000,
                TxKind::Call(target_addr),
                U256::ZERO,
                call_calldata.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 200_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: vec![fixture],
        blocks,
        prewarm_alloc: None,
    })
}

#[derive(Debug, Deserialize)]
struct StorageChurnParams {
    #[serde(default = "default_block_count")]
    block_count: usize,
    #[serde(default = "default_storage_writes_per_tx")]
    sstores_per_tx: usize,
    #[serde(default = "default_txs_per_storage_block")]
    txs_per_block: usize,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_storage_writes_per_tx() -> usize {
    16
}
fn default_txs_per_storage_block() -> usize {
    32
}

/// Every tx is a CREATE writing N SSTOREs.
fn storage_churn_block(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: StorageChurnParams =
        serde_json::from_value(params.clone()).unwrap_or(StorageChurnParams {
            block_count: 200,
            sstores_per_tx: 16,
            txs_per_block: 32,
            seed: default_seed(),
        });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(1_000_000_000u128 * ONE_ETH)))
        .collect();

    // CREATE init code that performs `sstores_per_tx` SSTOREs then returns 0-length runtime.
    // Per slot: PUSH1 i, PUSH1 i, SSTORE  (5 bytes per cycle). Then PUSH1 0 PUSH1 0 RETURN (5).
    let mut init: Vec<u8> = Vec::with_capacity(5 * p.sstores_per_tx + 5);
    for i in 0..p.sstores_per_tx {
        init.push(0x60); // PUSH1
        init.push(((i + 1) & 0xff) as u8); // value
        init.push(0x60); // PUSH1
        init.push((i & 0xff) as u8); // key
        init.push(0x55); // SSTORE
    }
    init.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xF3]); // PUSH1 0 PUSH1 0 RETURN
    let init_bytes: Bytes = init.into();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            // ~22k SSTORE-cold per slot; allocate generously.
            let gas = 100_000 + 25_000 * p.sstores_per_tx as u64;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                gas,
                TxKind::Create,
                U256::ZERO,
                init_bytes.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 250_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

#[derive(Debug, Deserialize)]
struct MegaBlockParams {
    #[serde(default = "default_mega_block_count")]
    block_count: usize,
    #[serde(default = "default_target_block_gas")]
    target_block_gas: u64,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_mega_block_count() -> usize {
    50
}
fn default_target_block_gas() -> u64 {
    200_000_000
}

/// Blocks packed near a 200M gas target via dense storage-write CREATEs.
fn mega_block(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: MegaBlockParams = serde_json::from_value(params.clone()).unwrap_or(MegaBlockParams {
        block_count: 50,
        target_block_gas: 200_000_000,
        seed: default_seed(),
    });
    // Each tx targets ~5M gas. Number of txs = target / 5M.
    let per_tx_gas: u64 = 5_000_000;
    let txs_per_block = (p.target_block_gas / per_tx_gas) as usize;
    // Each tx deploys a contract with ~150 SSTOREs (~3.3M gas in writes).
    let sstores_per_tx = 150usize;
    let inner = serde_json::json!({
        "block_count": p.block_count,
        "sstores_per_tx": sstores_per_tx,
        "txs_per_block": txs_per_block,
        "seed": p.seed,
    });
    let mut wl = storage_churn_block(name, chain_id, arbos_version, &inner)?;
    for b in &mut wl.blocks {
        b.gas_limit = (p.target_block_gas as u128 + 50_000_000) as u64;
    }
    Ok(wl)
}

#[derive(Debug, Deserialize)]
struct StateGrowthParams {
    #[serde(default = "default_state_growth_blocks")]
    block_count: usize,
    #[serde(default = "default_state_growth_per_block")]
    deploys_per_block: usize,
    #[serde(default = "default_seed")]
    seed: u64,
}

fn default_state_growth_blocks() -> usize {
    1000
}
fn default_state_growth_per_block() -> usize {
    32
}

/// Empty CREATEs to grow the global account trie organically.
fn state_growth(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: StateGrowthParams =
        serde_json::from_value(params.clone()).unwrap_or(StateGrowthParams {
            block_count: 1000,
            deploys_per_block: 32,
            seed: default_seed(),
        });
    let senders = derive_keys(p.seed, 16);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(1_000_000_000u128 * ONE_ETH)))
        .collect();
    // Init code: returns a 1-byte runtime.
    let init: Bytes = vec![0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xF3].into();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.deploys_per_block);
        for i in 0..p.deploys_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                250_000,
                TxKind::Create,
                U256::ZERO,
                init.clone(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 100_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Stylus deep call stack.
fn stylus_deep_call_stack(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    stylus_call_workload(
        name,
        chain_id,
        arbos_version,
        params,
        "stylus-deep-call-stack",
    )
}

/// Stylus calls with cold cache (same shape as deep_call_stack).
fn stylus_cold_cache(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    stylus_call_workload(name, chain_id, arbos_version, params, "stylus-cold-cache")
}

/// Transfers to fresh recipient addresses (new-account creation pressure).
fn deposit_burst(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 20,
        txs_per_block: 64,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(100_000_000u128 * ONE_ETH)))
        .collect();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut recipient_idx: u64 = 1;
    let mut blocks = Vec::with_capacity(p.block_count);
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            let mut bytes = [0u8; 20];
            bytes[12..].copy_from_slice(&recipient_idx.to_be_bytes());
            recipient_idx = recipient_idx.wrapping_add(1);
            let recipient = Address::from(bytes);
            txs.push(sign_legacy(
                chain_id,
                nonce,
                ONE_GWEI,
                25_000,
                TxKind::Call(recipient),
                U256::from(1_000u64),
                Bytes::new(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee: 100_000_000,
            gas_limit: 30_000_000,
            txs,
        });
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Base fee ramps 5% per block.
fn fee_escalation(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 50,
        txs_per_block: 16,
        seed: default_seed(),
    });
    let senders = derive_keys(p.seed, 8);
    let funded: Vec<(Address, U256)> = senders
        .iter()
        .map(|(_, a)| (*a, U256::from(1_000_000_000u128 * ONE_ETH)))
        .collect();
    let mut nonces: BTreeMap<Address, u64> = senders.iter().map(|(_, a)| (*a, 0)).collect();
    let mut blocks = Vec::with_capacity(p.block_count);
    let mut base_fee: u64 = 100_000_000;
    for b in 0..p.block_count {
        let mut txs = Vec::with_capacity(p.txs_per_block);
        for i in 0..p.txs_per_block {
            let idx = i % senders.len();
            let (sk, addr) = senders[idx];
            let nonce = *nonces.get(&addr).unwrap();
            *nonces.get_mut(&addr).unwrap() = nonce + 1;
            let max_fee = (base_fee as u128).saturating_mul(20);
            txs.push(sign_1559(
                chain_id,
                nonce,
                max_fee,
                ONE_GWEI,
                30_000,
                TxKind::Call(RECIPIENT),
                U256::from(1u64),
                Bytes::new(),
                sk,
            ));
        }
        blocks.push(BlockInput {
            block_number: (b + 1) as u64,
            timestamp: 1_700_000_000 + (b as u64) * 12,
            base_fee,
            gas_limit: 30_000_000,
            txs,
        });
        base_fee = (base_fee as u128 * 105 / 100) as u64;
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded,
        deployed_contracts: Vec::new(),
        blocks,
        prewarm_alloc: None,
    })
}

/// Weighted mix mirroring representative Arbitrum traffic.
fn mixed_realistic(
    name: &str,
    chain_id: u64,
    arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 100,
        txs_per_block: 32,
        seed: default_seed(),
    });
    let pieces = [
        ("transfer_train", 70u32),
        ("precompile_fanout", 12u32),
        ("revert_storm", 8u32),
        ("deposit_burst", 7u32),
        ("max_calldata", 3u32),
    ];
    let total_weight: u32 = pieces.iter().map(|(_, w)| *w).sum();
    let mut rng = ChaCha20Rng::seed_from_u64(p.seed);
    let mut blocks: Vec<BlockInput> = Vec::with_capacity(p.block_count);
    let mut funded: BTreeMap<Address, U256> = BTreeMap::new();
    let mut deployed: Vec<DeployedContract> = Vec::new();
    let mut block_no: u64 = 1;
    for _ in 0..p.block_count {
        let r = rng.next_u32() % total_weight;
        let mut acc = 0u32;
        let mut chosen = pieces[0].0;
        for (kind, w) in pieces.iter() {
            acc += *w;
            if r < acc {
                chosen = *kind;
                break;
            }
        }
        let sub_params = serde_json::json!({
            "block_count": 1,
            "txs_per_block": p.txs_per_block,
            "seed": rng.next_u64(),
        });
        let mut sub = generate(name, chain_id, arbos_version, chosen, &sub_params)?;
        for (a, b) in &sub.funded_accounts {
            funded.entry(*a).or_insert(*b);
        }
        for c in sub.deployed_contracts.drain(..) {
            if !deployed.iter().any(|d| d.address == c.address) {
                deployed.push(c);
            }
        }
        if let Some(mut b) = sub.blocks.into_iter().next() {
            b.block_number = block_no;
            b.timestamp = 1_700_000_000 + block_no * 12;
            blocks.push(b);
            block_no += 1;
        }
    }
    Ok(Workload {
        manifest_name: name.into(),
        chain_id,
        arbos_version,
        funded_accounts: funded.into_iter().collect(),
        deployed_contracts: deployed,
        blocks,
        prewarm_alloc: None,
    })
}

/// Advances block timestamps one day per block to drive retryable timeout sweeps.
fn retryable_timeout_sweep(
    name: &str,
    chain_id: u64,
    _arbos_version: u64,
    params: &serde_json::Value,
) -> eyre::Result<Workload> {
    let p: CommonShape = serde_json::from_value(params.clone()).unwrap_or(CommonShape {
        block_count: 30,
        txs_per_block: 16,
        seed: default_seed(),
    });
    // For now this is structurally identical to transfer_train but with
    // accelerated timestamps so the retryable sweep code observes timeouts.
    let mut wl = transfer_train(name, chain_id, _arbos_version, params)?;
    for (i, b) in wl.blocks.iter_mut().enumerate() {
        b.timestamp = 1_700_000_000 + (i as u64) * 86_400; // 1 day per block
    }
    let _ = p;
    Ok(wl)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn transfer_train_produces_blocks() {
        let g = generate(
            "test/transfer_train",
            421614,
            30,
            "transfer_train",
            &serde_json::json!({ "block_count": 3, "txs_per_block": 5 }),
        )
        .unwrap();
        assert_eq!(g.blocks.len(), 3);
        assert_eq!(g.blocks[0].txs.len(), 5);
        assert!(!g.funded_accounts.is_empty());
    }

    #[test]
    fn thousand_tx_block_packs() {
        let g = generate(
            "test/thousand",
            421614,
            30,
            "thousand_tx_block",
            &serde_json::json!({ "block_count": 1, "txs_per_block": 256 }),
        )
        .unwrap();
        assert_eq!(g.blocks[0].txs.len(), 256);
    }

    #[test]
    fn unknown_generator_errors() {
        let r = generate("x", 1, 30, "does-not-exist", &serde_json::json!({}));
        assert!(r.is_err());
    }

    #[test]
    fn mixed_realistic_produces_blocks() {
        let g = generate(
            "x",
            421614,
            30,
            "mixed_realistic",
            &serde_json::json!({ "block_count": 8, "txs_per_block": 4 }),
        )
        .unwrap();
        assert_eq!(g.blocks.len(), 8);
        assert!(!g.funded_accounts.is_empty());
    }

    #[test]
    fn deposit_burst_creates_unique_recipients() {
        let g = generate(
            "x",
            421614,
            30,
            "deposit_burst",
            &serde_json::json!({ "block_count": 1, "txs_per_block": 5 }),
        )
        .unwrap();
        assert_eq!(g.blocks[0].txs.len(), 5);
    }

    #[test]
    fn fee_escalation_ramps_base_fee() {
        let g = generate(
            "x",
            421614,
            30,
            "fee_escalation",
            &serde_json::json!({ "block_count": 5, "txs_per_block": 2 }),
        )
        .unwrap();
        let first = g.blocks.first().unwrap().base_fee;
        let last = g.blocks.last().unwrap().base_fee;
        assert!(last > first, "base fee should escalate ({first} -> {last})");
    }

    #[test]
    fn revert_storm_includes_revert_contract() {
        let g = generate(
            "x",
            421614,
            30,
            "revert_storm",
            &serde_json::json!({ "block_count": 1, "txs_per_block": 3 }),
        )
        .unwrap();
        assert_eq!(g.deployed_contracts.len(), 1);
    }
}
