//! Cross-language interop scenarios: real Stylus-SDK contracts driven via
//! signed L2 EIP-1559 envelopes, with optional Solidity-companion calls
//! through `SolCaller.forward`.
//!
//! Pre-built Stylus initcode lives in `crates/arb-fuzz/prebuilt/*.hex`,
//! generated from `crates/arb-fuzz/stylus-programs/` via `cargo stylus
//! get-initcode`. The Solidity-companion runtimes are assembled at test time
//! to keep `solc` out of the build path (see `stylus_callback_runtime` etc).

use alloy_primitives::{b256, keccak256, Address, Bytes, B256, U256};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

use crate::{
    arbitrary_impls::{message_step, ArbosVersion, FUZZ_GAS_CAP, FUZZ_L1_BASE_FEE},
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};

const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);

/// Default signing key for the interop EOA. Hard-coded so every test sees
/// the same nonces / CREATE addresses.
pub fn interop_signing_key() -> B256 {
    b256!("d8f2c1b3a4e5f6970a1b2c3d4e5f60718293a4b5c6d7e8f900112233445566ff")
}

pub fn interop_eoa() -> Address {
    derive_address(interop_signing_key())
}

// -- prebuilt Stylus initcode ------------------------------------------------

const COUNTER_HEX: &str = include_str!("../../prebuilt/counter.hex");
const ERC20_MINI_HEX: &str = include_str!("../../prebuilt/erc20_mini.hex");
const SOL_CALLER_HEX: &str = include_str!("../../prebuilt/sol_caller.hex");
const STORAGE_STRESS_HEX: &str = include_str!("../../prebuilt/storage_stress.hex");

fn decode_hex(s: &str) -> Vec<u8> {
    let t = s.trim();
    (0..t.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&t[i..i + 2], 16).expect("valid hex initcode"))
        .collect()
}

/// Which Stylus program drives the scenario.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub enum WhichProgram {
    Counter,
    Erc20Mini,
    SolCaller,
    StorageStress,
}

impl<'a> Arbitrary<'a> for WhichProgram {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(match u.int_in_range::<u8>(0..=3)? {
            0 => Self::Counter,
            1 => Self::Erc20Mini,
            2 => Self::SolCaller,
            _ => Self::StorageStress,
        })
    }
}

impl WhichProgram {
    pub fn initcode(self) -> Vec<u8> {
        match self {
            Self::Counter => decode_hex(COUNTER_HEX),
            Self::Erc20Mini => decode_hex(ERC20_MINI_HEX),
            Self::SolCaller => decode_hex(SOL_CALLER_HEX),
            Self::StorageStress => decode_hex(STORAGE_STRESS_HEX),
        }
    }

    pub fn name(self) -> &'static str {
        match self {
            Self::Counter => "counter",
            Self::Erc20Mini => "erc20_mini",
            Self::SolCaller => "sol_caller",
            Self::StorageStress => "storage_stress",
        }
    }
}

/// Single interop scenario: deploy program, optionally deploy Solidity
/// companion, invoke with deterministic calldata derived from seeds.
#[derive(Debug, Clone, Serialize)]
pub struct DiffStylusInteropScenario {
    pub arbos_version: ArbosVersion,
    pub program: WhichProgram,
    pub action_seed: u64,
    pub interop_seed: u64,
}

impl<'a> Arbitrary<'a> for DiffStylusInteropScenario {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let arbos_version = ArbosVersion::arbitrary(u)?;
        let program = WhichProgram::arbitrary(u)?;
        let action_seed = u.arbitrary::<u64>()?;
        let interop_seed = u.arbitrary::<u64>()?;
        Ok(Self {
            arbos_version,
            program,
            action_seed,
            interop_seed,
        })
    }
}

