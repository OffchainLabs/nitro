#![no_main]
use evm::{
    backend::MemoryAccount,
    executor::stack::{self as evm_stack, StackSubstateMetadata},
};
use eyre::{bail, Result};
use libfuzzer_sys::fuzz_target;
use primitive_types::{H160, U256};
use prover::{
    binary,
    machine::{GlobalState, Machine},
    utils::Bytes32,
    wavm::Opcode,
};
use serde::Deserialize;
use std::{collections::BTreeMap, fs::File, rc::Rc};

const MAX_STEPS: u64 = 200;
const DEBUG: bool = false;
const EVM_CONFIG: evm::Config = evm::Config::london();
const MAX_OSP_GAS: u64 = 15_000_000;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ContractInfo {
    deployed_bytecode: String,
}

fn get_contract_deployed_bytecode(contract: &str) -> Vec<u8> {
    let f = File::open(format!(
        "../../../contracts/build/contracts/src/osp/{0}.sol/{0}.json",
        contract,
    ))
    .expect("Failed to read contract JSON");
    let info: ContractInfo = serde_json::from_reader(f).expect("Failed to parse contract JSON");
    hex::decode(&info.deployed_bytecode[2..]).unwrap()
}

lazy_static::lazy_static! {
    static ref OSP_PREFIX: Vec<u8> = {
        let mut data = Vec::new();
        data.extend(hex::decode("5d3adcfb").unwrap()); // function selector
        data.extend([0; 32]); // maxInboxMessagesRead
        data.extend([0; 32]); // bridge
        data
    };
    static ref EVM_VICINITY: evm::backend::MemoryVicinity = {
        evm::backend::MemoryVicinity {
            gas_price: Default::default(),
            origin: Default::default(),
            chain_id: Default::default(),
            block_hashes: Default::default(),
            block_number: Default::default(),
            block_coinbase: Default::default(),
            block_timestamp: Default::default(),
            block_difficulty: Default::default(),
            block_gas_limit: Default::default(),
            block_base_fee_per_gas: Default::default(),
        }
    };
    static ref OSP_ENTRY_ADDRESS: H160 = H160::repeat_byte(1);
    static ref OSP_ENTRY_CODE: Vec<u8> = get_contract_deployed_bytecode("OneStepProofEntry");
    static ref EVM_BACKEND: evm::backend::MemoryBackend<'static> = {
        const CONTRACTS: &[&str] = &[
            "OneStepProofEntry",
            "OneStepProver0",
            "OneStepProverMemory",
            "OneStepProverMath",
            "OneStepProverHostIo",
        ];
        let mut state = BTreeMap::new();
        for (i, contract) in CONTRACTS.iter().enumerate() {
            let mut account = MemoryAccount::default();
            if i == 0 {
                account.code = OSP_ENTRY_CODE.clone();
                // Put the other provers' addresses in the entry contract's storage
                for i in 1..CONTRACTS.len() {
                    let mut key = [0u8; 32];
                    key[31] = i as u8 - 1;
                    account.storage.insert(key.into(), H160::repeat_byte(i as u8 + 1).into());
                }
            } else {
                account.code = get_contract_deployed_bytecode(contract);
            }
            state.insert(H160::repeat_byte(i as u8 + 1), account);
        }
        evm::backend::MemoryBackend::new(&*EVM_VICINITY, state)
    };
}

thread_local! {
    static OSP_ENTRY_CODE_RC: Rc<Vec<u8>> = Rc::new(OSP_ENTRY_CODE.clone());
}

fn make_evm_executor() -> evm_stack::StackExecutor<
    'static,
    'static,
    evm_stack::MemoryStackState<'static, 'static, evm::backend::MemoryBackend<'static>>,
    (),
> {
    let stack = evm_stack::MemoryStackState::new(
        StackSubstateMetadata::new(MAX_OSP_GAS, &EVM_CONFIG),
        &*EVM_BACKEND,
    );
    evm_stack::StackExecutor::new_with_precompiles(stack, &EVM_CONFIG, &())
}

