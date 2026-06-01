//! revm inspector that accumulates per-transaction multi-gas by dimension.
//!
//! Each opcode's observed gas delta (`step` to `step_end`) is split with
//! [`classify`]. Frame-spawning opcodes are special: their delta includes the
//! gas forwarded to the child frame, whose own opcodes are observed separately,
//! so the forwarded portion is removed via the `call`/`create` hooks. Contract
//! code-deposit gas is charged at frame return rather than at an opcode, so it
//! is added in `create_end`.

use alloy_evm::Database;
use alloy_primitives::{Address, B256, U256};
use arb_primitives::multigas::MultiGas;
use parking_lot::Mutex;
use revm::{
    bytecode::opcode,
    context::{JournalEntry, JournalInner},
    interpreter::{
        interpreter::EthInterpreter,
        interpreter_types::{InputsTr, Jumps},
        CallInputs, CallOutcome, CallValue, CreateInputs, CreateOutcome, Interpreter,
    },
    Context, Inspector, Journal,
};

/// EVM context the inspector observes, generic over the block/tx/cfg/chain
/// environments so it works on both the top-level executor and the Stylus
/// sub-call interpreter; only the journal (fixed to `Journal<DB>`) is read.
type Ctx<B, T, C, DB, Ch> = Context<B, T, C, DB, Journal<DB>, Ch>;
use std::sync::Arc;

use crate::multi_gas::classify::{classify, OpKind};

/// Shared slot a [`MultiGasInspector`] writes each transaction's multi-gas to,
/// read by the block executor after execution.
pub type MultiGasSink = Arc<Mutex<Option<MultiGas>>>;

const WARM: u64 = 100; // WarmStorageReadCostEIP2929
const CREATE_DATA_GAS: u64 = 200; // CreateDataGas (code storage, per byte)

/// Accumulates per-transaction multi-gas across every executed frame.
#[derive(Debug, Default)]
pub struct MultiGasInspector {
    prev_gas: u64,
    pending: Pending,
    accumulated: MultiGas,
    sink: Option<MultiGasSink>,
    /// Count of frames opened (call/create) and not yet closed. The outermost
    /// frame closes when this returns to zero; the journal depth cannot be used
    /// for this because a create frame can leave it offset.
    open_frames: u32,
}

#[derive(Debug, Default)]
enum Pending {
    #[default]
    None,
    /// SLOAD (`account = false`) or BALANCE/EXTCODESIZE/EXTCODEHASH
    /// (`account = true`): the cold surcharge is the whole dynamic cost, so
    /// cold is read off the step delta.
    DeltaCold {
        account: bool,
    },
    Log {
        topics: u8,
        data_len: u64,
    },
    ExtCodeCopy {
        cold: bool,
        words: u64,
    },
    SelfDestruct {
        cold: bool,
        new_account: bool,
    },
    /// SSTORE needs the committed value, only reliable after the write; the new
    /// value, warmth, and pre-write current value are captured at step time.
    SStore {
        cold: bool,
        contract: Address,
        key: U256,
        new: U256,
        current: Option<U256>,
    },
    /// CALL/CREATE family. `delta` is filled at `step_end`; the own cost is
    /// resolved in the matching `call`/`create` hook.
    Frame {
        cold: bool,
        is_create: bool,
        is_plain_call: bool,
        delta: u64,
    },
    Other,
}

impl MultiGasInspector {
    /// Creates an inspector that publishes each transaction's multi-gas to a
    /// shared sink when the top-level frame returns.
    pub fn with_sink(sink: MultiGasSink) -> Self {
        Self {
            sink: Some(sink),
            ..Default::default()
        }
    }

    /// Returns the accumulated multi-gas and resets for the next transaction.
    pub fn take_multi_gas(&mut self) -> MultiGas {
        self.flush_dangling_frame();
        self.pending = Pending::None;
        self.prev_gas = 0;
        self.open_frames = 0;
        core::mem::replace(&mut self.accumulated, MultiGas::zero())
    }

    /// Publishes the accumulated multi-gas to the sink and resets, called when
    /// the outermost frame returns.
    fn publish(&mut self) {
        if self.sink.is_some() {
            let gas = self.take_multi_gas();
            if let Some(sink) = &self.sink {
                *sink.lock() = Some(gas);
            }
        }
    }

    fn add(&mut self, gas: MultiGas) {
        self.accumulated = self.accumulated.saturating_add(gas);
    }

    /// Records a frame closing. When the outermost frame closes (no frames left
    /// open) the transaction's multi-gas is complete and is published.
    fn frame_closed(&mut self) {
        self.open_frames = self.open_frames.saturating_sub(1);
        if self.open_frames == 0 {
            self.publish();
        }
    }

    /// A frame opcode that halted before forwarding (e.g. out of gas) never
    /// reaches its `call`/`create` hook; classify it from the full delta.
    fn flush_dangling_frame(&mut self) {
        if let Pending::Frame {
            cold,
            is_create,
            delta,
            ..
        } = self.pending
        {
            let gas = if is_create {
                MultiGas::computation_gas(delta)
            } else {
                classify(
                    OpKind::Call {
                        cold,
                        new_account: false,
                    },
                    delta,
                )
            };
            self.add(gas);
            self.pending = Pending::None;
        }
    }
}