impl DiffStylusInteropScenario {
    pub fn into_scenario(self) -> Option<Scenario> {
        let eoa = interop_eoa();
        let mut steps: Vec<ScenarioStep> = Vec::new();

        // 1. Fund the interop EOA.
        let fund_idx = next_msg_idx();
        let fund = DepositBuilder {
            from: eoa,
            to: eoa,
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: fund_idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Ok(msg) = fund.build() {
            steps.push(message_step(fund_idx, msg, fund_idx));
        } else {
            return None;
        }

        // 2. Deploy the Stylus program.
        let deploy_nonce = 0u64;
        let stylus_addr = create_address(eoa, deploy_nonce);
        let deploy_initcode = self.program.initcode();
        let deploy = signed_eip1559(
            deploy_nonce,
            None,
            Bytes::from(deploy_initcode),
            U256::ZERO,
            FUZZ_GAS_CAP,
        )
        .build()
        .ok()?;
        let deploy_idx = next_msg_idx();
        steps.push(message_step(deploy_idx, deploy, deploy_idx));

        // 3. activateProgram(stylus_addr).
        let mut activate_data = Vec::with_capacity(4 + 32);
        activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
        let mut padded = [0u8; 32];
        padded[12..].copy_from_slice(stylus_addr.as_slice());
        activate_data.extend_from_slice(&padded);
        let activate = signed_eip1559(
            1,
            Some(ARBWASM_ADDR),
            Bytes::from(activate_data),
            U256::from(10u128).pow(U256::from(15u64)),
            FUZZ_GAS_CAP,
        )
        .build()
        .ok()?;
        let activate_idx = next_msg_idx();
        steps.push(message_step(activate_idx, activate, activate_idx));

        // 4. For SolCaller, deploy the Solidity companion.
        let mut next_nonce = 2u64;
        let mut companion_addr: Option<Address> = None;
        if self.program == WhichProgram::SolCaller {
            let runtime = stylus_callback_runtime();
            let initcode = wrap_init_code(&runtime);
            let companion = create_address(eoa, next_nonce);
            companion_addr = Some(companion);
            let deploy_companion =
                signed_eip1559(next_nonce, None, Bytes::from(initcode), U256::ZERO, FUZZ_GAS_CAP)
                    .build()
                    .ok()?;
            let idx = next_msg_idx();
            steps.push(message_step(idx, deploy_companion, idx));
            next_nonce += 1;
        }

        // 5. Action-seeded invoke.
        let calldata = self.build_calldata(companion_addr);
        let invoke =
            signed_eip1559(next_nonce, Some(stylus_addr), Bytes::from(calldata), U256::ZERO, FUZZ_GAS_CAP)
                .build()
                .ok()?;
        let idx = next_msg_idx();
        steps.push(message_step(idx, invoke, idx));

        Some(Scenario {
            name: format!("fuzz_stylus_interop_{}", self.program.name()),
            description: format!(
                "fuzz-generated interop (program={}, action={}, interop={})",
                self.program.name(),
                self.action_seed,
                self.interop_seed
            ),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        })
    }

    fn build_calldata(&self, companion: Option<Address>) -> Vec<u8> {
        match self.program {
            WhichProgram::Counter => counter_calldata(self.action_seed),
            WhichProgram::Erc20Mini => erc20_calldata(self.action_seed),
            WhichProgram::SolCaller => {
                sol_caller_calldata(self.action_seed, self.interop_seed, companion)
            }
            WhichProgram::StorageStress => storage_stress_calldata(self.action_seed),
        }
    }
}

// -- calldata builders -------------------------------------------------------

pub fn counter_calldata(seed: u64) -> Vec<u8> {
    match seed % 4 {
        0 => selector_for("get()").to_vec(),
        1 => selector_for("increment()").to_vec(),
        2 => with_u256_arg("add(uint256)", U256::from((seed >> 2) & 0xff)),
        _ => with_u256_arg("set(uint256)", U256::from((seed >> 3) & 0xffff)),
    }
}

pub fn erc20_calldata(seed: u64) -> Vec<u8> {
    let actor = pseudo_addr(seed ^ 0xa5a5_a5a5_5a5a_5a5a);
    let amount = U256::from((seed >> 1) & 0xffff_ffff);
    match seed % 5 {
        0 => with_addr_uint("mint(address,uint256)", actor, amount),
        1 => with_addr_uint("transfer(address,uint256)", actor, amount),
        2 => with_addr_arg("balanceOf(address)", actor),
        3 => selector_for("totalSupply()").to_vec(),
        _ => with_addr_uint("mint(address,uint256)", interop_eoa(), amount),
    }
}

pub fn sol_caller_calldata(
    action_seed: u64,
    interop_seed: u64,
    companion: Option<Address>,
) -> Vec<u8> {
    let target = companion.unwrap_or_else(|| pseudo_addr(action_seed));
    match action_seed % 3 {
        0 => {
            // forward(target, ping(input))
            let inner = with_u256_arg("ping(uint256)", U256::from(interop_seed));
            with_addr_bytes("forward(address,bytes)", target, &inner)
        }
        1 => {
            // forwardStatic(target, pingCount())
            let inner = selector_for("pingCount()").to_vec();
            with_addr_bytes("forwardStatic(address,bytes)", target, &inner)
        }
        _ => selector_for("callCount()").to_vec(),
    }
}

pub fn storage_stress_calldata(seed: u64) -> Vec<u8> {
    let count = U256::from(((seed >> 4) & 0x1f) + 1);
    let start = U256::from((seed & 0xff) * 32);
    let base = U256::from(seed.wrapping_mul(0x9e37_79b9_7f4a_7c15));
    match seed % 3 {
        0 => with_three_u256("writeRange(uint256,uint256,uint256)", start, count, base),
        1 => with_two_u256("readRange(uint256,uint256)", start, count),
        _ => with_bool_arg("flush(bool)", (seed & 1) == 1),
    }
}

// -- low-level abi helpers ---------------------------------------------------

fn selector_for(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn with_u256_arg(sig: &str, v: U256) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    out.extend_from_slice(&v.to_be_bytes::<32>());
    out
}

fn with_addr_arg(sig: &str, a: Address) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(a.as_slice());
    out.extend_from_slice(&pad);
    out
}

