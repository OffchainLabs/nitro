use eyre::{bail, ensure};
use prover::machine::{GlobalState, Machine};
use serde::{Deserialize, Serialize};
use std::{
    collections::{HashMap, HashSet},
    fs::File,
    io::BufReader,
    path::PathBuf,
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
        line: usize,
        filename: String,
    },
    AssertReturn {
        line: usize,
        action: Action,
        expected: Vec<TextValue>,
    },
    AssertExhaustion {},
    AssertTrap {},

    Action {},
    AssertMalformed {},
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
    ty: String,
    value: String,
}

fn main() -> eyre::Result<()> {
    let opts = Opts::from_args();
    println!("test {:?}", opts.json);

    let mut path = PathBuf::from("tests/");
    path.push(&opts.json);

    let reader = BufReader::new(File::open(path)?);
    let case: Case = serde_json::from_reader(reader)?;

    let mut currentModule: Option<String> = None;
    let mut valid = HashSet::new();
    let mut invalid = HashSet::new();

    for command in &case.commands {
        use Command::*;

        match command {
            Module { line: _, filename } => {
                if let Some(prior) = currentModule {
                    invalid.insert(prior.clone());
                }
                currentModule = Some(filename.clone());
            }
            AssertReturn { .. } | AssertTrap { .. } | AssertExhaustion { .. } => {
                valid.insert(currentModule.clone().expect("no module"));
            }
            _ => {}
        }
    }

    let soft_float = PathBuf::from("../../target/machines/latest/soft-float.wasm");

    let mut wasmpath = PathBuf::new();
    let mut wasmfile = String::new();
    let mut machine = None;

    for command in case.commands {
        match command {
            Command::Module { line: _, filename } => {
                wasmpath = PathBuf::from("tests/");
                wasmpath.push(&filename);

                let mech = Machine::from_binary(
                    &[soft_float.clone()],
                    &wasmpath,
                    false,
                    false,
                    GlobalState::default(),
                    HashMap::default(),
                    HashMap::default(),
                );

                if mech.is_ok() && invalid.contains(&filename) {
                    bail!("failed to reject invalid module {}", filename);
                }
                if mech.is_err() && valid.contains(&filename) {
                    bail!("failed to accept valid module {}", filename);
                }

                match mech {
                    Ok(mech) => machine = Some(mech),
                    Err(err) if err.to_string().contains("Module has no code") => machine = None,
                    Err(err) => return Err(err),
                }

                wasmfile = filename;
            }
            Command::AssertReturn {
                line: _,
                action,
                expected,
            } => {
                let (field, args) = match action {
                    Action::Invoke { field, args } => (field, args),
                    _ => continue,
                };
                //println!("{} {:?}", field, args)
            }
            Command::AssertTrap {} => {}
            Command::AssertExhaustion {} => {}
            _ => {}
        }
    }

    Ok(())
}
