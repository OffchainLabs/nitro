// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{FuncMiddleware, Middleware, ModuleMod};
use crate::Machine;

use arbutil::operator::{operator_at_end_of_basic_block, operator_factor, OperatorCode};
use eyre::Result;
use fnv::FnvHashMap as HashMap;
use parking_lot::Mutex;
use std::collections::BTreeMap;
use std::{clone::Clone, fmt::Debug, sync::Arc};
use wasmer::{wasmparser::Operator, GlobalInit, Type};
use wasmer_types::{GlobalIndex, LocalFunctionIndex};

#[derive(Debug)]
pub struct Counter {
    pub index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
    pub opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
}

pub fn opcode_count_name(index: &usize) -> String {
    format!("stylus_opcode{}_count", index)
}

impl Counter {
    pub fn new(opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>) -> Self {
        Self {
            index_counts_global: Arc::new(Mutex::new(Vec::with_capacity(
                OperatorCode::OPERATOR_COUNT,
            ))),
            opcode_indexes,
        }
    }
}

impl<M> Middleware<M> for Counter
where
    M: ModuleMod,
{
    type FM<'a> = FuncCounter<'a>;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let zero_count = GlobalInit::I64Const(0);
        let mut index_counts_global = self.index_counts_global.lock();
        for index in 0..OperatorCode::OPERATOR_COUNT {
            let count_global =
                module.add_global(&opcode_count_name(&index), Type::I64, zero_count)?;
            index_counts_global.push(count_global);
        }
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(FuncCounter::new(
            self.index_counts_global.clone(),
            self.opcode_indexes.clone(),
        ))
    }

    fn name(&self) -> &'static str {
        "operator counter"
    }
}

#[derive(Debug)]
pub struct FuncCounter<'a> {
    /// WASM global variables to keep track of opcode counts
    index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
    ///  Mapping of operator code to index for opcode_counts_global and block_opcode_counts
    opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
    /// Instructions of the current basic block
    block: Vec<Operator<'a>>,
    /// Number of times each opcode was used in current basic block
    block_index_counts: Vec<u64>,
}

impl<'a> FuncCounter<'a> {
    fn new(
        index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
        opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
    ) -> Self {
        Self {
            index_counts_global,
            opcode_indexes,
            block: vec![],
            block_index_counts: vec![0; OperatorCode::OPERATOR_COUNT],
        }
    }
}

impl<'a> FuncMiddleware<'a> for FuncCounter<'a> {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use Operator::*;

        macro_rules! opcode_count_add {
            ($self:expr, $op:expr, $count:expr) => {{
                let mut opcode_indexes = $self.opcode_indexes.lock();
                let next = opcode_indexes.len();
                let index = opcode_indexes.entry($op.into()).or_insert(next);
                assert!(
                    *index < OperatorCode::OPERATOR_COUNT,
                    "too many unique opcodes {next}"
                );
                $self.block_index_counts[*index] += $count * operator_factor($op);
            }};
        }

        let get_wasm_opcode_count_add = |global_index: u32, value: u64| -> Vec<Operator> {
            vec![
                GlobalGet { global_index },
                I64Const {
                    value: value as i64,
                },
                I64Add,
                GlobalSet { global_index },
            ]
        };

        let end = operator_at_end_of_basic_block(&op);

        opcode_count_add!(self, &op, 1);
        self.block.push(op);

        if end {
            // Ensure opcode count add instruction counts are all greater than zero
            for opcode in get_wasm_opcode_count_add(0, 0) {
                opcode_count_add!(self, &opcode, 1);
            }

            // Get list of all opcodes with nonzero counts
            let nonzero_counts: Vec<_> = self
                .index_counts_global
                .lock()
                .iter()
                .enumerate()
                .filter_map(
                    |(index, global_index)| match self.block_index_counts[index] {
                        0 => None,
                        count => Some((global_index.as_u32(), count)),
                    },
                )
                .collect();

            // Account for all wasm instructions added, minus 1 for what we already added above
            let unique_instructions = nonzero_counts.len() - 1;
            for opcode in get_wasm_opcode_count_add(0, 0) {
                opcode_count_add!(self, &opcode, unique_instructions as u64);
            }

            // Inject wasm instructions for adding counts
            for (global_index, count) in nonzero_counts {
                out.extend(get_wasm_opcode_count_add(global_index, count));
            }

            out.extend(self.block.clone());
            self.block.clear();
            self.block_index_counts = vec![0; OperatorCode::OPERATOR_COUNT]
        }
        Ok(())
    }

    fn name(&self) -> &'static str {
        "opcode counter"
    }
}

pub trait CountingMachine {
    fn get_opcode_counts(
        &mut self,
        opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
    ) -> Result<BTreeMap<OperatorCode, u64>>;
}

impl CountingMachine for Machine {
    fn get_opcode_counts(
        &mut self,
        opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
    ) -> Result<BTreeMap<OperatorCode, u64>> {
        Ok(opcode_indexes
            .lock()
            .clone()
            .iter()
            .filter_map(|(opcode, index)| -> Option<(OperatorCode, u64)> {
                let count = self
                    .get_global(&opcode_count_name(index))
                    .expect(&format!(
                        "global variable {} should have been present",
                        opcode_count_name(index)
                    ))
                    .try_into()
                    .expect(&format!(
                        "global variable {} should be u64",
                        opcode_count_name(index)
                    ));
                match count {
                    0 => None,
                    count => Some((*opcode, count)),
                }
            })
            .collect())
    }
}