fn with_bool_arg(sig: &str, b: bool) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    let mut pad = [0u8; 32];
    pad[31] = if b { 1 } else { 0 };
    out.extend_from_slice(&pad);
    out
}

fn with_two_u256(sig: &str, a: U256, b: U256) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    out.extend_from_slice(&a.to_be_bytes::<32>());
    out.extend_from_slice(&b.to_be_bytes::<32>());
    out
}

fn with_three_u256(sig: &str, a: U256, b: U256, c: U256) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    out.extend_from_slice(&a.to_be_bytes::<32>());
    out.extend_from_slice(&b.to_be_bytes::<32>());
    out.extend_from_slice(&c.to_be_bytes::<32>());
    out
}

fn with_addr_uint(sig: &str, a: Address, v: U256) -> Vec<u8> {
    let mut out = selector_for(sig).to_vec();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(a.as_slice());
    out.extend_from_slice(&pad);
    out.extend_from_slice(&v.to_be_bytes::<32>());
    out
}

fn with_addr_bytes(sig: &str, a: Address, data: &[u8]) -> Vec<u8> {
    // ABI: head = (address, offset); tail = (len, data)
    let mut out = selector_for(sig).to_vec();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(a.as_slice());
    out.extend_from_slice(&pad);
    let offset = U256::from(64u64);
    out.extend_from_slice(&offset.to_be_bytes::<32>());
    let len = U256::from(data.len() as u64);
    out.extend_from_slice(&len.to_be_bytes::<32>());
    out.extend_from_slice(data);
    let pad_len = (32 - (data.len() % 32)) % 32;
    out.extend(std::iter::repeat(0u8).take(pad_len));
    out
}

fn pseudo_addr(seed: u64) -> Address {
    let h = keccak256(seed.to_be_bytes());
    Address::from_slice(&h.as_slice()[12..])
}

// -- Solidity-companion bytecode --------------------------------------------

