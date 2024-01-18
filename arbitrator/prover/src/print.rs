// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    host,
    machine::Module,
    value::{FunctionType, Value},
    wavm::{self, Opcode},
};
use arbutil::Color;
use fnv::FnvHashMap as HashMap;
use num_traits::FromPrimitive;
use std::fmt::{self, Display};

impl FunctionType {
    fn wat_string(&self) -> String {
        let param_str = if self.inputs.len() > 0 {
            let param_str = self
                .inputs
                .iter()
                .enumerate()
                .fold(String::new(), |acc, (j, ty)| {
                    format!("{} {} {}", acc, format!("$arg{}", j).pink(), ty.mint())
                });
            format!(" ({}{})", "param".grey(), param_str)
        } else {
            String::new()
        };

        let result_str = if self.outputs.len() > 0 {
            let result_str = self
                .outputs
                .iter()
                .fold(String::new(), |acc, t| format!("{acc} {t}"));
            format!(" {}{})", "result".grey(), result_str.mint())
        } else {
            String::new()
        };

        format!(" {param_str}{result_str}")
    }
}

impl Module {
    fn func_name(&self, i: u32) -> String {
        match self.maybe_func_name(i) {
            Some(func) => format!(" ${func}"),
            None => format!(" $func_{i}"),
        }
    }

    fn maybe_func_name(&self, i: u32) -> Option<String> {
        if i < self.internals_offset {
            // imported function or user function
            self.names.functions.get(&i).cloned()
        } else {
            // internal function
            host::InternalFunc::from_u32(i - self.internals_offset)
                .map_or(None, |f| Some(format!("{:?}", f)))
        }
    }
}

impl Display for Module {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        let mut level = 0;
        writeln!(
            f,
            "{:level$}({} {}",
            "",
            "module".grey(),
            self.name().mint()
        )?;
        level += 2;

        for (i, g) in self.globals.iter().enumerate() {
            let global_label = format!("$global_{}", i).pink();
            let global_str = format!("{:?}", g).mint();
            writeln!(
                f,
                "{:level$}({} {} {})",
                "",
                "global".grey(),
                global_label,
                global_str
            )?;
        }

        for ty in self.types.iter() {
            writeln!(
                f,
                "{:level$}({} ({}{}))",
                "",
                "type".grey(),
                "func".grey(),
                ty.wat_string()
            )?;
        }

        for (i, table) in self.tables.iter().enumerate() {
            let initial_str = format!("{}", table.ty.initial);
            let max_str = match table.ty.maximum {
                Some(max) => format!(" {max}"),
                None => String::new(),
            };
            let type_str = format!("{:?}", table.ty.element_type);
            writeln!(
                f,
                "{:level$}({} {} {} {} {})",
                "",
                "table".grey(),
                format!("$table_{i}").pink(),
                initial_str.mint(),
                max_str.mint(),
                type_str.mint()
            )?;
            for j in 1..table.elems.len() {
                let val = table.elems[j].val;
                let elem = match table.elems[j].val {
                    Value::FuncRef(id) => self.func_name(id).pink(),
                    Value::RefNull => {
                        continue;
                    }
                    _ => format!("{val}"),
                };
                writeln!(
                    f,
                    "{:level$}({} ({} {}) {})",
                    "",
                    "elem".grey(),
                    "I32Const".mint(),
                    format!("{j:#x}").mint(),
                    elem
                )?;
            }
        }

        for (i, hook) in self.host_call_hooks.iter().enumerate() {
            if let Some(hook) = hook {
                writeln!(
                    f,
                    r#"{:level$}({} "{}" "{}", ({} {}{}))"#,
                    "",
                    "import".grey(),
                    hook.0.pink(),
                    hook.1.pink(),
                    "func".grey(),
                    self.func_name(i as u32).pink(),
                    self.funcs[i].ty.wat_string()
                )?;
            }
        }

        let args = format!(
            "{} {}",
            (self.memory.size() + 65535) / 65536,
            self.memory.max_size
        );

        write!(f, "{:level$}({} {}", "", "memory".grey(), args.mint())?;
        let mut byte_index = 0;
        let mut nonzero_bytes = Vec::new();
        let mut first_nonzero_index = 0;
        level += 2;
        let mut empty = true;
        while byte_index < self.memory.max_size {
            let current_byte = match self.memory.get_u8(byte_index) {
                Some(byte) => byte,
                None => {
                    break;
                }
            };
            if current_byte != 0 {
                if nonzero_bytes.is_empty() {
                    first_nonzero_index = byte_index
                }
                nonzero_bytes.push(current_byte);
            }

            byte_index += 1;
            if (current_byte == 0 || byte_index == self.memory.max_size)
                && !nonzero_bytes.is_empty()
            {
                empty = false;
                let range = format!("[{:#06x}-{:#06x}]", first_nonzero_index, byte_index - 2);
                write!(
                    f,
                    "\n{:level$}{}: {}",
                    "",
                    range.grey(),
                    hex::encode(&nonzero_bytes).mint()
                )?;
                nonzero_bytes.clear();
            }
        }
        level -= 2;
        if empty {
            writeln!(f, ")")?;
        } else {
            writeln!(f, "\n{:level$})", "")?;
        }

