use alloy_evm::{
    eth::EthEvmContext, precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory,
};
use alloy_primitives::{Address, Bytes};
use arb_precompiles::register_arb_precompiles;
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
    CallInputs, Host, InstructionContext, InstructionResult, InterpreterResult, InterpreterTypes,
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
    let Some(target) = ctx.interpreter.stack.pop_address() else {
        ctx.interpreter.halt(InstructionResult::StackUnderflow);
        return;
    };

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
        // The journal depth is incremented by checkpoint() at the start of each call
        // frame, matching Go's evm.Depth() semantics exactly.
        arb_precompiles::set_evm_depth(context.journaled_state.inner.depth);
        <PrecompilesMap as PrecompileProvider<
            revm::Context<BlockEnv, TxEnv, CfgEnv, DB, revm::Journal<DB>, Chain>,
        >>::run(&mut self.0, context, inputs)
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
///
/// Internally stores `EthEvm<DB, I, ArbPrecompilesMap>` for depth tracking,
/// but exposes `Precompiles = PrecompilesMap` to satisfy reth's
/// `ConfigureEvm` constraint.
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

/// Helper: customize instruction table and register Arb precompiles on the
/// inner revm EVM, then wrap in `EthEvm<DB, I, ArbPrecompilesMap>`.
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
    // Destructure to access and wrap precompiles with a different type.
    let revm::context::Evm {
        ctx,
        inspector,
        mut instruction,
        mut precompiles,
        frame_stack: _,
    } = inner;

    // BLOBBASEFEE is not supported on Arbitrum — override to halt.
    instruction.insert_instruction(
        BLOBBASEFEE_OPCODE,
        revm::interpreter::Instruction::new(arb_blob_basefee, 2),
    );
    // SELFDESTRUCT: revert if the acting account is a Stylus program.
    instruction.insert_instruction(
        SELFDESTRUCT_OPCODE,
        revm::interpreter::Instruction::new(arb_selfdestruct, 5000),
    );

    // Register Arbitrum precompiles, then wrap in depth-tracking provider.
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