/// Hand-assembled runtime for `StylusCallback`:
///   bumps slot 0 each call, returns calldata[4..36] + 1.
/// Dispatch only matches `ping(uint256)` (selector 0x773acdef). Anything
/// else reverts with empty data. The `pingCount()` accessor (selector
/// 0x87704569) returns slot 0 raw — it's exercised by `forwardStatic`.
pub fn stylus_callback_runtime() -> Vec<u8> {
    let mut out = Vec::with_capacity(160);
    // Load 4-byte selector into stack as uint32 (high bytes).
    out.extend_from_slice(&[
        0x60, 0x00, // PUSH1 0
        0x35,       // CALLDATALOAD
        0x60, 0xe0, // PUSH1 0xe0
        0x1c,       // SHR  -> selector
    ]);
    // dup selector; compare against pingCount() selector
    out.extend_from_slice(&[0x80]); // DUP1
    out.extend_from_slice(&[0x63, 0x87, 0x70, 0x45, 0x69]); // PUSH4 pingCount()
    out.extend_from_slice(&[0x14]); // EQ
    // jumpi to 0x46 (ping_count_handler — patched below)
    let ping_count_jumpi_pos = out.len();
    out.extend_from_slice(&[0x60, 0x00, 0x57]); // PUSH1 dest PUSH1?  fix below

    // compare against ping(uint256) = 0x773acdef
    out.extend_from_slice(&[0x80]); // DUP1
    out.extend_from_slice(&[0x63, 0x77, 0x3a, 0xcd, 0xef]); // PUSH4
    out.extend_from_slice(&[0x14]); // EQ
    let ping_jumpi_pos = out.len();
    out.extend_from_slice(&[0x60, 0x00, 0x57]); // PUSH1 dest, JUMPI (patched)

    // fallback: revert
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xfd]); // PUSH1 0 PUSH1 0 REVERT

    // ping(uint256) handler
    let ping_dest = out.len();
    out.push(0x5b); // JUMPDEST
    // Load calldata[4..36] (the argument)
    out.extend_from_slice(&[0x60, 0x04]); // PUSH1 4
    out.push(0x35); // CALLDATALOAD -> stack: [selector, arg]
    // increment slot 0
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.push(0x54); // SLOAD -> [sel, arg, count]
    out.extend_from_slice(&[0x60, 0x01]); // PUSH1 1
    out.push(0x01); // ADD -> [sel, arg, count+1]
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.push(0x55); // SSTORE -> [sel, arg]
    // ret = arg + 1
    out.extend_from_slice(&[0x60, 0x01]); // PUSH1 1
    out.push(0x01); // ADD -> [sel, arg+1]
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.push(0x52); // MSTORE -> [sel]
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]); // PUSH1 32 PUSH1 0 RETURN

    // pingCount() handler
    let ping_count_dest = out.len();
    out.push(0x5b); // JUMPDEST
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.push(0x54); // SLOAD
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.push(0x52); // MSTORE
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]);

    // Patch jumpi destinations
    out[ping_count_jumpi_pos + 1] = ping_count_dest as u8;
    out[ping_jumpi_pos + 1] = ping_dest as u8;

    out
}