impl<B, T, C, DB: Database, Ch> Inspector<Ctx<B, T, C, DB, Ch>, EthInterpreter>
    for MultiGasInspector
{
    fn step(&mut self, interp: &mut Interpreter<EthInterpreter>, ctx: &mut Ctx<B, T, C, DB, Ch>) {
        self.flush_dangling_frame();
        self.prev_gas = interp.gas.remaining();
        let op = interp.bytecode.opcode();
        let journal = &ctx.journaled_state.inner;
        self.pending = match op {
            opcode::SLOAD => Pending::DeltaCold { account: false },
            opcode::BALANCE | opcode::EXTCODESIZE | opcode::EXTCODEHASH => {
                Pending::DeltaCold { account: true }
            }
            opcode::EXTCODECOPY => Pending::ExtCodeCopy {
                cold: address_cold(journal, addr_arg(interp, 0)),
                words: word_count(peek(interp, 3)),
            },
            opcode::LOG0..=opcode::LOG4 => Pending::Log {
                topics: op - opcode::LOG0,
                data_len: to_u64(peek(interp, 1)),
            },
            opcode::SSTORE => {
                let contract = interp.input.target_address();
                let key = peek(interp, 0);
                Pending::SStore {
                    cold: slot_cold(journal, contract, key),
                    contract,
                    key,
                    new: peek(interp, 1),
                    current: slot_values(journal, contract, key).map(|(_, present)| present),
                }
            }
            opcode::SELFDESTRUCT => {
                let beneficiary = addr_arg(interp, 0);
                let contract = interp.input.target_address();
                Pending::SelfDestruct {
                    cold: address_cold(journal, beneficiary),
                    new_account: account_empty(journal, beneficiary)
                        && !account_balance_zero(journal, contract),
                }
            }
            opcode::CALL | opcode::CALLCODE => Pending::Frame {
                cold: address_cold(journal, addr_arg(interp, 1)),
                is_create: false,
                is_plain_call: op == opcode::CALL,
                delta: 0,
            },
            opcode::DELEGATECALL | opcode::STATICCALL => Pending::Frame {
                cold: address_cold(journal, addr_arg(interp, 1)),
                is_create: false,
                is_plain_call: false,
                delta: 0,
            },
            opcode::CREATE | opcode::CREATE2 => Pending::Frame {
                cold: false,
                is_create: true,
                is_plain_call: false,
                delta: 0,
            },
            _ => Pending::Other,
        };
    }

    fn step_end(
        &mut self,
        interp: &mut Interpreter<EthInterpreter>,
        ctx: &mut Ctx<B, T, C, DB, Ch>,
    ) {
        let delta = self.prev_gas.saturating_sub(interp.gas.remaining());
        let pending = core::mem::replace(&mut self.pending, Pending::None);
        let gas = match pending {
            Pending::Frame {
                cold,
                is_create,
                is_plain_call,
                ..
            } => {
                self.pending = Pending::Frame {
                    cold,
                    is_create,
                    is_plain_call,
                    delta,
                };
                return;
            }
            Pending::None => return,
            Pending::DeltaCold { account } => {
                let cold = delta > WARM;
                let kind = if account {
                    OpKind::AccountAccess { cold }
                } else {
                    OpKind::StorageRead { cold }
                };
                classify(kind, delta)
            }
            Pending::Log { topics, data_len } => classify(OpKind::Log { topics, data_len }, delta),
            Pending::ExtCodeCopy { cold, words } => {
                classify(OpKind::ExtCodeCopy { cold, words }, delta)
            }
            Pending::SelfDestruct { cold, new_account } => {
                classify(OpKind::SelfDestruct { cold, new_account }, delta)
            }
            Pending::SStore {
                cold,
                contract,
                key,
                new,
                current,
            } => {
                let original = slot_values(&ctx.journaled_state.inner, contract, key)
                    .map(|(original, _)| original)
                    .unwrap_or(U256::ZERO);
                let present = current.unwrap_or(original);
                classify(
                    OpKind::StorageWrite {
                        cold,
                        original,
                        present,
                        new,
                    },
                    delta,
                )
            }
            Pending::Other => classify(OpKind::Other, delta),
        };
        self.add(gas);
    }

    fn call(
        &mut self,
        ctx: &mut Ctx<B, T, C, DB, Ch>,
        inputs: &mut CallInputs,
    ) -> Option<CallOutcome> {
        self.open_frames += 1;
        if let Pending::Frame {
            cold,
            is_create: false,
            is_plain_call,
            delta,
        } = self.pending
        {
            self.pending = Pending::None;
            let value_transfer = matches!(inputs.value, CallValue::Transfer(v) if !v.is_zero());
            let own = call_own_cost(delta, inputs.gas_limit);
            let new_account = is_plain_call
                && value_transfer
                && account_empty(&ctx.journaled_state.inner, inputs.target_address);
            self.add(classify(OpKind::Call { cold, new_account }, own));
        }
        None
    }

    fn call_end(
        &mut self,
        _ctx: &mut Ctx<B, T, C, DB, Ch>,
        _inputs: &CallInputs,
        _outcome: &mut CallOutcome,
    ) {
        self.frame_closed();
    }

    fn create(
        &mut self,
        _ctx: &mut Ctx<B, T, C, DB, Ch>,
        inputs: &mut CreateInputs,
    ) -> Option<CreateOutcome> {
        self.open_frames += 1;
        if let Pending::Frame {
            is_create: true,
            delta,
            ..
        } = self.pending
        {
            self.pending = Pending::None;
            let own = delta.saturating_sub(inputs.gas_limit());
            self.add(MultiGas::computation_gas(own));
        }
        None
    }

    fn create_end(
        &mut self,
        _ctx: &mut Ctx<B, T, C, DB, Ch>,
        _inputs: &CreateInputs,
        outcome: &mut CreateOutcome,
    ) {
        if outcome.result.is_ok() {
            let deposit = (outcome.result.output.len() as u64).saturating_mul(CREATE_DATA_GAS);
            self.add(MultiGas::storage_growth_gas(deposit));
        }
        self.frame_closed();
    }
}

