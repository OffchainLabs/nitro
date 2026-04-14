use alloy_evm::{
    eth::EthEvmContext, precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory,
};
use alloy_primitives::{Address, Bytes, B256, U256};
use arb_precompiles::register_arb_precompiles;
use arb_stylus::{
    config::StylusConfig, ink::Gas as StylusGas, meter::MeteredMachine, run::RunProgram,
    StylusEvmApi,
};
use arbos::programs::types::EvmData;
use core::fmt::Debug;
use revm::{
    context::{
        result::{EVMError, ExecutionResult, InvalidTransaction},
        ContextSetters, Evm as RevmEvm, FrameStack,
    },
    context_interface::{
        host::LoadError,
        result::{HaltReason, ResultAndState},
        ContextTr, JournalTr,
    },
    handler::{
        instructions::EthInstructions,
        EthFrame, EvmTr, FrameResult, Handler, ItemOrResult, MainnetHandler,
        PrecompileProvider,
    },
    inspector::InspectorHandler,
    inspector::NoOpInspector,
    interpreter::{
        interpreter::EthInterpreter,
        interpreter_action::FrameInit,
        interpreter_types::{InputsTr, ReturnData, RuntimeFlag, StackTr},
        CallInput, CallInputs, CallOutcome, CallScheme, FrameInput, Gas as EvmGas, Host,
        InstructionContext, InstructionResult, InterpreterResult, InterpreterTypes,
    },
    primitives::hardfork::SpecId,
    ExecuteEvm, InspectEvm, Inspector, SystemCallEvm,
};

use crate::transaction::ArbTransaction;

/// BLOBBASEFEE opcode (0x4a).
const BLOBBASEFEE_OPCODE: u8 = 0x4a;

/// SELFDESTRUCT opcode (0xff).
const SELFDESTRUCT_OPCODE: u8 = 0xff;

/// NUMBER opcode (0x43).
const NUMBER_OPCODE: u8 = 0x43;

/// BLOCKHASH opcode (0x40).
const BLOCKHASH_OPCODE: u8 = 0x40;

/// BALANCE opcode (0x31).
const BALANCE_OPCODE: u8 = 0x31;

/// Arbitrum NUMBER: returns the L1 block number from ArbOS state.
///
/// Nitro's NUMBER reads from `ProcessingHook.L1BlockNumber()` which returns
/// the value stored by `record_new_l1_block` during StartBlock. The mixHash
/// L1 block number in the header can differ from this value, so we read from
/// the thread-local set after StartBlock processing.
fn arb_number<WIRE: InterpreterTypes, H: Host + ?Sized>(ctx: InstructionContext<'_, H, WIRE>) {
    let l1_block = arb_precompiles::get_l1_block_number_for_evm();
    if !ctx.interpreter.stack.push(U256::from(l1_block)) {
        ctx.interpreter.halt(InstructionResult::StackOverflow);
    }
}

/// Arbitrum BLOCKHASH: uses L1 block number for range check.
///
/// Standard BLOCKHASH compares the requested block number against block_env.number,
/// which is the L2 block number. Since Arbitrum's NUMBER opcode returns the L1
/// block number, BLOCKHASH must also use L1 block numbers for the range check.
/// Otherwise requests for L1 block hashes would always be out of range.
fn arb_blockhash<WIRE: InterpreterTypes, H: Host + ?Sized>(ctx: InstructionContext<'_, H, WIRE>) {
    use revm::interpreter::InstructionResult;

    let requested = match ctx.interpreter.stack.pop() {
        Some(v) => v,
        None => {
            ctx.interpreter.halt(InstructionResult::StackUnderflow);
            return;
        }
    };

    let l1_block_number = U256::from(arb_precompiles::get_l1_block_number_for_evm());

    let Some(diff) = l1_block_number.checked_sub(requested) else {
        if !ctx.interpreter.stack.push(U256::ZERO) {
            ctx.interpreter.halt(InstructionResult::StackOverflow);
        }
        return;
    };

    let diff_u64: u64 = diff.try_into().unwrap_or(u64::MAX);
    if diff_u64 == 0 || diff_u64 > 256 {
        if !ctx.interpreter.stack.push(U256::ZERO) {
            ctx.interpreter.halt(InstructionResult::StackOverflow);
        }
        return;
    }

    let requested_u64: u64 = requested.try_into().unwrap_or(u64::MAX);
    match ctx.host.block_hash(requested_u64) {
        Some(hash) => {
            if !ctx.interpreter.stack.push(U256::from_be_bytes(hash.0)) {
                ctx.interpreter.halt(InstructionResult::StackOverflow);
            }
        }
        None => {
            ctx.interpreter.halt_fatal();
        }
    }
}

// SHA3 tracer removed — can't easily wrap standard handler

/// Arbitrum BALANCE: adjusts the sender's balance by the poster fee correction.
///
/// Nitro's BuyGas charges gas_limit * baseFee, but our reduced gas_limit
/// charges posterGas * baseFee less. When a contract checks BALANCE(sender),
/// we subtract the correction from the result to match Nitro.
fn arb_balance<WIRE: InterpreterTypes, H: Host + ?Sized>(ctx: InstructionContext<'_, H, WIRE>) {
    // Pop address from stack
    let addr_u256 = match ctx.interpreter.stack.pop() {
        Some(v) => v,
        None => {
            ctx.interpreter
                .halt(revm::interpreter::InstructionResult::StackUnderflow);
            return;
        }
    };

    let addr = alloy_primitives::Address::from_word(alloy_primitives::B256::from(
        addr_u256.to_be_bytes::<32>(),
    ));

    // Load account via host (handles cold/warm tracking)
    let spec_id = ctx.interpreter.runtime_flag.spec_id();
    if spec_id.is_enabled_in(revm::primitives::hardfork::SpecId::BERLIN) {
        // Berlin+: use balance() which tracks cold/warm
        let Some(state_load) = ctx.host.balance(addr) else {
            ctx.interpreter.halt_fatal();
            return;
        };
        // Charge gas: 2600 for cold, 100 for warm
        let gas_cost = if state_load.is_cold { 2600u64 } else { 100u64 };
        if !ctx.interpreter.gas.record_cost(gas_cost) {
            ctx.interpreter
                .halt(revm::interpreter::InstructionResult::OutOfGas);
            return;
        }

        // Apply poster fee correction for sender
        let balance = if addr == arb_precompiles::get_current_tx_sender() {
            state_load
                .data
                .saturating_sub(arb_precompiles::get_poster_balance_correction())
        } else {
            state_load.data
        };

        if !ctx.interpreter.stack.push(balance) {
            ctx.interpreter
                .halt(revm::interpreter::InstructionResult::StackOverflow);
        }
    } else {
        // Pre-Berlin: always 400 gas, load via basic path
        let Some(state_load) = ctx.host.balance(addr) else {
            ctx.interpreter.halt_fatal();
            return;
        };

        let balance = if addr == arb_precompiles::get_current_tx_sender() {
            state_load
                .data
                .saturating_sub(arb_precompiles::get_poster_balance_correction())
        } else {
            state_load.data
        };

        if !ctx.interpreter.stack.push(balance) {
            ctx.interpreter
                .halt(revm::interpreter::InstructionResult::StackOverflow);
        }
    }
}

/// SELFBALANCE opcode (0x47).
const SELFBALANCE_OPCODE: u8 = 0x47;