        for (i, func) in self.funcs.iter().enumerate() {
            let i1 = i as u32;
            let padding = 11;

            let export_str = match self.maybe_func_name(i1) {
                Some(name) => {
                    let description = if (i1 as usize) < self.host_call_hooks.len() {
                        "import"
                    } else {
                        "export"
                    };
                    format!(r#" ({} "{}")"#, description.grey(), name.pink())
                }
                None => format!(" $func_{i}").pink(),
            };
            writeln!(
                f,
                "{:level$}({}{}{}",
                "",
                "func".grey(),
                export_str,
                func.ty.wat_string()
            )?;

            level += 2;
            for (i, ty) in func.local_types.iter().enumerate() {
                let local_str = format!("$local_{}", i);
                writeln!(
                    f,
                    "{:level$}{:padding$}{} {} {}",
                    "",
                    "",
                    "local".grey(),
                    local_str.pink(),
                    ty.mint()
                )?;
            }

            let mut labels = HashMap::default();
            use Opcode::*;
            for op in func.code.iter() {
                if op.opcode == ArbitraryJump || op.opcode == ArbitraryJumpIf {
                    labels.insert(
                        op.argument_data as usize,
                        format!("label_{}", op.argument_data),
                    );
                }
            }

            for (j, op) in func.code.iter().enumerate() {
                let op_str = format!("{:?}", op.opcode).grey();
                let arg_str = match op.opcode {
                    ArbitraryJump | ArbitraryJumpIf => {
                        match labels.get(&(op.argument_data as usize)) {
                            Some(label) => format!(" ${label}"),
                            None => " UNKNOWN".to_string(),
                        }
                        .pink()
                    }
                    Call
                    | CallerModuleInternalCall
                    | CrossModuleForward
                    | CrossModuleInternalCall => self.func_name(op.argument_data as u32).pink(),
                    CrossModuleCall => {
                        let (module, func) = wavm::unpack_cross_module_call(op.argument_data);
                        format!(
                            " {} {}",
                            format!("{module}").mint(),
                            format!("{func}").mint()
                        )
                    }
                    CallIndirect => {
                        let (table_index, type_index) =
                            wavm::unpack_call_indirect(op.argument_data);
                        format!(
                            "{} {}",
                            self.types[type_index as usize].to_string(),
                            format!("{table_index}").mint()
                        )
                    }
                    F32Const | F64Const | I32Const | I64Const => {
                        format!(" {:#x}", op.argument_data).mint()
                    }
                    GlobalGet | GlobalSet => format!(" $global_{}", op.argument_data).pink(),
                    LocalGet | LocalSet => format!(" $local_{}", op.argument_data).pink(),
                    MemoryLoad { .. } | MemoryStore { .. } | ReadInboxMessage => {
                        format!(" {:#x}", op.argument_data).mint()
                    }
                    _ => {
                        if op.argument_data == 0 {
                            String::new()
                        } else {
                            format!(" UNEXPECTED_ARGUMENT:{}", op.argument_data).mint()
                        }
                    }
                };
                let proving_str = if let Some(data) = op.proving_argument_data {
                    hex::encode(&data)
                } else {
                    String::new()
                }
                .orange();
                let label = labels.get(&j).cloned().unwrap_or(String::new());
                let (colon, padding) = if label.len() == 0 {
                    ("", padding)
                } else {
                    writeln!(f, "")?;
                    if label.len() >= padding - 1 {
                        (":", 1)
                    } else {
                        (":", padding - 1 - label.len())
                    }
                };
                let label = format!("{}{colon}{:padding$}", label.pink(), "");
                writeln!(f, "{:level$}{label}{op_str}{arg_str} {proving_str}", "")?;
            }
            level -= 2;
            writeln!(f, "{:level$})", "")?;
            ()
        }

        if let Some(start) = self.start_function {
            writeln!(
                f,
                "{:level$}{} {}",
                "",
                "start".grey(),
                self.func_name(start).pink()
            )?;
        }

        level -= 2;
        writeln!(f, "{:level$})", "")?;

        Ok(())
    }
}
