// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    host::InternalFunc,
    machine::Module,
    value::{FunctionType, Value},
    wavm::{self, Opcode},
};
use arbutil::Color;
use fnv::FnvHashSet as HashSet;
use num_traits::FromPrimitive;
use std::fmt::{self, Display};
use wasmer_types::WASM_PAGE_SIZE;

impl FunctionType {
    fn wat_string(&self, name_args: bool) -> String {
        let params = if !self.inputs.is_empty() {
            let inputs = self.inputs.iter().enumerate();
            let params = inputs.fold(String::new(), |acc, (j, ty)| match name_args {
                true => format!("{acc} {} {}", format!("$arg{j}").pink(), ty.mint()),
                false => format!("{acc} {}", ty.mint()),
            });
            format!(" ({}{params})", "param".grey())
        } else {
            String::new()
        };

        let results = if !self.outputs.is_empty() {
            let outputs = self.outputs.iter();
            let results = outputs.fold(String::new(), |acc, t| format!("{acc} {t}"));
            format!(" ({}{})", "result".grey(), results.mint())
        } else {
            String::new()
        };

        format!("{params}{results}")
    }
}

impl Module {
    fn func_name(&self, i: u32) -> String {
        match self.maybe_func_name(i) {
            Some(func) => format!("${func}"),
            None => format!("$func_{i}"),
        }
        .pink()
    }

    fn maybe_func_name(&self, i: u32) -> Option<String> {
        if let Some(name) = self.names.functions.get(&i) {
            Some(name.to_owned())
        } else if i >= self.internals_offset {
            InternalFunc::from_u32(i - self.internals_offset).map(|f| format!("{f:?}"))
        } else {
            None
        }
    }
}

impl Display for Module {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        let mut pad = 0;

        macro_rules! w {
            ($($args:expr),*) => {{
                let text = format!($($args),*);
                write!(f, "{:pad$}{text}", "")?;
            }};
        }
        macro_rules! wln {
            ($($args:expr),*) => {{
                w!($($args),*);
                writeln!(f)?;
            }};
        }

        wln!("({} {}", "module".grey(), self.name().mint());
        pad += 4;

        for ty in &*self.types {
            let ty = ty.wat_string(false);
            wln!("({} ({}{ty}))", "type".grey(), "func".grey());
        }

        for (i, hook) in self.host_call_hooks.iter().enumerate() {
            if let Some((module, func)) = hook {
                wln!(
                    r#"({} "{}" "{}" ({} {}{}))"#,
                    "import".grey(),
                    module.pink(),
                    func.pink(),
                    "func".grey(),
                    self.func_name(i as u32),
                    self.funcs[i].ty.wat_string(false)
                );
            }
        }

        for (i, g) in self.globals.iter().enumerate() {
            let global_label = format!("$global_{i}").pink();
            wln!("({} {global_label} {})", "global".grey(), g.mint());
        }

        for (i, table) in self.tables.iter().enumerate() {
            let ty = table.ty;
            let initial = format!("{}", ty.initial).mint();
            let max = ty.maximum.map(|x| format!(" {x}")).unwrap_or_default();
            let type_str = format!("{:?}", ty.element_type).mint();
            w!(
                "({} {} {initial} {}{type_str}",
                "table".grey(),
                format!("$table_{i}").pink(),
                max.mint()
            );

            pad += 4;
            let mut empty = true;
            let mut segment = vec![];
            let mut start = None;
            let mut end = 0;
            for (j, elem) in table.elems.iter().enumerate() {
                if let Value::FuncRef(id) = elem.val {
                    segment.push(self.func_name(id));
                    start.get_or_insert(j);
                    end = j;
                    empty = false;
                }

                let last = j == table.elems.len() - 1;
                if (last || matches!(elem.val, Value::RefNull)) && !segment.is_empty() {
                    let start = start.unwrap();
                    wln!("");
                    w!("{}", format!("[{start:#05x}-{end:#05x}]:").grey());
                    for item in &segment {
                        write!(f, " {item}")?;
                    }
                    segment.clear();
                }
            }
            pad -= 4;
            if !empty {
                wln!("");
                w!("");
            }
            writeln!(f, ")")?;
        }

        let args = format!(
            "{} {}",
            self.memory.size() / WASM_PAGE_SIZE as u64,
            self.memory.max_size
        );
        w!("({} {}", "memory".grey(), args.mint());

        pad += 4;
        let mut empty = true;
        let mut segment = None;
        for index in 0..self.memory.size() {
            let byte = self.memory.get_u8(index).unwrap();

            // start new segment
            if byte != 0 && segment.is_none() {
                segment = Some(index as usize);
                empty = false;
            }

            // print the segment
            if (byte == 0x00 || index == self.memory.size() - 1) && segment.is_some() {
                let start = segment.unwrap();
                let end = index - 1 + (byte != 0x00) as u64;
                let len = end as usize - start + 1;
                let range = format!("[{start:#06x}-{end:#06x}]");
                let data = self.memory.get_range(start, len).unwrap();
                wln!("");
                w!("{}: {}", range.grey(), hex::encode(data).yellow());
                segment = None;
            }
        }
        pad -= 4;
        if !empty {
            wln!("");
            w!("");
        }
        writeln!(f, ")")?;

        for (i, func) in self.funcs.iter().enumerate() {
            let i1 = i as u32;
            let padding = 12;

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
            w!(
                "({}{}{}",
                "func".grey(),
                export_str,
                func.ty.wat_string(true)
            );

            pad += 4;
            if !func.local_types.is_empty() {
                write!(f, " ({}", "local".grey())?;
                for (i, ty) in func.local_types.iter().enumerate() {
                    let local_str = format!("$local_{i}");
                    write!(f, " {} {}", local_str.pink(), ty.mint())?;
                }
                write!(f, ")")?;
            }
            writeln!(f)?;

            let mut labels = HashSet::default();
            use Opcode::*;
            for op in func.code.iter() {
                if op.opcode == ArbitraryJump || op.opcode == ArbitraryJumpIf {
                    labels.insert(op.argument_data as usize);
                }
            }

            for (j, op) in func.code.iter().enumerate() {
                let op_str = format!("{:?}", op.opcode).grey();
                let arg_str = match op.opcode {
                    ArbitraryJump | ArbitraryJumpIf => {
                        match labels.get(&(op.argument_data as usize)) {
                            Some(label) => format!(" label_${label}").pink(),
                            None => " ???".to_string().red(),
                        }
                    }
                    Call
                    | CallerModuleInternalCall
                    | CrossModuleForward
                    | CrossModuleInternalCall => {
                        format!(" {}", self.func_name(op.argument_data as u32))
                    }
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
                            " {} {}",
                            self.types[type_index as usize].pink(),
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
                            format!(" UNEXPECTED_ARG: {}", op.argument_data).mint()
                        }
                    }
                };

                let proof = op
                    .proving_argument_data
                    .map(hex::encode)
                    .unwrap_or_default()
                    .orange();

                match labels.get(&j) {
                    Some(label) => {
                        let label = format!("label_{label}");
                        let spaces = padding - label.len() - 1;
                        wln!("{}:{:spaces$}{op_str}{arg_str} {proof}", label.pink(), "")
                    }
                    None => wln!("{:padding$}{op_str}{arg_str} {proof}", ""),
                }
            }
            pad -= 4;
            wln!(")");
        }

        if let Some(start) = self.start_function {
            wln!("({} {})", "start".grey(), self.func_name(start));
        }
        pad -= 4;
        wln!(")");
        Ok(())
    }
}