fn test_proof(
    before_hash: Bytes32,
    steps: u64,
    opcode: Option<Opcode>,
    proof: Vec<u8>,
    after_hash: Bytes32,
) -> Result<()> {
    let mut data = OSP_PREFIX.clone();
    data.extend([0u8; (32 - 8)]);
    data.extend(steps.to_be_bytes());
    data.extend(before_hash);
    let proof_offset = data.len() + 32 - 4;
    data.extend([0u8; (32 - 8)]);
    data.extend(proof_offset.to_be_bytes());
    data.extend([0u8; (32 - 8)]);
    data.extend(proof.len().to_be_bytes());
    data.extend(&proof);
    if proof.len() % 32 != 0 {
        data.extend(std::iter::repeat(0).take(32 - (proof.len() % 32)));
    }
    if DEBUG {
        println!("Proving {:?} with {}", opcode, hex::encode(&data));
    }
    let code = OSP_ENTRY_CODE_RC.with(|code| code.clone());
    let context = evm::Context {
        address: *OSP_ENTRY_ADDRESS,
        caller: H160::default(),
        apparent_value: U256::default(),
    };
    let mut runtime = evm::Runtime::new(code, Rc::new(data), context, &EVM_CONFIG);
    let mut handler = make_evm_executor();
    let res = match runtime.run(&mut handler) {
        evm::Capture::Exit(res) => res,
        evm::Capture::Trap(_) => bail!("hit trap executing EVM"),
    };
    match res {
        evm::ExitReason::Succeed(_) => {
            let result = runtime.machine().return_value();
            if result.as_slice() != *after_hash {
                bail!(
                    "executing {:?} expecting after hash {} but got {}",
                    opcode,
                    after_hash,
                    hex::encode(result),
                );
            }
        }
        evm::ExitReason::Revert(_) => {
            let result = runtime.machine().return_value();
            if result.is_empty() {
                bail!("execution reverted");
            } else if result.len() >= 64
                && result[..31].iter().all(|b| *b == 0)
                && result[31] == 32
                && result[32..48].iter().all(|b| *b == 0)
            {
                bail!(
                    "execution reverted: {} ({})",
                    String::from_utf8_lossy(&result[64..]),
                    hex::encode(&result),
                );
            } else {
                bail!("execution reverted: {}", hex::encode(&result));
            }
        }
        evm::ExitReason::Error(err) => {
            bail!("EVM hit error: {:?}", err);
        }
        evm::ExitReason::Fatal(err) => {
            bail!("EVM hit fatal error: {:?}", err);
        }
    }
    Ok(())
}

fn fuzz_impl(data: &[u8]) -> Result<()> {
    let wavm_binary = binary::parse(data)?;
    let mut mach = Machine::from_binaries(
        &[],
        wavm_binary,
        true,
        true,
        false,
        GlobalState::default(),
        Default::default(),
        prover::machine::get_empty_preimage_resolver(),
    )?;
    let mut last_hash = mach.hash();
    while mach.get_steps() <= MAX_STEPS {
        let proof = mach.serialize_proof();
        let op = mach.get_next_instruction().map(|i| i.opcode);
        if DEBUG {
            println!("Executing {:?} with stack {:?}", op, mach.get_data_stack());
        }
        mach.step_n(1).expect("Failed to execute machine step");
        let new_hash = mach.hash();
        test_proof(last_hash, mach.get_steps(), op, proof, new_hash)
            .expect("Failed to validate proof");
        if new_hash == last_hash {
            break;
        }
        last_hash = new_hash;
    }
    Ok(())
}

static CONFIGURE_RAYON: std::sync::Once = std::sync::Once::new();

fuzz_target!(|data: &[u8]| {
    CONFIGURE_RAYON.call_once(|| {
        std::env::set_var("RAYON_NUM_THREADS", "1"); // in case a different version of rayon is loaded
        rayon::ThreadPoolBuilder::new()
            .num_threads(1)
            .build_global()
            .expect("Failed to configure global Rayon thread pool");
    });
    if let Err(err) = fuzz_impl(data) {
        if DEBUG {
            eprintln!("Non-critical error: {}", err);
        }
    }
});