/// Hand-assembled runtime for `Reentrant.attack(address)`:
/// stylus.call(forward.selector || abi(this_addr, 0x))
///
/// This forces the SolCaller stylus contract to re-enter itself by calling
/// forward(this, "") -> Reentrant — which short-circuits since no other
/// fallback. The key path is the call/return-data plumbing across language
/// boundaries, not infinite recursion.
pub fn reentrant_runtime() -> Vec<u8> {
    // Layout: dispatch attack(address) by selector match; build the
    // forward(address,bytes) calldata at mem[0x40..], call stylus, return ok/0.
    let attack_sel = selector_for("attack(address)");
    let forward_sel = selector_for("forward(address,bytes)");
    let mut out = Vec::with_capacity(400);
    // Load selector
    out.extend_from_slice(&[0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c]);
    out.push(0x63);
    out.extend_from_slice(&attack_sel);
    out.push(0x14); // EQ
    let attack_jumpi_pos = out.len();
    out.extend_from_slice(&[0x60, 0x00, 0x57]); // JUMPI patched
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xfd]); // REVERT

    let attack_dest = out.len();
    out.push(0x5b); // JUMPDEST

    // Build memory layout for the outer call:
    // mem[0x00]: forward.selector (4 bytes, right-padded to 32)
    // mem[0x04..0x24]: address(this) padded
    // mem[0x24..0x44]: offset = 0x40 (64)
    // mem[0x44..0x64]: bytes_len = 0
    // total calldata bytes = 4 + 32 + 32 + 32 = 100

    // Write selector to mem[0]. PUSH32 (selector left-aligned). MSTORE shifts so we use
    // a 4-byte mstore via PUSH4 + shift.
    let mut sel_shifted = [0u8; 32];
    sel_shifted[..4].copy_from_slice(&forward_sel);
    out.push(0x7f); // PUSH32
    out.extend_from_slice(&sel_shifted);
    out.extend_from_slice(&[0x60, 0x00, 0x52]); // PUSH1 0 MSTORE

    // mem[0x04..0x24] = address(this)
    out.push(0x30); // ADDRESS
    out.extend_from_slice(&[0x60, 0x04, 0x52]); // PUSH1 4 MSTORE — overlapping with selector but stylus padding ok... Actually MSTORE writes 32 bytes starting at offset 4 so it overwrites bytes 4..36. The address is right-aligned (12 zero bytes + 20 addr bytes).

    // mem[0x24..0x44] = 0x40 (offset)
    out.extend_from_slice(&[0x60, 0x40]); // PUSH1 0x40
    out.extend_from_slice(&[0x60, 0x24, 0x52]); // PUSH1 0x24 MSTORE

    // mem[0x44..0x64] = 0 (bytes len)
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    out.extend_from_slice(&[0x60, 0x44, 0x52]); // PUSH1 0x44 MSTORE

    // CALL(gas=GAS, addr=calldata[4..36], value=0, in=mem[0x00], in_size=0x64, out=0, out_size=0)
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0 (out_size)
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0 (out_off)
    out.extend_from_slice(&[0x60, 0x64]); // PUSH1 0x64 (in_size)
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0 (in_off)
    out.extend_from_slice(&[0x60, 0x00]); // PUSH1 0 (value)
    out.extend_from_slice(&[0x60, 0x04, 0x35]); // PUSH1 4 CALLDATALOAD -> address (right-aligned in 32 bytes; CALL pops it as uint160)
    out.push(0x5a); // GAS
    out.push(0xf1); // CALL
    // ignore return; return uint256(1)
    out.push(0x50); // POP
    out.extend_from_slice(&[0x60, 0x01, 0x60, 0x00, 0x52]); // PUSH1 1 PUSH1 0 MSTORE
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]); // RETURN 32

    out[attack_jumpi_pos + 1] = attack_dest as u8;
    out
}

/// EVM deployer that returns `runtime` as the contract's bytecode.
/// 14-byte prologue + runtime, CODECOPY srcoff = 0x0e.
pub fn wrap_init_code(runtime: &[u8]) -> Vec<u8> {
    let size = runtime.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(14 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0e, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
        0xF3,
    ]);
    out.extend_from_slice(runtime);
    out
}

// -- tx helper ---------------------------------------------------------------

fn signed_eip1559(
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to,
        value,
        data,
        gas_limit: gas,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: interop_signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

/// CREATE address derivation. Public so reentrancy test can pre-compute the
/// stylus contract address before deploy.
pub fn create_address(sender: Address, nonce: u64) -> Address {
    let nonce_rlp = if nonce == 0 {
        vec![0x80u8]
    } else {
        let bytes = nonce.to_be_bytes();
        let trimmed: &[u8] = bytes
            .iter()
            .position(|b| *b != 0)
            .map(|i| &bytes[i..])
            .unwrap_or(&bytes[..0]);
        if trimmed.len() == 1 && trimmed[0] < 0x80 {
            vec![trimmed[0]]
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut payload = Vec::new();
    payload.push(0x80 + 20);
    payload.extend_from_slice(sender.as_slice());
    payload.extend_from_slice(&nonce_rlp);
    let mut rlp = vec![0xC0 + payload.len() as u8];
    rlp.extend_from_slice(&payload);
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.as_slice()[12..])
}
