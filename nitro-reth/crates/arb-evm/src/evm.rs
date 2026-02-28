use alloy_evm::{
    eth::EthEvmContext, precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory,
};
use alloy_primitives::{Address, Bytes, B256, U256};
use arbos::programs::types::EvmData;
use arb_precompiles::register_arb_precompiles;
use arb_stylus::config::StylusConfig;
use arb_stylus::ink::Gas as StylusGas;
use arb_stylus::meter::MeteredMachine;
use arb_stylus::run::RunProgram;
use arb_stylus::StylusEvmApi;
use core::fmt::Debug;
use revm::context::result::EVMError;
use revm::context_interface::host::LoadError;
use revm::context_interface::result::{HaltReason, ResultAndState};
use revm::handler::instructions::EthInstructions;
use revm::handler::{EthFrame, PrecompileProvider};
use revm::inspector::NoOpInspector;
use revm::interpreter::interpreter::EthInterpreter;
use revm::interpreter::interpreter_types::{InputsTr, RuntimeFlag, StackTr};
use revm::interpreter::{
    CallInput, CallInputs, CallScheme, Gas as EvmGas, Host, InstructionContext, InstructionResult,
    InterpreterResult, InterpreterTypes,
};
use revm::primitives::hardfork::SpecId;

use crate::transaction::ArbTransaction;

/// BLOBBASEFEE opcode (0x4a).
const BLOBBASEFEE_OPCODE: u8 = 0x4a;

/// SELFDESTRUCT opcode (0xff).
const SELFDESTRUCT_OPCODE: u8 = 0xff;

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

use std::cell::Cell;
use std::cell::RefCell;
use std::collections::HashMap;

thread_local! {
    /// Currently open WASM memory pages across all active Stylus calls.
    static STYLUS_PAGES_OPEN: Cell<u16> = const { Cell::new(0) };
    /// High-water mark of pages ever open during this transaction.
    static STYLUS_PAGES_EVER: Cell<u16> = const { Cell::new(0) };
    /// Per-address count of open Stylus execution contexts (for reentrancy).
    static STYLUS_PROGRAM_COUNTS: RefCell<HashMap<Address, u32>> = RefCell::new(HashMap::new());
}

/// Reset Stylus page counters at transaction start.
pub fn reset_stylus_pages() {
    STYLUS_PAGES_OPEN.with(|v| v.set(0));
    STYLUS_PAGES_EVER.with(|v| v.set(0));
    STYLUS_PROGRAM_COUNTS.with(|v| v.borrow_mut().clear());
}

/// Get current (open, ever) page counts.
pub fn get_stylus_pages() -> (u16, u16) {
    let open = STYLUS_PAGES_OPEN.with(|v| v.get());
    let ever = STYLUS_PAGES_EVER.with(|v| v.get());
    (open, ever)
}

/// Add pages for a new Stylus call. Returns previous (open, ever).
fn add_stylus_pages(footprint: u16) -> (u16, u16) {
    let open = STYLUS_PAGES_OPEN.with(|v| v.get());
    let ever = STYLUS_PAGES_EVER.with(|v| v.get());
    let new_open = open.saturating_add(footprint);
    STYLUS_PAGES_OPEN.with(|v| v.set(new_open));
    STYLUS_PAGES_EVER.with(|v| v.set(ever.max(new_open)));
    (open, ever)
}

/// Restore page count after Stylus call returns.
fn set_stylus_pages_open(open: u16) {
    STYLUS_PAGES_OPEN.with(|v| v.set(open));
}

/// Push a Stylus program address onto the reentrancy tracker.
/// Returns true if this is a reentrant call (address was already active).
fn push_stylus_program(addr: Address) -> bool {
    STYLUS_PROGRAM_COUNTS.with(|v| {
        let mut map = v.borrow_mut();
        let count = map.entry(addr).or_insert(0);
        *count += 1;
        *count > 1
    })
}

/// Pop a Stylus program address from the reentrancy tracker.
fn pop_stylus_program(addr: Address) {
    STYLUS_PROGRAM_COUNTS.with(|v| {
        let mut map = v.borrow_mut();
        if let Some(count) = map.get_mut(&addr) {
            *count = count.saturating_sub(1);
            if *count == 0 {
                map.remove(&addr);
            }
        }
    });
}

// ── Stylus WASM dispatch ────────────────────────────────────────────