/// Own cost of a call opcode: the step delta minus the child's full gas limit.
/// The child limit includes any value-transfer stipend; the stipend and any
/// unused forwarded gas are returned to the caller when the child frame ends,
/// so they belong to the child's accounting, not the caller's own cost.
fn call_own_cost(delta: u64, child_gas_limit: u64) -> u64 {
    delta.saturating_sub(child_gas_limit)
}

fn peek(interp: &Interpreter<EthInterpreter>, from_top: usize) -> U256 {
    let data = interp.stack.data();
    data.len()
        .checked_sub(from_top + 1)
        .map(|i| data[i])
        .unwrap_or(U256::ZERO)
}

fn addr_arg(interp: &Interpreter<EthInterpreter>, from_top: usize) -> Address {
    Address::from_word(B256::from(peek(interp, from_top).to_be_bytes::<32>()))
}

fn word_count(len: U256) -> u64 {
    to_u64(len).div_ceil(32)
}

fn to_u64(v: U256) -> u64 {
    u64::try_from(v).unwrap_or(u64::MAX)
}

fn address_cold(journal: &JournalInner<JournalEntry>, addr: Address) -> bool {
    if journal.warm_addresses.is_warm(&addr) {
        return false;
    }
    match journal.state.get(&addr) {
        Some(account) => account.is_cold_transaction_id(journal.transaction_id),
        None => true,
    }
}

fn slot_cold(journal: &JournalInner<JournalEntry>, addr: Address, key: U256) -> bool {
    match journal.state.get(&addr).and_then(|a| a.storage.get(&key)) {
        Some(slot) => slot.is_cold_transaction_id(journal.transaction_id),
        None => true,
    }
}

fn slot_values(
    journal: &JournalInner<JournalEntry>,
    addr: Address,
    key: U256,
) -> Option<(U256, U256)> {
    let slot = journal.state.get(&addr)?.storage.get(&key)?;
    Some((slot.original_value, slot.present_value))
}

fn account_empty(journal: &JournalInner<JournalEntry>, addr: Address) -> bool {
    match journal.state.get(&addr) {
        Some(account) => {
            account.info.balance.is_zero()
                && account.info.nonce == 0
                && account.info.code_hash == revm::primitives::KECCAK_EMPTY
        }
        None => true,
    }
}

fn account_balance_zero(journal: &JournalInner<JournalEntry>, addr: Address) -> bool {
    match journal.state.get(&addr) {
        Some(account) => account.info.balance.is_zero(),
        None => true,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn call_own_cost_strips_forwarded_gas() {
        // delta = own (2600 cold + 9000 value + 100 warm) + forwarded 50000.
        let delta = 2_600 + 9_000 + 100 + 50_000;
        assert_eq!(call_own_cost(delta, 50_000), 2_600 + 9_000 + 100);
    }

    #[test]
    fn call_own_cost_strips_value_transfer_stipend() {
        // A value transfer to an EOA forwards no gas; the child limit is just the
        // 2300 stipend, which the EOA does not spend and revm returns to the
        // caller. So the caller's own cost is the call cost minus the stipend.
        const CALL_STIPEND: u64 = 2_300;
        let call_cost = 2_600 + 9_000; // cold access + value transfer
        let delta = call_cost; // stipend is not deducted from the caller
        assert_eq!(call_own_cost(delta, CALL_STIPEND), call_cost - CALL_STIPEND);
    }

    #[test]
    fn call_own_cost_saturates() {
        assert_eq!(call_own_cost(100, 50_000), 0);
    }

    #[test]
    fn word_count_rounds_up() {
        assert_eq!(word_count(U256::ZERO), 0);
        assert_eq!(word_count(U256::from(1)), 1);
        assert_eq!(word_count(U256::from(32)), 1);
        assert_eq!(word_count(U256::from(33)), 2);
    }
}