/// Arbitrum SELFBALANCE: adjusts for poster fee correction if the executing
/// contract IS the tx sender (edge case: sender calls own address).
fn arb_selfbalance<WIRE: InterpreterTypes, H: Host + ?Sized>(ctx: InstructionContext<'_, H, WIRE>) {
    let target = ctx.interpreter.input.target_address();

    let Some(state_load) = ctx.host.balance(target) else {
        ctx.interpreter.halt_fatal();
        return;
    };

    // Apply poster fee correction if the contract being executed is the tx sender
    let balance = if target == arb_precompiles::get_current_tx_sender() {
        state_load
            .data
            .saturating_sub(arb_precompiles::get_poster_balance_correction())
    } else {
        state_load.data
    };

    if !ctx.interpreter.stack.push(balance) {
        ctx.interpreter
            .halt(revm::interpreter::InstructionResult::StackOverflow);
    }
}

/// BLOBBASEFEE is not supported on Arbitrum — execution halts.
fn arb_blob_basefee<WIRE: InterpreterTypes, H: Host + ?Sized>(
    ctx: InstructionContext<'_, H, WIRE>,
) {
    ctx.interpreter.halt(InstructionResult::OpcodeNotFound);
}

/// Arbitrum SELFDESTRUCT: reverts if the acting account is a Stylus program,
/// otherwise delegates to the standard EIP-6780 selfdestruct logic.
fn arb_selfdestruct<WIRE: InterpreterTypes, H: Host + ?Sized>(
    ctx: InstructionContext<'_, H, WIRE>,
) {
    if ctx.interpreter.runtime_flag.is_static() {
        ctx.interpreter
            .halt(InstructionResult::StateChangeDuringStaticCall);
        return;
    }

    // Stylus programs cannot be self-destructed.
    let acting_addr = ctx.interpreter.input.target_address();
    match ctx.host.load_account_code(acting_addr) {
        Some(code_load) => {
            if arb_stylus::is_stylus_program(&code_load.data) {
                ctx.interpreter.halt(InstructionResult::Revert);
                return;
            }
        }
        None => {
            ctx.interpreter.halt_fatal();
            return;
        }
    }

    // Standard selfdestruct logic (matching revm's EIP-6780 implementation).
    // Pop U256 and convert to Address manually (avoids pop_address() which
    // triggers a ruint 1.17 const eval panic due to U256->Address byte size mismatch).
    let Some(raw) = ctx.interpreter.stack.pop() else {
        ctx.interpreter.halt(InstructionResult::StackUnderflow);
        return;
    };
    let target = Address::from_word(alloy_primitives::B256::from(raw.to_be_bytes()));

    let spec = ctx.interpreter.runtime_flag.spec_id();
    let cold_load_gas = ctx.host.gas_params().selfdestruct_cold_cost();
    let skip_cold_load = ctx.interpreter.gas.remaining() < cold_load_gas;

    let res = match ctx.host.selfdestruct(acting_addr, target, skip_cold_load) {
        Ok(res) => res,
        Err(LoadError::ColdLoadSkipped) => {
            ctx.interpreter.halt_oog();
            return;
        }
        Err(LoadError::DBError) => {
            ctx.interpreter.halt_fatal();
            return;
        }
    };

    // EIP-161: State trie clearing.
    let should_charge_topup = if spec.is_enabled_in(SpecId::SPURIOUS_DRAGON) {
        res.had_value && !res.target_exists
    } else {
        !res.target_exists
    };

    let gas_cost = ctx
        .host
        .gas_params()
        .selfdestruct_cost(should_charge_topup, res.is_cold);
    if !ctx.interpreter.gas.record_cost(gas_cost) {
        ctx.interpreter.halt_oog();
        return;
    }

    if !res.previously_destroyed {
        ctx.interpreter
            .gas
            .record_refund(ctx.host.gas_params().selfdestruct_refund());
    }

    ctx.interpreter.halt(InstructionResult::SelfDestruct);
}

// ── Stylus page tracking & reentrancy ───────────────────────────────
//
// Page tracking and reentrancy state lives in arb_stylus::pages so that
// both arb-evm (dispatch) and arb-stylus (EvmApi add_pages) can access it.

pub use arb_stylus::pages::{
    add_stylus_pages, get_stylus_pages, get_stylus_program_count, pop_stylus_program,
    push_stylus_program, reset_stylus_pages, set_stylus_pages_open,
};

// ── Stylus storage helpers ───────────────────────────────────────────

use arb_precompiles::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, PROGRAMS_DATA_KEY,
    PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
};
use arbos::programs::{memory::MemoryModel, params::StylusParams, Program};

/// Read a storage slot from ArbOS state via the journal.
fn sload_arbos<DB: Database>(journal: &mut revm::Journal<DB>, slot: U256) -> Option<U256> {
    let _ = journal
        .inner
        .load_account(&mut journal.database, ARBOS_STATE_ADDRESS)
        .ok()?;
    let result = journal
        .inner
        .sload(&mut journal.database, ARBOS_STATE_ADDRESS, slot, false)
        .ok()?;
    Some(result.data)
}

/// Read the StylusParams packed word from storage.
fn read_params_word<DB: Database>(journal: &mut revm::Journal<DB>) -> Option<[u8; 32]> {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    let slot = map_slot(params_key.as_slice(), 0);
    sload_arbos(journal, slot).map(|v| v.to_be_bytes::<32>())
}

/// Read program data word by code hash.
fn read_program_word<DB: Database>(
    journal: &mut revm::Journal<DB>,
    code_hash: B256,
) -> Option<B256> {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let slot = map_slot_b256(data_key.as_slice(), &code_hash);
    sload_arbos(journal, slot).map(|v| B256::from(v.to_be_bytes::<32>()))
}

/// Read the activation-time module hash for a Stylus program by code hash.
/// Stored at programs subspace key [2] (mirroring Nitro's `moduleHashesKey`).
fn read_module_hash<DB: Database>(
    journal: &mut revm::Journal<DB>,
    code_hash: B256,
) -> Option<B256> {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let module_hashes_key = derive_subspace_key(programs_key.as_slice(), &[2]);
    let slot = map_slot_b256(module_hashes_key.as_slice(), &code_hash);
    sload_arbos(journal, slot).map(|v| B256::from(v.to_be_bytes::<32>()))
}

/// Parse essential StylusParams fields from the packed storage word.
/// This mirrors `StylusParams::load()` but works with raw bytes from journal sload.
fn parse_stylus_params(word: &[u8; 32], arbos_version: u64) -> StylusParams {
    StylusParams {
        arbos_version,
        version: u16::from_be_bytes([word[0], word[1]]),
        ink_price: (word[2] as u32) << 16 | (word[3] as u32) << 8 | word[4] as u32,
        max_stack_depth: u32::from_be_bytes([word[5], word[6], word[7], word[8]]),
        free_pages: u16::from_be_bytes([word[9], word[10]]),
        page_gas: u16::from_be_bytes([word[11], word[12]]),
        page_ramp: arbos::programs::params::INITIAL_PAGE_RAMP,
        page_limit: u16::from_be_bytes([word[13], word[14]]),
        min_init_gas: word[15],
        min_cached_init_gas: word[16],
        init_cost_scalar: word[17],
        cached_cost_scalar: word[18],
        expiry_days: u16::from_be_bytes([word[19], word[20]]),
        keepalive_days: u16::from_be_bytes([word[21], word[22]]),
        block_cache_size: u16::from_be_bytes([word[23], word[24]]),
        // These fields span to word 2; not needed for dispatch.
        max_wasm_size: 0,
        max_fragment_count: 0,
    }
}

/// Compute upfront gas cost for a Stylus call, per `Programs.CallProgram`.
fn stylus_call_gas_cost(
    params: &StylusParams,
    program: &Program,
    pages_open: u16,
    pages_ever: u16,
) -> u64 {
    let model = MemoryModel::new(params.free_pages, params.page_gas);
    let mut cost = model.gas_cost(program.footprint, pages_open, pages_ever);

    let cached = program.cached;
    if cached || program.version > 1 {
        cost = cost.saturating_add(program.cached_gas(params));
    }
    if !cached {
        cost = cost.saturating_add(program.init_gas(params));
    }
    cost
}

