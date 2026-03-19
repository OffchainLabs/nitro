use std::collections::BTreeMap;

use alloy_primitives::{address, hex, Address, U256};
use revm::database::{EmptyDB, State, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use serde_json::{json, Map, Value};

use arb_node::genesis::{initialize_arbos_state, INITIAL_ARBOS_VERSION};
use arbos::arbos_types::ParsedInitMessage;

/// Arbitrum Sepolia chain owner.
const SEPOLIA_CHAIN_OWNER: Address = address!("71B61c2E250AFa05dFc36304D6c91501bE0965D8");

/// Arbitrum Sepolia chain ID.
const CHAIN_ID: u64 = 421614;

fn main() {
    // Create in-memory state backed by EmptyDB.
    let mut state: State<EmptyDB> = StateBuilder::new()
        .with_database(EmptyDB::default())
        .with_bundle_update()
        .build();

    // Serialized chain config JSON (exact bytes from canonical genesis).
    let serialized_chain_config = br#"{"chainId":421614,"homesteadBlock":0,"daoForkBlock":null,"daoForkSupport":true,"eip150Block":0,"eip150Hash":"0x0000000000000000000000000000000000000000000000000000000000000000","eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"clique":{"period":0,"epoch":0},"arbitrum":{"EnableArbOS":true,"AllowDebugPrecompiles":false,"DataAvailabilityCommittee":false,"InitialArbOSVersion":10,"InitialChainOwner":"0x71B61c2E250AFa05dFc36304D6c91501bE0965D8","GenesisBlockNum":0}}"#;

    // Initialize ArbOS state with Arbitrum Sepolia parameters.
    let init_msg = ParsedInitMessage {
        chain_id: U256::from(CHAIN_ID),
        initial_l1_base_fee: U256::from(1_517_780_963u64), /* L1 base fee at rollup creation
                                                            * (0x5a777fe3) */
        serialized_chain_config: serialized_chain_config.to_vec(),
    };

    initialize_arbos_state(
        &mut state,
        &init_msg,
        CHAIN_ID,
        INITIAL_ARBOS_VERSION,
        SEPOLIA_CHAIN_OWNER,
    )
    .expect("failed to initialize ArbOS state");

    // Merge all transitions into the bundle state.
    state.merge_transitions(BundleRetention::Reverts);

    // Extract all accounts and storage from the bundle state.
    let mut alloc: BTreeMap<String, Value> = BTreeMap::new();

    for (addr, account) in &state.bundle_state.state {
        let mut entry = Map::new();

        if let Some(ref info) = account.info {
            // Always include balance (reth requires it).
            entry.insert(
                "balance".into(),
                Value::String(format!("{:#x}", info.balance)),
            );

            // Nonce
            if info.nonce > 0 {
                entry.insert("nonce".into(), Value::String(format!("{:#x}", info.nonce)));
            }

            // Code
            if let Some(ref code) = info.code {
                let bytecode = code.original_bytes();
                if !bytecode.is_empty() {
                    entry.insert(
                        "code".into(),
                        Value::String(format!("0x{}", hex::encode(&bytecode))),
                    );
                }
            }
        } else {
            // Accounts without info still need balance for reth.
            entry.insert("balance".into(), Value::String("0x0".into()));
        }

        // Storage
        if !account.storage.is_empty() {
            let mut storage_map = Map::new();
            for (slot, slot_val) in &account.storage {
                if !slot_val.present_value.is_zero() {
                    let slot_key = format!("{:#066x}", slot);
                    let slot_value = format!("{:#066x}", slot_val.present_value);
                    storage_map.insert(slot_key, Value::String(slot_value));
                }
            }
            if !storage_map.is_empty() {
                entry.insert("storage".into(), Value::Object(storage_map));
            }
        }

        if !entry.is_empty() {
            let addr_str = format!("{:x}", addr);
            alloc.insert(addr_str, Value::Object(entry));
        }
    }

    // Build the complete genesis JSON.
    let genesis = json!({
        "config": {
            "chainId": CHAIN_ID,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "terminalTotalDifficulty": 0,
            "terminalTotalDifficultyPassed": true
        },
        "nonce": "0x0000000000000001",
        "timestamp": "0x0",
        "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "gasLimit": "0x4000000000000",
        "difficulty": "0x1",
        "mixHash": "0x00000000000000000000000000000000000000000000000a0000000000000000",
        "coinbase": "0x0000000000000000000000000000000000000000",
        "baseFeePerGas": "0x5f5e100",
        "alloc": alloc
    });

    let output = serde_json::to_string_pretty(&genesis).expect("failed to serialize genesis");
    println!("{}", output);
}
