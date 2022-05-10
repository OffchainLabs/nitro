use eyre::bail;
use prover::{
    console::Color,
    machine::{GlobalState, Machine, MachineStatus},
    value::Value,
};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, fs::File, io::BufReader, path::PathBuf};
use structopt::StructOpt;

#[derive(StructOpt)]
#[structopt(name = "wasm-testsuite")]
struct Opts {
    json: PathBuf,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
struct Case {
    source_filename: String,
    commands: Vec<Command>,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum Command {
    Module {
        filename: String,
    },
    AssertReturn {
        action: Action,
        expected: Vec<TextValue>,
    },
    AssertExhaustion {},
    AssertTrap {
        action: Action,
    },
    Action {
        action: Action,
    },
    AssertMalformed {
        filename: String,
    },
    AssertInvalid {},
    AssertUninstantiable {},
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum Action {
    Invoke { field: String, args: Vec<TextValue> },
    Get { field: String },
}

#[derive(Clone, Debug, Serialize, Deserialize)]
struct TextValue {
    #[serde(rename = "type")]
    ty: TextValueType,
    value: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
enum TextValueType {
    I32,
    I64,
    F32,
    F64,
}

impl Into<Value> for TextValue {
    fn into(self) -> Value {
        match self.ty {
            TextValueType::I32 => {
                let value = self.value.parse().expect("not an i32");
                Value::I32(value)
            }
            TextValueType::I64 => {
                let value = self.value.parse().expect("not an i64");
                Value::I64(value)
            }
            TextValueType::F32 => {
                if self.value.contains("nan") {
                    return Value::F32(f32::NAN);
                }
                let message = format!("{} not the bit representation of an f32", self.value);
                let bits: u32 = self.value.parse().expect(&message);
                Value::F32(f32::from_bits(bits))
            }
            TextValueType::F64 => {
                if self.value.contains("nan") {
                    return Value::F64(f64::NAN);
                }
                let message = format!("{} not the bit representation of an f64", self.value);
                let bits: u64 = self.value.parse().expect(&message);
                Value::F64(f64::from_bits(bits))
            }
        }
    }
}

impl PartialEq<Value> for TextValue {
    fn eq(&self, other: &Value) -> bool {
        if &Into::<Value>::into(self.clone()) == other {
            return true;
        }

        match self.ty {
            TextValueType::F32 => match other {
                Value::F32(value) => value.is_nan() && self.value.contains("nan"),
                _ => false,
            },
            TextValueType::F64 => match other {
                Value::F64(value) => value.is_nan() && self.value.contains("nan"),
                _ => false,
            },
            _ => false,
        }
    }
}

fn pretty_print_values(prefix: &str, values: Vec<Value>) {
    let mut result = format!("  {}  ", prefix);
    for value in values {
        result += &format!("{}, ", value.pretty_print());
    }
    if result.len() > 2 {
        result.pop();
        result.pop();
    }
    println!("{}", result)
}

fn main() -> eyre::Result<()> {
    let opts = Opts::from_args();
    println!("test {:?}", opts.json);

    let mut path = PathBuf::from("tests/");
    path.push(&opts.json);

    let reader = BufReader::new(File::open(path)?);
    let case: Case = serde_json::from_reader(reader)?;

    let soft_float = PathBuf::from("../../target/machines/latest/soft-float.wasm");

    let mut wasmfile = String::new();
    let mut machine = None;
    let mut subtest = 0;

    for (index, command) in case.commands.into_iter().enumerate() {
        macro_rules! test_success {
            ($func:expr, $args:expr, $expected:expr) => {
                let args: Vec<_> = $args.into_iter().map(Into::into).collect();

                let machine = machine.as_mut().unwrap();
                machine.jump_into_function(&$func, args.clone());
                machine.step_n(10000);

                let output = match machine.get_final_result() {
                    Ok(output) => output,
                    Err(error) => {
                        let expected: Vec<Value> = $expected.into_iter().map(Into::into).collect();
                        println!(
                            "Divergence in func {} of test {}",
                            Color::red($func),
                            Color::red(index),
                        );
                        pretty_print_values("Args    ", args);
                        pretty_print_values("Expected", expected);
                        println!();
                        bail!("{}", error)
                    }
                };

                if $expected != output {
                    let expected: Vec<Value> = $expected.into_iter().map(Into::into).collect();
                    println!(
                        "Divergence in func {} of test {}",
                        Color::red($func),
                        Color::red(index),
                    );
                    pretty_print_values("Args    ", args);
                    pretty_print_values("Expected", expected);
                    pretty_print_values("Observed", output);
                    println!();
                    bail!(
                        "Failure in test {}",
                        Color::red(format!("{} #{}", wasmfile, subtest))
                    )
                }
                subtest += 1;
            };
        }

        match command {
            Command::Module { filename } => {
                wasmfile = filename;
                machine = None;
                subtest = 1;

                let mech = Machine::from_binary(
                    &[soft_float.clone()],
                    &PathBuf::from("tests").join(&wasmfile),
                    false,
                    false,
                    false,
                    GlobalState::default(),
                    HashMap::default(),
                    HashMap::default(),
                );

                if let Err(error) = &mech {
                    let error = error.root_cause().to_string();

                    if error.contains("Module has no code") {
                        // We don't support metadata-only modules that have no code
                        continue;
                    }
                    if error.contains("Unsupported import") {
                        // We don't support the import test's functions
                        continue;
                    }
                    if error.contains("multiple tables") {
                        // We don't support the reference-type extension
                        continue;
                    }
                    if error.contains("bulk memory") {
                        // We don't support the bulk-memory extension
                        continue;
                    }
                    bail!("Unexpected error parsing module {}: {}", wasmfile, error)
                }

                machine = mech.ok();

                if let Some(machine) = &mut machine {
                    machine.step_n(1000);
                }
            }
            Command::AssertReturn { action, expected } => {
                let (func, args) = match action {
                    Action::Invoke { field, args } => (field, args),
                    _ => bail!("unimplemented"),
                };
                test_success!(func, args, expected);
            }
            Command::Action { action } => {
                let (func, args) = match action {
                    Action::Invoke { field, args } => (field, args),
                    _ => bail!("unimplemented"),
                };
                let expected: Vec<TextValue> = vec![];
                test_success!(func, args, expected);
            }
            Command::AssertTrap { action } => {
                let (func, args) = match action {
                    Action::Invoke { field, args } => (field, args),
                    _ => bail!("unimplemented"),
                };

                let args: Vec<_> = args.into_iter().map(Into::into).collect();

                let machine = machine.as_mut().unwrap();
                machine.jump_into_function(&func, args.clone());
                machine.step_n(1000);

                let test = Color::red(format!("{} #{}", wasmfile, subtest));

                if machine.get_status() == MachineStatus::Running {
                    bail!("machine failed to trap in test {}", test)
                }
                if let Ok(output) = machine.get_final_result() {
                    println!(
                        "Divergence in func {} of test {}",
                        Color::red(func),
                        Color::red(index),
                    );
                    pretty_print_values("Args  ", args);
                    pretty_print_values("Output", output);
                    println!();
                    bail!("Unexpected success in test {}", test)
                }

                subtest += 1;
            }
            Command::AssertExhaustion {} => {
                //subtest += 1;
                unimplemented!("here")
            }
            Command::AssertMalformed { filename } => {
                let wasmpath = PathBuf::from("tests").join(&filename);

                Machine::from_binary(
                    &[soft_float.clone()],
                    &wasmpath,
                    false,
                    false,
                    false,
                    GlobalState::default(),
                    HashMap::default(),
                    HashMap::default(),
                )
                .expect_err(&format!("failed to reject invalid module {}", filename));
            }
            _ => {}
        }
    }

    Ok(())
}
