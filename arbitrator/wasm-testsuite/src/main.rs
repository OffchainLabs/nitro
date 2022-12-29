// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Color;
use eyre::{bail, ErrReport};
use prover::{
    machine,
    machine::{GlobalState, Machine, MachineStatus, ProofInfo},
    value::Value,
};
use serde::{Deserialize, Serialize};
use std::{
    collections::{HashMap, HashSet},
    convert::TryInto,
    fs::File,
    io::BufReader,
    path::PathBuf,
    time::Instant,
};
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
    AssertExhaustion {
        action: Action,
    },
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
    Register {},
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
    value: TextValueData,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
enum TextValueType {
    I32,
    I64,
    F32,
    F64,
    V128,
    Funcref,
    Externref,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(untagged)]
enum TextValueData {
    String(String),
    Array(Vec<String>),
}

impl TryInto<Value> for TextValue {
    type Error = ErrReport;

    fn try_into(self) -> eyre::Result<Value> {
        let TextValueData::String(value) = self.value else {
            bail!("array-expressed values not supported");
        };

        use TextValueType::*;
        Ok(match self.ty {
            I32 => {
                let value = value.parse().expect("not an i32");
                Value::I32(value)
            }
            I64 => {
                let value = value.parse().expect("not an i64");
                Value::I64(value)
            }
            F32 => {
                if value.contains("nan") {
                    return Ok(Value::F32(f32::NAN));
                }
                let message = format!("{} not the bit representation of an f32", value);
                let bits: u32 = value.parse().expect(&message);
                Value::F32(f32::from_bits(bits))
            }
            F64 => {
                if value.contains("nan") {
                    return Ok(Value::F64(f64::NAN));
                }
                let message = format!("{} not the bit representation of an f64", value);
                let bits: u64 = value.parse().expect(&message);
                Value::F64(f64::from_bits(bits))
            }
            x @ (V128 | Funcref | Externref) => bail!("not supported {:?}", x),
        })
    }
}