// ── Stylus sub-call trampolines ─────────────────────────────────────

use arb_stylus::evm_api_impl::{SubCallResult, SubCreateResult};

/// Monomorphized trampoline for Stylus sub-calls (CALL/DELEGATECALL/STATICCALL).
///
/// This function is created as a concrete `fn(...)` pointer by monomorphizing
/// generic type parameters at the call site in `execute_stylus_program`.
/// The `ctx` pointer is cast back to the concrete Context type.
fn stylus_call_trampoline<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
    ctx: *mut (),
    call_type: u8,
    contract: Address,
    caller: Address,
    storage_addr: Address,
    input: &[u8],
    gas: u64,
    value: U256,
) -> SubCallResult
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    let context = unsafe {
        &mut *(ctx as *mut revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>)
    };

    let is_static = call_type == 2;
    let is_delegate = call_type == 1;

    // Create a journal checkpoint for the sub-call
    let checkpoint = context.journaled_state.inner.checkpoint();

    // For CALL with value, transfer ETH
    if !is_delegate && !value.is_zero() {
        let transfer_result = context.journaled_state.inner.transfer(
            &mut context.journaled_state.database,
            caller,
            contract,
            value,
        );
        if transfer_result.is_err() {
            context.journaled_state.inner.checkpoint_revert(checkpoint);
            return SubCallResult {
                output: Vec::new(),
                gas_cost: 0,
                success: false,
                refund: 0,
            };
        }
    }

    let code_address = contract;
    let target_address = storage_addr;
    let _ = is_delegate;

    let call_scheme = match call_type {
        0 => CallScheme::Call,
        1 => CallScheme::DelegateCall,
        2 => CallScheme::StaticCall,
        _ => CallScheme::Call,
    };

    let call_value = if is_delegate {
        revm::interpreter::CallValue::Apparent(value)
    } else {
        revm::interpreter::CallValue::Transfer(value)
    };

    let sub_inputs = CallInputs {
        input: CallInput::Bytes(input.to_vec().into()),
        gas_limit: gas,
        target_address,
        bytecode_address: code_address,
        caller,
        value: call_value,
        scheme: call_scheme,
        is_static,
        return_memory_offset: 0..0,
        known_bytecode: None,
    };

    {
        arb_precompiles::set_evm_depth(context.journaled_state.inner.depth);
        let spec: revm::primitives::hardfork::SpecId = context.cfg.spec().into();
        let mut precompiles =
            alloy_evm::precompiles::PrecompilesMap::from(revm::handler::EthPrecompiles::new(spec));
        register_arb_precompiles(&mut precompiles, arb_precompiles::get_arbos_version());
        let mut arb_map = ArbPrecompilesMap(precompiles);
        let dispatch_result = <ArbPrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::run(&mut arb_map, context, &sub_inputs);

        match dispatch_result {
            Ok(Some(result)) => {
                let success = result.result.is_ok();
                let output = result.output.to_vec();
                let gas_used = gas.saturating_sub(result.gas.remaining());
                let refund = result.gas.refunded();
                if success {
                    context.journaled_state.inner.checkpoint_commit();
                } else {
                    context.journaled_state.inner.checkpoint_revert(checkpoint);
                }
                return SubCallResult {
                    output,
                    gas_cost: gas_used,
                    success,
                    refund,
                };
            }
            Ok(None) => {}
            Err(_) => {
                context.journaled_state.inner.checkpoint_revert(checkpoint);
                return SubCallResult {
                    output: Vec::new(),
                    gas_cost: 0,
                    success: false,
                    refund: 0,
                };
            }
        }
    }

    let bytecode = match context
        .journaled_state
        .inner
        .load_code(&mut context.journaled_state.database, code_address)
    {
        Ok(acc) => acc
            .data
            .info
            .code
            .as_ref()
            .map(|c| c.original_bytes())
            .unwrap_or_default(),
        Err(_) => {
            context.journaled_state.inner.checkpoint_revert(checkpoint);
            return SubCallResult {
                output: Vec::new(),
                gas_cost: 0,
                success: false,
                refund: 0,
            };
        }
    };

    if bytecode.is_empty() {
        context.journaled_state.inner.checkpoint_commit();
        return SubCallResult {
            output: Vec::new(),
            gas_cost: 0,
            success: true,
            refund: 0,
        };
    }

    // Stylus → Stylus call: detect the Stylus discriminant on the loaded
    // bytecode and dispatch through the WASM runtime instead of running the
    // bytes as raw EVM. Without this branch, the inner call hits 0xef on the
    // very first byte and aborts with OpcodeNotFound. Block 55755413's failing
    // tx tripped this when a Solidity contract called a Stylus token's
    // `transferFrom` from inside a Stylus → EVM CALL chain.
    if arb_precompiles::get_arbos_version() >= arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS
        && arb_stylus::is_stylus_program(&bytecode)
    {
        let result = execute_stylus_program(context, &sub_inputs, &bytecode);
        let success = result.result.is_ok();
        let output = result.output.to_vec();
        let gas_used = gas.saturating_sub(result.gas.remaining());
        let refund = result.gas.refunded();
        if success {
            context.journaled_state.inner.checkpoint_commit();
        } else {
            context.journaled_state.inner.checkpoint_revert(checkpoint);
        }
        return SubCallResult {
            output,
            gas_cost: gas_used,
            success,
            refund,
        };
    }

    let result = run_evm_bytecode(context, &sub_inputs, &bytecode, gas);
    let success = result.result.is_ok();
    let output = result.output.to_vec();
    let gas_used = gas.saturating_sub(result.gas.remaining());
    let refund = result.gas.refunded();
    if success {
        context.journaled_state.inner.checkpoint_commit();
    } else {
        context.journaled_state.inner.checkpoint_revert(checkpoint);
    }
    SubCallResult {
        output,
        gas_cost: gas_used,
        success,
        refund,
    }
}