/// Execute a Stylus WASM program by creating a NativeInstance and running it.
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

    // Strip the 4-byte Stylus prefix to get the serialized module.
    let (module_bytes, _version_byte) = match arb_stylus::strip_stylus_prefix(bytecode) {
        Ok(v) => v,
        Err(_) => {
            return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
        }
    };

    // Track reentrancy.
    let target_addr = inputs.target_address;
    let is_delegate = matches!(inputs.scheme, CallScheme::DelegateCall | CallScheme::CallCode);
    let reentrant = if !is_delegate {
        push_stylus_program(target_addr)
    } else {
        false
    };

    // Build EvmData from the execution context.
    let mut evm_data = build_evm_data(context, inputs);
    evm_data.reentrant = reentrant as u32;

    // Default Stylus config (params come from ArbOS state in Phase 5).
    let ink_price = 10_000u32;
    let stylus_config = StylusConfig::new(1, 4 * 65536, ink_price);

    // Track pages — add this program's footprint.
    // TODO: get actual footprint from program metadata in Phase 5.
    let footprint: u16 = 0; // Placeholder until we look up program metadata.
    let (prev_open, _prev_ever) = add_stylus_pages(footprint);

    // Create the type-erased StylusEvmApi bridge.
    let journal_ptr = &mut context.journaled_state as *mut revm::Journal<DB>;
    let is_static = inputs.is_static || matches!(inputs.scheme, CallScheme::StaticCall);
    let evm_api = unsafe { StylusEvmApi::new(journal_ptr, target_addr, is_static) };

    let code_hash = alloy_primitives::keccak256(bytecode);

    // Build the NativeInstance from the module bytes.
    let mut instance = match arb_stylus::NativeInstance::from_bytes(
        module_bytes,
        evm_api,
        evm_data,
        &arb_stylus::CompileConfig::version(1, false),
        stylus_config,
    ) {
        Ok(inst) => inst,
        Err(e) => {
            tracing::warn!(target: "stylus", codehash = %code_hash, err = %e, "failed to create WASM instance");
            set_stylus_pages_open(prev_open);
            if !is_delegate {
                pop_stylus_program(target_addr);
            }
            return InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas());
        }
    };

    // Convert EVM gas to ink.
    let gas = inputs.gas_limit;
    let ink = stylus_config.pricing.gas_to_ink(StylusGas(gas));

    // Get calldata from CallInput enum.
    let calldata: &[u8] = match &inputs.input {
        CallInput::Bytes(bytes) => bytes,
        CallInput::SharedBuffer(_) => &[],
    };

    // Execute the WASM program.
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
    let gas_result = EvmGas::new(gas_left);

    // Map UserOutcome to InterpreterResult.
    let output: Bytes = instance.env().outs.clone().into();
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
            InterpreterResult::new(InstructionResult::Revert, Bytes::new(), zero_gas())
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

    EvmData {
        arbos_version: arb_precompiles::get_arbos_version(),
        block_basefee: B256::from(basefee.to_be_bytes()),
        chain_id: context.cfg.chain_id(),
        block_coinbase: context.block.beneficiary(),
        block_gas_limit: context.block.gas_limit(),
        block_number: context.block.number().saturating_to(),
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

// ── Depth-tracking precompile provider ─────────────────────────────

/// Wraps [`PrecompilesMap`] to set the thread-local EVM call depth before
/// each precompile invocation. The depth is read from revm's journal, which
/// mirrors Go's `evm.Depth()` counter used by `ArbSys.isTopLevelCall`.
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

        // Load the target's bytecode to check for Stylus discriminant.
        let bytecode = if let Some((_hash, ref code)) = inputs.known_bytecode {
            code.original_bytes()
        } else {
            let account = context
                .journaled_state
                .inner
                .load_code(&mut context.journaled_state.database, inputs.bytecode_address)
                .map_err(|_| "failed to load bytecode for Stylus check".to_string())?;
            account
                .data
                .info
                .code
                .as_ref()
                .map(|c| c.original_bytes())
                .unwrap_or_default()
        };

        if arb_stylus::is_stylus_program(&bytecode) {
            let result = execute_stylus_program(context, inputs, &bytecode);
            return Ok(Some(result));
        }

        // Not a precompile or Stylus program — fall through to normal EVM.
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

/// Arbitrum EVM wrapper with depth-tracking precompiles and custom opcodes.
pub struct ArbEvm<DB: Database, I> {
    inner: alloy_evm::EthEvm<DB, I, ArbPrecompilesMap>,
}

impl<DB, I> ArbEvm<DB, I>
where
    DB: Database,
{
    pub fn new(inner: alloy_evm::EthEvm<DB, I, ArbPrecompilesMap>) -> Self {
        Self { inner }
    }

    pub fn into_inner(self) -> alloy_evm::EthEvm<DB, I, ArbPrecompilesMap> {
        self.inner
    }
}

impl<DB, I> Evm for ArbEvm<DB, I>
where
    DB: Database,
    I: revm::inspector::Inspector<EthEvmContext<DB>>,
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
        self.inner.block()
    }

    fn chain_id(&self) -> u64 {
        self.inner.chain_id()
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.transact_raw(tx.into_inner())
    }

    fn transact_system_call(
        &mut self,
        caller: Address,
        contract: Address,
        data: Bytes,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.transact_system_call(caller, contract, data)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec>) {
        self.inner.finish()
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inner.set_inspector_enabled(enabled)
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        let (db, inspector, arb_precompiles) = self.inner.components();
        (db, inspector, &arb_precompiles.0)
    }

    fn components_mut(
        &mut self,
    ) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        let (db, inspector, arb_precompiles) = self.inner.components_mut();
        (db, inspector, &mut arb_precompiles.0)
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
    inner: revm::context::Evm<
        EthEvmContext<DB>,
        I,
        EthInstructions<EthInterpreter, EthEvmContext<DB>>,
        PrecompilesMap,
        EthFrame,
    >,
    inspect: bool,
) -> ArbEvm<DB, I> {
    let revm::context::Evm {
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

    register_arb_precompiles(&mut precompiles);
    let arb_precompiles = ArbPrecompilesMap(precompiles);

    let revm_evm =
        revm::context::Evm::new_with_inspector(ctx, inspector, instruction, arb_precompiles);
    let eth_evm = alloy_evm::eth::EthEvm::new(revm_evm, inspect);
    ArbEvm::new(eth_evm)
}

impl EvmFactory for ArbEvmFactory {
    type Evm<DB: Database, I: revm::inspector::Inspector<EthEvmContext<DB>>> = ArbEvm<DB, I>;
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

    fn create_evm_with_inspector<DB: Database, I: revm::inspector::Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let eth_evm = self.0.create_evm_with_inspector(db, input, inspector);
        build_arb_evm(eth_evm.into_inner(), true)
    }
}
