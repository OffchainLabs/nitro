// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{FuncMiddleware, Middleware, ModuleMod};

use eyre::Result;
use fnv::FnvHashMap as HashMap;
use parking_lot::Mutex;
use std::{clone::Clone, fmt::Debug, sync::Arc};
use wasmer::{wasmparser::Operator, GlobalInit, Type};
use wasmer_types::{GlobalIndex, LocalFunctionIndex};

#[cfg(feature = "native")]
use super::native::{GlobalMod, NativeInstance};

macro_rules! opcode_count_name {
    ($val:expr) => {
        &format!("polyglot_opcode{}_count", $val)
    };
}

#[derive(Debug)]
pub struct Counter {
    pub max_unique_opcodes: usize,
    pub index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
    pub opcode_indexes: Arc<Mutex<HashMap<usize, usize>>>,
}

impl Counter {
    pub fn new(
        max_unique_opcodes: usize,
        opcode_indexes: Arc<Mutex<HashMap<usize, usize>>>,
    ) -> Self {
        Self {
            max_unique_opcodes,
            index_counts_global: Arc::new(Mutex::new(Vec::with_capacity(max_unique_opcodes))),
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
        for index in 0..self.max_unique_opcodes {
            let count_global =
                module.add_global(opcode_count_name!(index), Type::I64, zero_count)?;
            index_counts_global.push(count_global);
        }
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(FuncCounter::new(
            self.max_unique_opcodes,
            self.index_counts_global.clone(),
            self.opcode_indexes.clone(),
        ))
    }

    fn name(&self) -> &'static str {
        "opcode counter"
    }
}

#[derive(Debug)]
pub struct FuncCounter<'a> {
    /// Maximum number of unique opcodes to count
    max_unique_opcodes: usize,
    /// WASM global variables to keep track of opcode counts
    index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
    ///  Mapping of operator code to index for opcode_counts_global and block_opcode_counts
    opcode_indexes: Arc<Mutex<HashMap<usize, usize>>>,
    /// Instructions of the current basic block
    block: Vec<Operator<'a>>,
    /// Number of times each opcode was used in current basic block
    block_index_counts: Vec<u64>,
}

impl<'a> FuncCounter<'a> {
    fn new(
        max_unique_opcodes: usize,
        index_counts_global: Arc<Mutex<Vec<GlobalIndex>>>,
        opcode_indexes: Arc<Mutex<HashMap<usize, usize>>>,
    ) -> Self {
        Self {
            max_unique_opcodes,
            index_counts_global,
            opcode_indexes,
            block: vec![],
            block_index_counts: vec![0; max_unique_opcodes],
        }
    }
}

macro_rules! opcode_count_add {
    ($self:expr, $op:expr, $count:expr) => {{
        let code = operator_lookup_code($op);
        let mut opcode_indexes = $self.opcode_indexes.lock();
        let next = opcode_indexes.len();
        let index = opcode_indexes.entry(code).or_insert(next);
        assert!(
            *index < $self.max_unique_opcodes,
            "too many unique opcodes {next}"
        );
        $self.block_index_counts[*index] += $count;
    }};
}

macro_rules! get_wasm_opcode_count_add {
    ($global_index:expr, $count:expr) => {
        vec![
            GlobalGet {
                global_index: $global_index,
            },
            I64Const {
                value: $count as i64,
            },
            I64Add,
            GlobalSet {
                global_index: $global_index,
            },
        ]
    };
}

impl<'a> FuncMiddleware<'a> for FuncCounter<'a> {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use arbutil::operator::operator_lookup_code;
        use Operator::*;

        macro_rules! dot {
            ($first:ident $(,$opcode:ident)*) => {
                $first { .. } $(| $opcode { .. })*
            };
        }

        let end = matches!(
            op,
            End | Else | Return | dot!(Loop, Br, BrTable, BrIf, Call, CallIndirect)
        );

        opcode_count_add!(self, &op, 1);
        self.block.push(op);

        if end {
            // Ensure opcode count add instruction counts are all greater than zero
            for opcode in get_wasm_opcode_count_add!(0, 0) {
                opcode_count_add!(self, &opcode, 1);
            }

            // Get list of all opcodes with nonzero counts
            let mut nonzero_opcodes: Vec<(u32, usize)> =
                Vec::with_capacity(self.max_unique_opcodes);
            for (index, global_index) in self.index_counts_global.lock().iter().enumerate() {
                if self.block_index_counts[index] > 0 {
                    nonzero_opcodes.push((global_index.as_u32(), index));
                }
            }

            // Account for all wasm instructions added, minus 1 for what we already added above
            let unique_instructions = nonzero_opcodes.len() - 1;
            for opcode in get_wasm_opcode_count_add!(0, 0) {
                opcode_count_add!(self, &opcode, unique_instructions as u64);
            }

            // Inject wasm instructions for adding counts
            for (global_index, index) in nonzero_opcodes {
                out.extend(get_wasm_opcode_count_add!(
                    global_index,
                    self.block_index_counts[index]
                ));
            }

            out.extend(self.block.clone());
            self.block.clear();
            self.block_index_counts = vec![0; self.max_unique_opcodes]
        }
        Ok(())
    }

    fn name(&self) -> &'static str {
        "opcode counter"
    }
}

/// Note: implementers may panic if uninstrumented
pub trait CountedMachine {
    fn opcode_counts(&mut self, opcode_count: usize) -> Vec<u64>;
    fn set_opcode_counts(&mut self, index_counts: Vec<u64>);
}

#[cfg(feature = "native")]
impl CountedMachine for NativeInstance {
    fn opcode_counts(&mut self, opcode_count: usize) -> Vec<u64> {
        let mut counts = Vec::with_capacity(opcode_count);
        for i in 0..opcode_count {
            counts.push(self.get_global(opcode_count_name!(i)).unwrap());
        }

        counts
    }

    fn set_opcode_counts(&mut self, index_counts: Vec<u64>) {
        for (index, count) in index_counts.iter().enumerate() {
            self.set_global(opcode_count_name!(index), *count).unwrap();
        }
    }
}