/// Monomorphized trampoline for Stylus CREATE/CREATE2 operations.
fn stylus_create_trampoline<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
    ctx: *mut (),
    caller: Address,
    code: &[u8],
    gas: u64,
    endowment: U256,
    salt: Option<B256>,
) -> SubCreateResult
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    let context = unsafe {
        &mut *(ctx as *mut revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>)
    };

    let checkpoint = context.journaled_state.inner.checkpoint();

    // Compute CREATE/CREATE2 address
    let caller_nonce = {
        let acc = context
            .journaled_state
            .inner
            .load_account(&mut context.journaled_state.database, caller);
        acc.map(|a| a.data.info.nonce).unwrap_or(0)
    };

    let created_address = if let Some(salt) = salt {
        // CREATE2: keccak256(0xff ++ sender ++ salt ++ keccak256(code))
        let code_hash = alloy_primitives::keccak256(code);
        let mut buf = Vec::with_capacity(1 + 20 + 32 + 32);
        buf.push(0xff);
        buf.extend_from_slice(caller.as_slice());
        buf.extend_from_slice(salt.as_slice());
        buf.extend_from_slice(code_hash.as_slice());
        Address::from_slice(&alloy_primitives::keccak256(&buf)[12..])
    } else {
        // CREATE: RLP([sender, nonce])
        use alloy_rlp::Encodable;
        let mut rlp_buf = Vec::with_capacity(64);
        alloy_rlp::Header {
            list: true,
            payload_length: caller.length() + caller_nonce.length(),
        }
        .encode(&mut rlp_buf);
        caller.encode(&mut rlp_buf);
        caller_nonce.encode(&mut rlp_buf);
        Address::from_slice(&alloy_primitives::keccak256(&rlp_buf)[12..])
    };

    // Increment caller nonce
    let _ = context
        .journaled_state
        .inner
        .load_account(&mut context.journaled_state.database, caller);
    if let Some(acc) = context.journaled_state.inner.state.get_mut(&caller) {
        acc.info.nonce += 1;
        context
            .journaled_state
            .inner
            .nonce_bump_journal_entry(caller);
    }

    // Transfer endowment
    if !endowment.is_zero()
        && context
            .journaled_state
            .inner
            .transfer(
                &mut context.journaled_state.database,
                caller,
                created_address,
                endowment,
            )
            .is_err()
    {
        context.journaled_state.inner.checkpoint_revert(checkpoint);
        return SubCreateResult {
            address: None,
            output: Vec::new(),
            gas_cost: gas,
        };
    }

    // Run init code as EVM
    let init_inputs = CallInputs {
        input: CallInput::Bytes(code.to_vec().into()),
        gas_limit: gas,
        target_address: created_address,
        bytecode_address: created_address,
        caller,
        value: revm::interpreter::CallValue::Transfer(endowment),
        scheme: CallScheme::Call,
        is_static: false,
        return_memory_offset: 0..0,
        known_bytecode: None,
    };

    let result = run_evm_bytecode(context, &init_inputs, code, gas);
    let success = result.result.is_ok();
    let gas_used = gas.saturating_sub(result.gas.remaining());

    if success {
        // Store the returned bytecode as the contract's code
        let deployed_code = result.output.to_vec();
        let code_hash = alloy_primitives::keccak256(&deployed_code);
        let bytecode = revm::bytecode::Bytecode::new_raw(deployed_code.into());
        // Ensure the account is loaded into state
        let _ = context
            .journaled_state
            .inner
            .load_account(&mut context.journaled_state.database, created_address);
        context
            .journaled_state
            .inner
            .set_code_with_hash(created_address, bytecode, code_hash);
        context.journaled_state.inner.checkpoint_commit();
        SubCreateResult {
            address: Some(created_address),
            output: Vec::new(), // success doesn't return data
            gas_cost: gas_used,
        }
    } else {
        let output = result.output.to_vec();
        context.journaled_state.inner.checkpoint_revert(checkpoint);
        SubCreateResult {
            address: None,
            output, // revert returns data
            gas_cost: gas_used,
        }
    }
}

/// Run EVM bytecode from a Stylus sub-call.
///
/// Creates an interpreter and runs in a loop, dispatching nested CALL/CREATE
/// actions through the Stylus call trampoline (which in turn uses
/// ArbPrecompilesMap for precompile/Stylus dispatch).
fn run_evm_bytecode<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
    context: &mut revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
    inputs: &CallInputs,
    bytecode: &[u8],
    gas_limit: u64,
) -> InterpreterResult
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    use revm::{
        bytecode::Bytecode,
        interpreter::{
            interpreter::{ExtBytecode, InputsImpl},
            FrameInput, InterpreterAction, SharedMemory,
        },
    };

    let code = Bytecode::new_raw(bytecode.to_vec().into());
    let ext_bytecode = ExtBytecode::new(code);

    let call_value = inputs.value.get();
    let interp_input = InputsImpl {
        target_address: inputs.target_address,
        bytecode_address: Some(inputs.bytecode_address),
        caller_address: inputs.caller,
        input: inputs.input.clone(),
        call_value,
    };

    let spec = context.cfg.spec();

    let mut interpreter = revm::interpreter::Interpreter::new(
        SharedMemory::new(),
        ext_bytecode,
        interp_input,
        inputs.is_static,
        spec.clone().into(),
        gas_limit,
    );

    // Build instruction table with our custom Arbitrum opcode handlers.
    // This MUST install the same set as the outer ArbEvmFactory (search for
    // `instruction.insert_instruction` further down in this file). When a
    // Stylus contract calls into an EVM contract via stylus_call_trampoline,
    // the new EVM frame goes through this code path. Without these handlers,
    // sub-frame NUMBER returns the L2 block number instead of the L1 block
    // number Arbitrum exposes — which silently breaks any EIP-712 signature
    // derivation that includes block.number in its digest. Block 55755413's
    // failing tx reproduced this as an `InvalidSignature()` revert deep
    // inside a Stylus -> EVM CALL chain.
    type Ctx<B, T, C, D, Ch> = revm::Context<B, T, C, D, revm::Journal<D>, Ch>;
    let mut instructions = EthInstructions::<
        EthInterpreter,
        Ctx<BlockEnv, TxEnv, CfgEnv, DB, Chain>,
    >::new_mainnet_with_spec(spec.into());
    instructions.insert_instruction(
        BLOBBASEFEE_OPCODE,
        revm::interpreter::Instruction::new(arb_blob_basefee, 2),
    );
    instructions.insert_instruction(
        SELFDESTRUCT_OPCODE,
        revm::interpreter::Instruction::new(arb_selfdestruct, 5000),
    );
    instructions.insert_instruction(
        NUMBER_OPCODE,
        revm::interpreter::Instruction::new(arb_number, 2),
    );
    instructions.insert_instruction(
        BLOCKHASH_OPCODE,
        revm::interpreter::Instruction::new(arb_blockhash, 20),
    );
    instructions.insert_instruction(
        BALANCE_OPCODE,
        revm::interpreter::Instruction::new(arb_balance, 0),
    );
    instructions.insert_instruction(
        SELFBALANCE_OPCODE,
        revm::interpreter::Instruction::new(arb_selfbalance, 5),
    );

    // Run the interpreter in a loop, handling nested calls/creates
    loop {
        let action = interpreter.run_plain(&instructions.instruction_table, context);

        match action {
            InterpreterAction::Return(result) => {
                return result;
            }
            InterpreterAction::NewFrame(FrameInput::Call(sub_call)) => {
                // Dispatch nested call through our trampoline.
                // For DELEGATECALL, target_address is the storage context (preserved
                // from parent), and caller is the msg.sender (preserved). For
                // CALL/STATICCALL, target_address == bytecode_address.
                //
                // Resolve `sub_call.input` against the *interpreter's* SharedMemory.
                // We can't use `sub_call.input.bytes(context)` because that reads
                // from `context.local()` — the global context's local memory —
                // whereas this frame's CALL/STATICCALL was issued against the
                // private SharedMemory we created at the top of this function.
                // Reading from the wrong memory returns garbage and silently
                // corrupts the input to ecrecover, transferFrom, etc.
                let resolved_input: Bytes = match &sub_call.input {
                    revm::interpreter::CallInput::Bytes(b) => b.clone(),
                    revm::interpreter::CallInput::SharedBuffer(range) => {
                        // The range was computed by call_helpers as
                        //   range.start = relative_offset + local_memory_offset()
                        // i.e. an absolute offset into the SharedMemory buffer.
                        Bytes::from(interpreter.memory.global_slice_range(range.clone()).to_vec())
                    }
                };
                let bytecode_address = sub_call.bytecode_address;
                let sub_result = stylus_call_trampoline::<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
                    context as *mut _ as *mut (),
                    match sub_call.scheme {
                        CallScheme::Call | CallScheme::CallCode => 0,
                        CallScheme::DelegateCall => 1,
                        CallScheme::StaticCall => 2,
                    },
                    bytecode_address,
                    sub_call.caller,
                    sub_call.target_address,
                    &resolved_input,
                    sub_call.gas_limit,
                    sub_call.value.get(),
                );

                // Inject result back into interpreter (matching EthFrame::return_result)
                let gas_remaining = sub_call.gas_limit.saturating_sub(sub_result.gas_cost);
                let ins_result = if sub_result.success {
                    InstructionResult::Return
                } else {
                    InstructionResult::Revert
                };

                let output: Bytes = sub_result.output.into();
                let returned_len = output.len();
                let mem_start = sub_call.return_memory_offset.start;
                let mem_length = sub_call.return_memory_offset.len();
                let target_len = mem_length.min(returned_len);

                interpreter.return_data.set_buffer(output);

                let item = if ins_result.is_ok() {
                    U256::from(1)
                } else {
                    U256::ZERO
                };
                let _ = interpreter.stack.push(item);

                if ins_result.is_ok_or_revert() {
                    interpreter.gas.erase_cost(gas_remaining);
                    if target_len > 0 {
                        interpreter
                            .memory
                            .set(mem_start, &interpreter.return_data.buffer()[..target_len]);
                    }
                }

                if ins_result.is_ok() {
                    // Propagate the sub-call's SSTORE refund (EIP-3529) up
                    // through the Stylus -> EVM bytecode -> Stylus call chain.
                    // Mirrors revm's `EthFrame::return_result` which also does
                    // `interpreter.gas.record_refund(out_gas.refunded())` on
                    // success. Without this, a Stylus contract that goes
                    // through a Solidity intermediary to reach an inner
                    // contract that clears storage loses the 4800-gas clearing
                    // refund (block 55,755,413 tx 2).
                    interpreter.gas.record_refund(sub_result.refund);
                }
            }
            InterpreterAction::NewFrame(FrameInput::Create(sub_create)) => {
                // Dispatch create through our trampoline
                let salt = match sub_create.scheme() {
                    revm::interpreter::CreateScheme::Create2 { salt } => {
                        Some(B256::from(salt.to_be_bytes()))
                    }
                    _ => None,
                };

                let sub_result = stylus_create_trampoline::<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
                    context as *mut _ as *mut (),
                    sub_create.caller(),
                    sub_create.init_code(),
                    sub_create.gas_limit(),
                    sub_create.value(),
                    salt,
                );

                let gas_remaining = sub_create.gas_limit().saturating_sub(sub_result.gas_cost);
                let created_addr = sub_result.address;

                let ins_result = if created_addr.is_some() {
                    InstructionResult::Return
                } else if !sub_result.output.is_empty() {
                    InstructionResult::Revert
                } else {
                    InstructionResult::CreateInitCodeStartingEF00
                };

                let output: Bytes = sub_result.output.into();
                interpreter.return_data.set_buffer(output);

                // Push created address or zero
                let item = match created_addr {
                    Some(addr) => addr.into_word().into(),
                    None => U256::ZERO,
                };
                let _ = interpreter.stack.push(item);

                if ins_result.is_ok_or_revert() {
                    interpreter.gas.erase_cost(gas_remaining);
                }
            }
            InterpreterAction::NewFrame(FrameInput::Empty) => {
                // Should not happen
                return InterpreterResult::new(
                    InstructionResult::Revert,
                    Bytes::new(),
                    EvmGas::new(0),
                );
            }
        }
    }
}