impl PartialEq<Value> for TextValue {
    fn eq(&self, other: &Value) -> bool {
        if &TryInto::<Value>::try_into(self.clone()).unwrap() == other {
            return true;
        }

        let TextValueData::String(text_value) = &self.value else {
            panic!("array-expressed values not supported");
        };

        match self.ty {
            TextValueType::F32 => match other {
                Value::F32(value) => value.is_nan() && text_value.contains("nan"),
                _ => false,
            },
            TextValueType::F64 => match other {
                Value::F64(value) => value.is_nan() && text_value.contains("nan"),
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
    let start_time = Instant::now();

    let soft_float = PathBuf::from("../../target/machines/latest/soft-float.wasm");

    // The modules listed below will be tested for compliance with the spec, but won't produce proofs for the OSP test.
    // We list the soft-float modules because, while compliance is necessary, the funcs are comprised of opcodes
    // better tested elsewhere and aren't worth 10x the test time.
    let mut do_not_prove = HashSet::new();
    do_not_prove.insert(PathBuf::from("f32.json"));
    do_not_prove.insert(PathBuf::from("f64.json"));
    do_not_prove.insert(PathBuf::from("f32_cmp.json"));
    do_not_prove.insert(PathBuf::from("f64_cmp.json"));
    do_not_prove.insert(PathBuf::from("float_exprs.json"));
    let export_proofs = !do_not_prove.contains(&opts.json);
    if !export_proofs {
        println!("{}", "skipping OSP proof generation".grey());
    }

    fn setup<'a>(
        machine: &'a mut Option<Machine>,
        func: &str,
        args: Vec<Value>,
        file: &str,
    ) -> &'a mut Machine {
        let Some(machine) = machine.as_mut() else {
            panic!("no machine {} {}", file.red(), func.red())
        };
        let main = machine.main_module_name();
        let (module, func) = machine.find_module_func(&main, func).unwrap();
        machine.jump_into_func(module, func, args);
        machine
    }

    fn to_values(text: Vec<TextValue>) -> eyre::Result<Vec<Value>> {
        text.into_iter().map(TryInto::try_into).collect()
    }

    let mut wasmfile = String::new();
    let mut machine = None;
    let mut subtest = 0;
    let mut skip = false;
    let mut has_skipped = false;

    macro_rules! run {
        ($machine:expr, $bound:expr, $path:expr, $prove:expr) => {{
            let mut proofs = vec![];
            let mut count = 0;
            let mut leap = 1;
            let prove = $prove && export_proofs;

            if !prove {
                $machine.step_n($bound)?;
            }

            while count + leap < $bound && prove {
                count += 1;

                let prior = $machine.hash().to_string();
                let proof = hex::encode($machine.serialize_proof());
                $machine.step_n(1)?;
                let after = $machine.hash().to_string();
                proofs.push(ProofInfo::new(prior, proof, after));
                $machine.step_n(leap - 1)?;

                if count % 100 == 0 {
                    leap *= leap + 1;
                    if leap > 6 {
                        let message = format!("backing off {} {} {}", leap, count, $bound);
                        println!("{}", message.grey());
                        $machine.stop_merkle_caching();
                    }
                }
                if $machine.is_halted() {
                    break;
                }
            }
            if prove {
                let out = File::create($path)?;
                serde_json::to_writer_pretty(out, &proofs)?;
            }
        }};
    }
    macro_rules! action {
        ($action:expr) => {
            match $action {
                Action::Invoke { field, args } => (field, args),
                Action::Get { .. } => {
                    // get() is only used in the export test, which we don't support
                    println!("skipping unsupported action {}", "get".red());
                    continue;
                }
            }
        };
    }
    macro_rules! outname {
        () => {
            format!(
                "../../contracts/test/prover/spec-proofs/{}-{:04}.json",
                wasmfile, subtest
            )
        };
    }

    'next: for (index, command) in case.commands.into_iter().enumerate() {
        // each iteration represets a test case

        macro_rules! test_success {
            ($func:expr, $args:expr, $expected:expr) => {
                let args = match to_values($args) {
                    Ok(args) => args,
                    Err(_) => continue, // TODO: can't use let-else due to rust fmt bug
                };
                if skip {
                    if !has_skipped {
                        println!("skipping {}", $func.red());
                    }
                    subtest += 1;
                    has_skipped = true;
                    continue;
                }

                let machine = setup(&mut machine, &$func, args.clone(), &wasmfile);
                machine.start_merkle_caching();
                run!(machine, 10_000_000, outname!(), true);

                let output = match machine.get_final_result() {
                    Ok(output) => output,
                    Err(error) => {
                        let expected = to_values($expected)?;
                        println!("Divergence in func {} of test {}", $func.red(), index.red());
                        pretty_print_values("Args    ", args);
                        pretty_print_values("Expected", expected);
                        println!();
                        bail!("{}", error)
                    }
                };

                if $expected != output {
                    let expected = to_values($expected)?;
                    println!("Divergence in func {} of test {}", $func.red(), index.red());
                    pretty_print_values("Args    ", args);
                    pretty_print_values("Expected", expected);
                    pretty_print_values("Observed", output);
                    println!();
                    bail!(
                        "Failure in test {}",
                        format!("{} #{}", wasmfile, subtest).red()
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

                let mech = Machine::from_paths(
                    &[soft_float.clone()],
                    &PathBuf::from("tests").join(&wasmfile),
                    false,
                    false,
                    false,
                    GlobalState::default(),
                    HashMap::default(),
                    machine::get_empty_preimage_resolver(),
                );

                if let Err(error) = &mech {
                    let error = error.root_cause().to_string();
                    skip = true;

                    let skippables = vec![
                        "module has no code", // we don't support metadata-only modules that have no code
                        "no such import",     // we don't support imports
                        "unsupported import", // we don't support imports
                        "reference types",    // we don't support the reference-type extension
                        "multiple tables",    // we don't support the reference-type extension
                        "bulk memory",        // we don't support the bulk-memory extension
                        "simd support",       // we don't support the SIMD extension
                    ];

                    for skippable in skippables {
                        if error.to_lowercase().contains(skippable) {
                            continue 'next;
                        }
                    }
                    bail!("Unexpected error parsing module {}: {}", wasmfile, error)
                }

                machine = mech.ok();
                skip = false;

                if let Some(machine) = &mut machine {
                    machine.step_n(1000)?; // run init
                    machine.start_merkle_caching();
                }
            }
            Command::AssertReturn { action, expected } => {
                let (func, args) = action!(action);
                test_success!(func, args, expected);
            }
            Command::Action { action } => {
                let (func, args) = action!(action);
                let expected: Vec<TextValue> = vec![];
                test_success!(func, args, expected);
            }
            Command::AssertTrap { action } => {
                let (func, args) = action!(action);
                let args = to_values(args)?;
                let test = format!("{} #{}", wasmfile, subtest).red();

                let machine = setup(&mut machine, &func, args.clone(), &wasmfile);
                run!(machine, 1000, outname!(), true);

                if machine.get_status() == MachineStatus::Running {
                    bail!("machine failed to trap in test {}", test)
                }
                if let Ok(output) = machine.get_final_result() {
                    println!("Divergence in func {} of test {}", func.red(), index.red());
                    pretty_print_values("Args  ", args);
                    pretty_print_values("Output", output);
                    println!();
                    bail!("Unexpected success in test {}", test)
                }
                subtest += 1;
            }
            Command::AssertExhaustion { action } => {
                let (func, args) = action!(action);
                let args = to_values(args)?;
                let test = format!("{} #{}", wasmfile, subtest).red();

                let machine = setup(&mut machine, &func, args.clone(), &wasmfile);
                run!(machine, 100_000, outname!(), false); // this is proportional to the amount of RAM

                if machine.get_status() != MachineStatus::Running {
                    bail!("machine should spin {}", test)
                }
                subtest += 1;
            }
            Command::AssertMalformed { filename } => {
                let wasmpath = PathBuf::from("tests").join(&filename);

                let _ = Machine::from_paths(
                    &[soft_float.clone()],
                    &wasmpath,
                    false,
                    false,
                    false,
                    GlobalState::default(),
                    HashMap::default(),
                    machine::get_empty_preimage_resolver(),
                )
                .expect_err(&format!("failed to reject invalid module {}", filename));
            }
            _ => {}
        }
    }

    println!(
        "{} {}",
        "done in".grey(),
        format!("{}ms", start_time.elapsed().as_millis()).pink()
    );
    Ok(())
}
