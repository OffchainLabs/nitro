#![no_main]
use eyre::{bail, Result};
use libfuzzer_sys::fuzz_target;
use prover::{
    binary,
    machine::{GlobalState, Machine},
    utils::Bytes32,
    wavm::Opcode,
};
use serde::{Deserialize, Serialize};
use std::{
    collections::VecDeque,
    time::{Duration, Instant},
};

const MAX_RUNTIME: Duration = Duration::from_millis(500);
const DEBUG: bool = false;
const PARALLEL: usize = if DEBUG { 1 } else { 8 };

lazy_static::lazy_static! {
    static ref RPC_URL: String =
        std::env::var("RPC_URL").expect("RPC_URL environment variable must be present");
    static ref REQWEST_CLIENT: reqwest::Client = reqwest::Client::new();
    static ref OSP_ENTRY_ADDRESS: String = std::env::var("OSP_ENTRY_ADDRESS")
        .expect("OSP_ENTRY_ADDRESS environment variable must be present");
    static ref OSP_PREFIX: Vec<u8> = {
        let mut data = Vec::new();
        data.extend(hex::decode("2fae8811").unwrap()); // function selector
        data.extend([0; 32]); // maxInboxMessagesRead
        data.extend([0; 32]); // sequencerInbox
        data.extend([0; 32]); // delayedInbox
        data
    };
}

#[derive(Serialize)]
struct EthCallParams<'a> {
    to: &'a str,
    data: String,
}

#[derive(Serialize)]
struct EthCallRequest<'a> {
    jsonrpc: &'static str,
    id: usize,
    method: &'static str,
    params: (EthCallParams<'a>, &'static str),
}

#[derive(Deserialize)]
struct EthCallError {
    message: String,
}

#[derive(Deserialize)]
#[serde(untagged)]
enum EthCallResponse {
    Success { result: String },
    Failure { error: EthCallError },
}

async fn test_proof(
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
    let params = EthCallParams {
        to: &*OSP_ENTRY_ADDRESS,
        data: format!("0x{}", hex::encode(data)),
    };
    let request = EthCallRequest {
        jsonrpc: "2.0",
        id: 0,
        method: "eth_call",
        params: (params, "latest"),
    };
    let res: EthCallResponse = REQWEST_CLIENT
        .post(&*RPC_URL)
        .json(&request)
        .send()
        .await?
        .json()
        .await?;
    match res {
        EthCallResponse::Success { result } => {
            let mut got_hash = Bytes32::default();
            hex::decode_to_slice(&result[2..], &mut *got_hash)?;
            if got_hash != after_hash {
                bail!(
                    "executing {:?} expecting after hash {} but got {}",
                    opcode,
                    after_hash,
                    got_hash,
                );
            }
        }
        EthCallResponse::Failure { error } => bail!("{}", error.message),
    }
    Ok(())
}

async fn fuzz_impl(data: &[u8]) -> Result<()> {
    let wavm_binary = binary::parse(data)?;
    let mut mach = Machine::from_binaries(
        &[],
        wavm_binary,
        true,
        false,
        GlobalState::default(),
        Default::default(),
        Default::default(),
    )?;
    let start = Instant::now();
    let mut handles = VecDeque::new();
    let mut last_hash = mach.hash();
    while start.elapsed() < MAX_RUNTIME {
        let proof = mach.serialize_proof();
        let op = mach.get_next_instruction().map(|i| i.opcode);
        if DEBUG {
            println!("Executing {:?} with stack {:?}", op, mach.get_data_stack());
        }
        mach.step_n(1);
        let new_hash = mach.hash();
        handles.push_back(tokio::spawn(test_proof(
            last_hash,
            mach.get_steps(),
            op,
            proof,
            new_hash,
        )));
        if PARALLEL >= handles.len() {
            handles.pop_front().unwrap().await.unwrap().unwrap();
        }
        if new_hash == last_hash {
            break;
        }
        last_hash = new_hash;
    }
    for handle in handles {
        handle.await.unwrap().expect("Failed to test proof");
    }
    Ok(())
}

fuzz_target!(|data: &[u8]| {
    let runtime = tokio::runtime::Runtime::new().unwrap();
    let _guard = runtime.enter();
    if let Err(err) = runtime.block_on(fuzz_impl(data)) {
        if DEBUG {
            eprintln!("Non-critical error: {}", err);
        }
    }
});