// ── Stylus WASM dispatch ────────────────────────────────────────────

/// Execute a Stylus WASM program by creating a NativeInstance and running it.
///
/// Validates the program, computes upfront gas costs (memory pages + init/cached
/// gas), deducts them, then runs the WASM.
fn execute_stylus_program<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
    context: &mut revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
    inputs: &CallInputs,
    bytecode: &[u8],
) -> InterpreterResult
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    use arbos::programs::types::UserOutcome;

    let zero_gas = || EvmGas::new(0);

    let code_hash = alloy_primitives::keccak256(bytecode);
    let arbos_version = arb_precompiles::get_arbos_version();
    let block_timestamp = arb_precompiles::get_block_timestamp();

    // ── Read and validate program metadata ──────────────────────────
    let params_word = match read_params_word(&mut context.journaled_state) {
        Some(w) => w,
        None => {
            tracing::warn!(target: "stylus", "failed to read StylusParams from storage");
            return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
        }
    };
    let params = parse_stylus_params(&params_word, arbos_version);

    let program_word = match read_program_word(&mut context.journaled_state, code_hash) {
        Some(w) => w,
        None => {
            tracing::warn!(target: "stylus", codehash = %code_hash, "failed to read program data");
            return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
        }
    };
    let program = Program::from_storage(program_word, block_timestamp);

    // Validate: program must be activated, correct version, not expired.
    if program.version == 0 || program.version != params.version {
        tracing::warn!(target: "stylus", codehash = %code_hash, program_ver = program.version, params_ver = params.version, "program version mismatch");
        return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
    }
    let expiry_seconds = (params.expiry_days as u64) * 24 * 3600;
    if program.age_seconds > expiry_seconds {
        tracing::warn!(target: "stylus", codehash = %code_hash, "program expired");
        return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
    }

    // ── Compute and deduct upfront gas costs ────────────────────────
    let (pages_open, pages_ever) = get_stylus_pages();
    // ArbOS v60+: recent WASMs cache hit makes the program count as cached
    // for the purposes of gas pricing (mirrors Nitro's GetRecentWasms.Insert).
    let recent_wasms_hit = if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_60 {
        arb_precompiles::insert_recent_wasm(code_hash)
    } else {
        false
    };
    let effective_cached = program.cached || recent_wasms_hit;
    let effective_program = if effective_cached != program.cached {
        let mut p = program.clone();
        p.cached = effective_cached;
        p
    } else {
        program.clone()
    };
    let upfront_cost = stylus_call_gas_cost(&params, &effective_program, pages_open, pages_ever);
    let total_gas = inputs.gas_limit;

    if total_gas < upfront_cost {
        return InterpreterResult::new(InstructionResult::OutOfGas, Bytes::new(), zero_gas());
    }
    let gas_for_wasm = total_gas - upfront_cost;

    let stylus_config = StylusConfig::new(params.version, params.max_stack_depth, params.ink_price);

    // ── Track reentrancy ────────────────────────────────────────────
    let target_addr = inputs.target_address;
    let is_delegate = matches!(
        inputs.scheme,
        CallScheme::DelegateCall | CallScheme::CallCode
    );
    // Only non-delegate-non-callcode calls increment the reentrancy counter.
    // Delegate and callcode frames check the counter without bumping it, so an
    // actual re-entry into the same storage context reports `reentrant=true`.
    let reentrant = if !is_delegate {
        push_stylus_program(target_addr)
    } else {
        get_stylus_program_count(target_addr) > 1
    };

    // Read the activation-time module hash from storage. This differs from
    // code_hash (which is keccak256 of the bytecode); it is the hash of the
    // compiled module computed during activateProgram.
    let module_hash = read_module_hash(&mut context.journaled_state, code_hash)
        .unwrap_or(code_hash);

    // Build EvmData from the execution context.
    let mut evm_data = build_evm_data(context, inputs);
    evm_data.reentrant = reentrant as u32;
    evm_data.cached = effective_program.cached;
    evm_data.module_hash = module_hash;

    // Track pages — add this program's footprint.
    let (prev_open, _prev_ever) = add_stylus_pages(program.footprint);

    // Create the type-erased StylusEvmApi bridge.
    let journal_ptr = &mut context.journaled_state as *mut revm::Journal<DB>;
    let is_static = inputs.is_static || matches!(inputs.scheme, CallScheme::StaticCall);
    let ctx_ptr = context as *mut _ as *mut ();
    let caller = inputs.caller;
    let call_value = inputs.value.get();
    let evm_api = unsafe {
        StylusEvmApi::new(
            journal_ptr,
            target_addr,
            caller,
            call_value,
            is_static,
            params.free_pages,
            params.page_gas,
            ctx_ptr,
            Some(stylus_call_trampoline::<BlockEnv, TxEnv, CfgEnv, DB, Chain>),
            Some(stylus_create_trampoline::<BlockEnv, TxEnv, CfgEnv, DB, Chain>),
        )
    };

    // Try the module cache first; compile from WASM on miss and populate cache.
    let long_term_tag = if program.cached { 1u32 } else { 0u32 };
    let mut instance = if let Some((module, store)) =
        arb_stylus::cache::InitCache::get(code_hash, params.version, long_term_tag, false)
    {
        let compile = arb_stylus::CompileConfig::version(params.version, false);
        let env = arb_stylus::env::WasmEnv::new(compile, Some(stylus_config), evm_api, evm_data);
        match arb_stylus::NativeInstance::from_module(module, store, env) {
            Ok(inst) => inst,
            Err(e) => {
                tracing::warn!(target: "stylus", codehash = %code_hash, err = %e, "failed from cached module");
                set_stylus_pages_open(prev_open);
                if !is_delegate {
                    pop_stylus_program(target_addr);
                }
                return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
            }
        }
    } else {
        let decompressed = match arb_stylus::decompress_wasm(bytecode) {
            Ok(w) => w,
            Err(e) => {
                tracing::warn!(target: "stylus", codehash = %code_hash, err = %e, "WASM decompression failed");
                set_stylus_pages_open(prev_open);
                if !is_delegate {
                    pop_stylus_program(target_addr);
                }
                return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
            }
        };
        let compile = arb_stylus::CompileConfig::version(params.version, false);
        match arb_stylus::NativeInstance::from_bytes(
            &decompressed,
            evm_api,
            evm_data,
            &compile,
            stylus_config,
        ) {
            Ok(inst) => inst,
            Err(e) => {
                tracing::warn!(target: "stylus", codehash = %code_hash, err = %e, "failed to compile WASM");
                set_stylus_pages_open(prev_open);
                if !is_delegate {
                    pop_stylus_program(target_addr);
                }
                return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
            }
        }
    };

    // Convert EVM gas (after upfront deduction) to ink.
    let ink = stylus_config.pricing.gas_to_ink(StylusGas(gas_for_wasm));

    // Get calldata from CallInput enum. SharedBuffer references parent's
    // memory and must be resolved via the context.
    let calldata_owned: Bytes = inputs.input.bytes(context);
    let calldata: &[u8] = &calldata_owned;
    let outcome = match instance.run_main(calldata, stylus_config, ink) {
        Ok(outcome) => outcome,
        Err(e) => {
            tracing::warn!(target: "stylus", codehash = %code_hash, err = %e, "WASM execution failed");
            set_stylus_pages_open(prev_open);
            if !is_delegate {
                pop_stylus_program(target_addr);
            }
            return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
        }
    };

    // Restore page count and pop reentrancy.
    set_stylus_pages_open(prev_open);
    if !is_delegate {
        pop_stylus_program(target_addr);
    }

    // Convert remaining ink back to gas.
    let ink_left = match instance.ink_left() {
        arb_stylus::MachineMeter::Ready(ink_val) => ink_val,
        arb_stylus::MachineMeter::Exhausted => arb_stylus::Ink(0),
    };
    let gas_left = stylus_config.pricing.ink_to_gas(ink_left).0;

    // Return data cost parity with EVM (ArbOS >= StylusFixes).
    let output: Bytes = instance.env().outs.clone().into();
    let gas_left = if !output.is_empty()
        && arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_FIXES
    {
        let evm_cost = arbos::programs::types::evm_memory_cost(output.len() as u64);
        if total_gas < evm_cost {
            0
        } else {
            gas_left.min(total_gas - evm_cost)
        }
    } else {
        gas_left
    };

    let mut gas_result = EvmGas::new(gas_left);
    // Propagate SSTORE refunds from Stylus flush to the EVM gas accounting.
    let sstore_refund = instance.env().evm_api.sstore_refund();
    if sstore_refund != 0 {
        gas_result.record_refund(sstore_refund);
    }

    // Map UserOutcome to InterpreterResult.
    match outcome {
        UserOutcome::Success => {
            InterpreterResult::new(InstructionResult::Return, output, gas_result)
        }
        UserOutcome::Revert => {
            InterpreterResult::new(InstructionResult::Revert, output, gas_result)
        }
        UserOutcome::OutOfInk => {
            InterpreterResult::new(InstructionResult::OutOfGas, Bytes::new(), zero_gas())
        }
        UserOutcome::OutOfStack => {
            InterpreterResult::new(InstructionResult::CallTooDeep, Bytes::new(), zero_gas())
        }
        UserOutcome::Failure => {
            InterpreterResult::new(InstructionResult::Revert, Bytes::new(), gas_result)
        }
    }
}

