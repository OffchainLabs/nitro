use alloy_evm::{
    eth::EthEvmContext, precompiles::PrecompilesMap, Database, Evm, EvmEnv, EvmFactory,
};
use alloy_primitives::{Address, Bytes, U256};
use arb_precompiles::register_arb_precompiles;
use core::fmt::Debug;
use revm::context::result::EVMError;
use revm::context_interface::host::LoadError;
use revm::context_interface::result::{HaltReason, ResultAndState};
use revm::inspector::NoOpInspector;
use revm::interpreter::interpreter_types::{InputsTr, RuntimeFlag, StackTr};
use revm::interpreter::{Host, InstructionContext, InstructionResult, InterpreterTypes};
use revm::primitives::hardfork::SpecId;

use crate::transaction::ArbTransaction;

/// BLOCKHASH opcode (0x40).
const BLOCKHASH_OPCODE: u8 = 0x40;

/// BLOBBASEFEE opcode (0x4a).
const BLOBBASEFEE_OPCODE: u8 = 0x4a;

/// SELFDESTRUCT opcode (0xff).
const SELFDESTRUCT_OPCODE: u8 = 0xff;

/// Arbitrum BLOCKHASH: uses L1 block hashes with an adjusted range.
///
/// Go's opBlockhash calls `ProcessingHook.L1BlockNumber()` which returns
/// `stored_l1_block_number + 1` (the next expected L1 block). This makes
/// the current L1 block's hash available via BLOCKHASH, unlike standard
/// Ethereum where `BLOCKHASH(NUMBER)` returns zero.
///
/// block_env.number = current L1 block number, so upper = number + 1
/// to match Go's range of [upper - 256, upper).
fn arb_blockhash<WIRE: InterpreterTypes, H: Host + ?Sized>(
    ctx: InstructionContext<'_, H, WIRE>,
) {
    let Some(([], number)) = ctx.interpreter.stack.popn_top::<0>() else {
        ctx.interpreter.halt(InstructionResult::StackUnderflow);
        return;
    };

    let requested = number.to::<u64>();
    let current = ctx.host.block_number().to::<u64>();
    // Go uses L1BlockNumber() = current + 1 as upper bound.
    let upper = current + 1;
    let lower = upper.saturating_sub(256);

    if requested >= lower && requested < upper {
        match ctx.host.block_hash(requested) {
            Some(hash) => *number = U256::from_be_bytes(hash.0),
            None => {
                ctx.interpreter.halt_fatal();
                return;
            }
        }
    } else {
        *number = U256::ZERO;
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

/// Arbitrum EVM wrapper that registers custom precompiles.
pub struct ArbEvm<DB: Database + Debug, I> {
    inner: alloy_evm::EthEvm<DB, I, PrecompilesMap>,
}

impl<DB, I> ArbEvm<DB, I>
where
    DB: Database + Debug,
{
    pub fn new(inner: alloy_evm::EthEvm<DB, I, PrecompilesMap>) -> Self {
        Self { inner }
    }

    pub fn into_inner(self) -> alloy_evm::EthEvm<DB, I, PrecompilesMap> {
        self.inner
    }
}

impl<DB, I> Evm for ArbEvm<DB, I>
where
    DB: Database + Debug,
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
        self.inner.components()
    }

    fn components_mut(
        &mut self,
    ) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        self.inner.components_mut()
    }
}

/// Factory for creating Arbitrum EVM instances with custom precompiles.
#[derive(Default, Debug, Clone, Copy)]
pub struct ArbEvmFactory(pub alloy_evm::EthEvmFactory);

impl ArbEvmFactory {
    pub fn new() -> Self {
        Self::default()
    }
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
        let mut inner = eth_evm.into_inner();
        // BLOCKHASH: use L1 block hashes with adjusted range (upper = number + 1).
        inner.instruction.insert_instruction(
            BLOCKHASH_OPCODE,
            revm::interpreter::Instruction::new(arb_blockhash, 20),
        );
        // BLOBBASEFEE is not supported on Arbitrum — override to halt.
        inner.instruction.insert_instruction(
            BLOBBASEFEE_OPCODE,
            revm::interpreter::Instruction::new(arb_blob_basefee, 2),
        );
        // SELFDESTRUCT: revert if the acting account is a Stylus program.
        inner.instruction.insert_instruction(
            SELFDESTRUCT_OPCODE,
            revm::interpreter::Instruction::new(arb_selfdestruct, 5000),
        );
        let eth_evm = alloy_evm::eth::EthEvm::new(inner, false);
        let mut evm = ArbEvm::new(eth_evm);
        let (_, _, precompiles) = evm.components_mut();
        register_arb_precompiles(precompiles);
        evm
    }

    fn create_evm_with_inspector<DB: Database, I: revm::inspector::Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<Self::Spec>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let eth_evm = self.0.create_evm_with_inspector(db, input, inspector);
        let mut inner = eth_evm.into_inner();
        // BLOCKHASH: use L1 block hashes with adjusted range (upper = number + 1).
        inner.instruction.insert_instruction(
            BLOCKHASH_OPCODE,
            revm::interpreter::Instruction::new(arb_blockhash, 20),
        );
        // BLOBBASEFEE is not supported on Arbitrum — override to halt.
        inner.instruction.insert_instruction(
            BLOBBASEFEE_OPCODE,
            revm::interpreter::Instruction::new(arb_blob_basefee, 2),
        );
        // SELFDESTRUCT: revert if the acting account is a Stylus program.
        inner.instruction.insert_instruction(
            SELFDESTRUCT_OPCODE,
            revm::interpreter::Instruction::new(arb_selfdestruct, 5000),
        );
        let eth_evm = alloy_evm::eth::EthEvm::new(inner, true);
        let mut evm = ArbEvm::new(eth_evm);
        let (_, _, precompiles) = evm.components_mut();
        register_arb_precompiles(precompiles);
        evm
    }
}