/// Build [`EvmData`] from the current execution context.
fn build_evm_data<BlockEnv, TxEnv, CfgEnv, DB, Chain>(
    context: &revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
    inputs: &CallInputs,
) -> EvmData
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    let basefee = U256::from(context.block.basefee());
    let gas_price = U256::from(context.tx.gas_price());
    let value = inputs.value.get();

    // Stylus's block.number must be the L1 block number, not the L2 block number.
    // Nitro uses `evm.ProcessingHook.L1BlockNumber(evm.Context)` for this field.
    let l1_block_number = arb_precompiles::get_l1_block_number_for_evm();

    EvmData {
        arbos_version: arb_precompiles::get_arbos_version(),
        block_basefee: B256::from(basefee.to_be_bytes()),
        chain_id: context.cfg.chain_id(),
        block_coinbase: context.block.beneficiary(),
        block_gas_limit: context.block.gas_limit(),
        block_number: l1_block_number,
        block_timestamp: context.block.timestamp().saturating_to(),
        contract_address: inputs.target_address,
        module_hash: alloy_primitives::keccak256(b""),
        msg_sender: inputs.caller,
        msg_value: B256::from(value.to_be_bytes()),
        tx_gas_price: B256::from(gas_price.to_be_bytes()),
        tx_origin: context.tx.caller(),
        reentrant: 0,
        cached: false,
        tracing: false,
    }
}

// ── Stylus frame-level intercept ──────────────────────────────────

/// Check if a `FrameInit` targets a Stylus WASM program via `known_bytecode`.
fn is_stylus_call(frame_init: &FrameInit) -> Option<Bytes> {
    if arb_precompiles::get_arbos_version() < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS {
        return None;
    }
    if let FrameInput::Call(ref inputs) = frame_init.frame_input {
        if let Some((_, ref code)) = inputs.known_bytecode {
            let raw = code.original_bytes();
            if arb_stylus::is_stylus_program(&raw) {
                return Some(raw);
            }
        }
    }
    None
}

/// Execute a Stylus call and return the appropriate `FrameResult`, handling
/// journal checkpoint commit/revert.
fn execute_stylus_call_concrete<DB: Database>(
    ctx: &mut EthEvmContext<DB>,
    inputs: &CallInputs,
    bytecode: &[u8],
    checkpoint: revm::context_interface::journaled_state::JournalCheckpoint,
) -> FrameResult {
    // Handle value transfer for non-delegate calls (matches EthFrame::make_call_frame).
    if let revm::interpreter::CallValue::Transfer(value) = inputs.value {
        if let Some(i) = ctx.journal_mut().transfer_loaded(
            inputs.caller,
            inputs.target_address,
            value,
        ) {
            let gas = EvmGas::new(inputs.gas_limit);
            ctx.journaled_state.inner.checkpoint_revert(checkpoint);
            return FrameResult::Call(CallOutcome {
                result: InterpreterResult::new(i.into(), Bytes::new(), gas),
                memory_offset: inputs.return_memory_offset.clone(),
                was_precompile_called: false,
                precompile_call_logs: Vec::new(),
            });
        }
    }

    let result = execute_stylus_program(ctx, inputs, bytecode);

    if result.result.is_ok() {
        ctx.journaled_state.inner.checkpoint_commit();
    } else {
        ctx.journaled_state.inner.checkpoint_revert(checkpoint);
    }
    FrameResult::Call(CallOutcome {
        result,
        memory_offset: inputs.return_memory_offset.clone(),
        was_precompile_called: false,
        precompile_call_logs: Vec::new(),
    })
}

// ── Depth-tracking precompile provider ─────────────────────────────

/// Wraps [`PrecompilesMap`] to set the thread-local EVM call depth before
/// each precompile invocation. The depth is read from revm's journal, which
/// mirrors the `evm.Depth()` counter used by `ArbSys.isTopLevelCall`.
#[derive(Clone, Debug)]
pub struct ArbPrecompilesMap(pub PrecompilesMap);

impl<BlockEnv, TxEnv, CfgEnv, DB, Chain>
    PrecompileProvider<revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>>
    for ArbPrecompilesMap
where
    BlockEnv: revm::context::Block,
    TxEnv: revm::context::Transaction,
    CfgEnv: revm::context::Cfg,
    DB: Database,
{
    type Output = InterpreterResult;

    fn set_spec(&mut self, spec: CfgEnv::Spec) -> bool {
        <PrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::set_spec(&mut self.0, spec)
    }

    fn run(
        &mut self,
        context: &mut revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        inputs: &CallInputs,
    ) -> Result<Option<Self::Output>, String> {
        // Sync the thread-local depth from revm's journal before the precompile runs.
        arb_precompiles::set_evm_depth(context.journaled_state.inner.depth);

        // Check precompiles first.
        if let result @ Some(_) = <PrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::run(&mut self.0, context, inputs)?
        {
            return Ok(result);
        }

        // Check for Stylus WASM programs (active at ArbOS v31+).
        let arbos_version = arb_precompiles::get_arbos_version();
        if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS {
            // Use known_bytecode from CallInputs if available (already loaded by
            // revm's CALL handler), otherwise load from journal.
            let bytecode = inputs
                .known_bytecode
                .as_ref()
                .map(|(_, code)| code.original_bytes())
                .or_else(|| {
                    context
                        .journaled_state
                        .inner
                        .load_code(
                            &mut context.journaled_state.database,
                            inputs.bytecode_address,
                        )
                        .ok()
                        .and_then(|acc| {
                            acc.data.info.code.as_ref().map(|c| c.original_bytes())
                        })
                });

            if let Some(bytecode) = bytecode {
                if arb_stylus::is_stylus_program(&bytecode) {
                    return Ok(Some(execute_stylus_program(context, inputs, &bytecode)));
                }
            }
        }

        Ok(None)
    }

    fn warm_addresses(&self) -> Box<impl Iterator<Item = Address>> {
        <PrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::warm_addresses(&self.0)
    }

    fn contains(&self, address: &Address) -> bool {
        <PrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::contains(&self.0, address)
    }
}

// ── ArbEvm ─────────────────────────────────────────────────────────

type InnerRevmEvm<DB, I> = RevmEvm<
    EthEvmContext<DB>,
    I,
    EthInstructions<EthInterpreter, EthEvmContext<DB>>,
    ArbPrecompilesMap,
    EthFrame,
>;

/// Arbitrum EVM with frame-level Stylus dispatch and custom opcodes.
///
/// Implements [`EvmTr`] directly to intercept Stylus WASM calls in
/// `frame_init` before they reach the precompile provider.
pub struct ArbEvm<DB: Database, I> {
    inner: InnerRevmEvm<DB, I>,
    inspect: bool,
}

impl<DB, I> ArbEvm<DB, I>
where
    DB: Database,
{
    pub fn new(inner: InnerRevmEvm<DB, I>, inspect: bool) -> Self {
        Self { inner, inspect }
    }

    pub fn into_inner(self) -> InnerRevmEvm<DB, I> {
        self.inner
    }

    pub fn ctx(&self) -> &EthEvmContext<DB> {
        &self.inner.ctx
    }

    pub fn ctx_mut(&mut self) -> &mut EthEvmContext<DB> {
        &mut self.inner.ctx
    }

    pub fn precompiles_mut(&mut self) -> &mut ArbPrecompilesMap {
        &mut self.inner.precompiles
    }
}

impl<DB: Database, I> core::ops::Deref for ArbEvm<DB, I> {
    type Target = EthEvmContext<DB>;
    fn deref(&self) -> &Self::Target {
        &self.inner.ctx
    }
}

impl<DB: Database, I> core::ops::DerefMut for ArbEvm<DB, I> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.inner.ctx
    }
}

// ── EvmTr for ArbEvm ─────────────────────────────────────────────

impl<DB, I> EvmTr for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
{
    type Context = EthEvmContext<DB>;
    type Instructions = EthInstructions<EthInterpreter, EthEvmContext<DB>>;
    type Precompiles = ArbPrecompilesMap;
    type Frame = EthFrame<EthInterpreter>;

    #[inline]
    fn all(
        &self,
    ) -> (
        &Self::Context,
        &Self::Instructions,
        &Self::Precompiles,
        &FrameStack<Self::Frame>,
    ) {
        self.inner.all()
    }

    #[inline]
    fn all_mut(
        &mut self,
    ) -> (
        &mut Self::Context,
        &mut Self::Instructions,
        &mut Self::Precompiles,
        &mut FrameStack<Self::Frame>,
    ) {
        self.inner.all_mut()
    }

    #[inline]
    fn frame_init(
        &mut self,
        frame_input: FrameInit,
    ) -> Result<
        ItemOrResult<&mut Self::Frame, FrameResult>,
        revm::handler::evm::ContextDbError<Self::Context>,
    > {
        // Intercept Stylus WASM calls before they reach EthFrame/precompiles.
        if let Some(bytecode) = is_stylus_call(&frame_input) {
            if let FrameInput::Call(ref inputs) = frame_input.frame_input {
                if frame_input.depth > revm::primitives::constants::CALL_STACK_LIMIT as usize {
                    let gas = EvmGas::new(inputs.gas_limit);
                    return Ok(ItemOrResult::Result(FrameResult::Call(CallOutcome {
                        result: InterpreterResult::new(
                            InstructionResult::CallTooDeep,
                            Bytes::new(),
                            gas,
                        ),
                        memory_offset: inputs.return_memory_offset.clone(),
                        was_precompile_called: false,
                        precompile_call_logs: Vec::new(),
                    })));
                }
                let checkpoint = self.inner.ctx.journal_mut().checkpoint();
                let result = execute_stylus_call_concrete(
                    &mut self.inner.ctx,
                    inputs,
                    &bytecode,
                    checkpoint,
                );
                return Ok(ItemOrResult::Result(result));
            }
        }

        // Non-Stylus: delegate to the inner RevmEvm's EvmTr implementation.
        self.inner.frame_init(frame_input)
    }

    #[inline]
    fn frame_run(
        &mut self,
    ) -> Result<ItemOrResult<FrameInit, FrameResult>, revm::handler::evm::ContextDbError<Self::Context>>
    {
        self.inner.frame_run()
    }

    #[inline]
    fn frame_return_result(
        &mut self,
        result: FrameResult,
    ) -> Result<Option<FrameResult>, revm::handler::evm::ContextDbError<Self::Context>> {
        self.inner.frame_return_result(result)
    }
}

// ── ExecuteEvm for ArbEvm ────────────────────────────────────────

impl<DB, I> ExecuteEvm for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
{
    type ExecutionResult = ExecutionResult<HaltReason>;
    type State = revm::state::EvmState;
    type Error = EVMError<<DB as revm::Database>::Error, InvalidTransaction>;
    type Tx = revm::context::TxEnv;
    type Block = revm::context::BlockEnv;

    #[inline]
    fn transact_one(&mut self, tx: Self::Tx) -> Result<Self::ExecutionResult, Self::Error> {
        self.inner.ctx.set_tx(tx);
        MainnetHandler::default().run(self)
    }

    #[inline]
    fn finalize(&mut self) -> Self::State {
        self.inner.ctx.journal_mut().finalize()
    }

    #[inline]
    fn set_block(&mut self, block: Self::Block) {
        self.inner.ctx.set_block(block);
    }

    #[inline]
    fn replay(&mut self) -> Result<ResultAndState<HaltReason>, Self::Error> {
        MainnetHandler::default().run(self).map(|result| {
            let state = self.finalize();
            ResultAndState::new(result, state)
        })
    }
}

// ── SystemCallEvm for ArbEvm ─────────────────────────────────────

impl<DB, I> SystemCallEvm for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
{
    fn system_call_one_with_caller(
        &mut self,
        caller: Address,
        system_contract_address: Address,
        data: Bytes,
    ) -> Result<Self::ExecutionResult, Self::Error> {
        use revm::handler::system_call::SystemCallTx;
        self.inner.ctx.set_tx(revm::context::TxEnv::new_system_tx_with_caller(
            caller,
            system_contract_address,
            data,
        ));
        MainnetHandler::default().run_system_call(self)
    }
}

// ── InspectorEvmTr for ArbEvm ────────────────────────────────────

impl<DB, I> revm::inspector::InspectorEvmTr for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
    revm::Journal<DB>: revm::inspector::JournalExt,
{
    type Inspector = I;

    fn all_inspector(
        &self,
    ) -> (
        &Self::Context,
        &Self::Instructions,
        &Self::Precompiles,
        &FrameStack<Self::Frame>,
        &Self::Inspector,
    ) {
        let (ctx, inst, pre, fs) = self.inner.all();
        (ctx, inst, pre, fs, &self.inner.inspector)
    }

    fn all_mut_inspector(
        &mut self,
    ) -> (
        &mut Self::Context,
        &mut Self::Instructions,
        &mut Self::Precompiles,
        &mut FrameStack<Self::Frame>,
        &mut Self::Inspector,
    ) {
        (
            &mut self.inner.ctx,
            &mut self.inner.instruction,
            &mut self.inner.precompiles,
            &mut self.inner.frame_stack,
            &mut self.inner.inspector,
        )
    }
}

// ── InspectEvm for ArbEvm ────────────────────────────────────────

impl<DB, I> InspectEvm for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
    revm::Journal<DB>: revm::inspector::JournalExt,
{
    type Inspector = I;

    fn set_inspector(&mut self, inspector: Self::Inspector) {
        self.inner.inspector = inspector;
    }

    fn inspect_one_tx(&mut self, tx: Self::Tx) -> Result<Self::ExecutionResult, Self::Error> {
        self.inner.ctx.set_tx(tx);
        MainnetHandler::default().inspect_run(self)
    }
}

// ── alloy_evm::Evm for ArbEvm ───────────────────────────────────

impl<DB, I> Evm for ArbEvm<DB, I>
where
    DB: Database,
    I: Inspector<EthEvmContext<DB>, EthInterpreter>,
    revm::Journal<DB>: revm::inspector::JournalExt,
{
    type DB = DB;
    type Tx = ArbTransaction;
    type Error = EVMError<<DB as revm::Database>::Error>;
    type HaltReason = HaltReason;
    type Spec = SpecId;
    type Precompiles = PrecompilesMap;
    type Inspector = I;
    type BlockEnv = revm::context::BlockEnv;

    fn block(&self) -> &revm::context::BlockEnv {
        &self.inner.ctx.block
    }

    fn chain_id(&self) -> u64 {
        self.inner.ctx.cfg.chain_id
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        if self.inspect {
            InspectEvm::inspect_tx(self, tx.into_inner())
        } else {
            ExecuteEvm::transact(self, tx.into_inner())
        }
    }

    fn transact_system_call(
        &mut self,
        caller: Address,
        contract: Address,
        data: Bytes,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        SystemCallEvm::system_call_with_caller(self, caller, contract, data)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec>) {
        let revm::Context { block: block_env, cfg: cfg_env, journaled_state, .. } = self.inner.ctx;
        (journaled_state.database, EvmEnv { block_env, cfg_env })
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inspect = enabled;
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        (
            &self.inner.ctx.journaled_state.database,
            &self.inner.inspector,
            &self.inner.precompiles.0,
        )
    }

    fn components_mut(&mut self) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        (
            &mut self.inner.ctx.journaled_state.database,
            &mut self.inner.inspector,
            &mut self.inner.precompiles.0,
        )
    }
}

// ── ArbEvmFactory ──────────────────────────────────────────────────

/// Factory for creating Arbitrum EVM instances with custom precompiles.
#[derive(Default, Debug, Clone, Copy)]
pub struct ArbEvmFactory(pub alloy_evm::EthEvmFactory);

impl ArbEvmFactory {
    pub fn new() -> Self {
        Self::default()
    }
}

fn build_arb_evm<DB: Database, I>(
    inner: RevmEvm<
        EthEvmContext<DB>,
        I,
        EthInstructions<EthInterpreter, EthEvmContext<DB>>,
        PrecompilesMap,
        EthFrame,
    >,
    inspect: bool,
) -> ArbEvm<DB, I> {
    let RevmEvm {
        ctx,
        inspector,
        mut instruction,
        mut precompiles,
        frame_stack: _,
    } = inner;

    instruction.insert_instruction(
        BLOBBASEFEE_OPCODE,
        revm::interpreter::Instruction::new(arb_blob_basefee, 2),
    );
    instruction.insert_instruction(
        SELFDESTRUCT_OPCODE,
        revm::interpreter::Instruction::new(arb_selfdestruct, 5000),
    );
    instruction.insert_instruction(
        NUMBER_OPCODE,
        revm::interpreter::Instruction::new(arb_number, 2),
    );
    instruction.insert_instruction(
        BLOCKHASH_OPCODE,
        revm::interpreter::Instruction::new(arb_blockhash, 20),
    );
    instruction.insert_instruction(
        BALANCE_OPCODE,
        revm::interpreter::Instruction::new(arb_balance, 0),
    );
    instruction.insert_instruction(
        SELFBALANCE_OPCODE,
        revm::interpreter::Instruction::new(arb_selfbalance, 5),
    );
    register_arb_precompiles(&mut precompiles, arb_precompiles::get_arbos_version());
    let arb_precompiles = ArbPrecompilesMap(precompiles);

    let revm_evm = RevmEvm::new_with_inspector(ctx, inspector, instruction, arb_precompiles);
    ArbEvm::new(revm_evm, inspect)
}

impl EvmFactory for ArbEvmFactory {
    type Evm<DB: Database, I: Inspector<EthEvmContext<DB>, EthInterpreter>> = ArbEvm<DB, I>;
    type Context<DB: Database> = EthEvmContext<DB>;
    type Tx = ArbTransaction;
    type Error<DBError: core::error::Error + Send + Sync + 'static> = EVMError<DBError>;
    type HaltReason = HaltReason;
    type Spec = SpecId;
    type Precompiles = PrecompilesMap;
    type BlockEnv = revm::context::BlockEnv;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let eth_evm = self.0.create_evm(db, input);
        build_arb_evm(eth_evm.into_inner(), false)
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>, EthInterpreter>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let eth_evm = self.0.create_evm_with_inspector(db, input, inspector);
        build_arb_evm(eth_evm.into_inner(), true)
    }
}
