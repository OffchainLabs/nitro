use std::{collections::HashMap, path::PathBuf, str::FromStr, sync::Arc, time::SystemTime};

use clap::Parser;
use sp1_core_executor::{MinimalExecutor, Program};
use sp1_sdk::{Elf, include_elf};
use wasmer::{
    Module, Store,
    sys::{CpuFeature, EngineBuilder, LLVM, Target, Triple},
};

pub const PROGRAM_ELF: Elf = include_elf!("program");

#[derive(Debug, Parser)]
#[command(version, about, long_about = None)]
struct Cli {
    /// Path to replay.wasm file
    #[arg(long)]
    replay_wasm: String,

    /// Output folder for generated artifacts
    #[arg(long)]
    output_folder: PathBuf,
}

fn main() {
    let cli = Cli::parse();
    let wasm = std::fs::read(&cli.replay_wasm).expect("read replay.wasm");

    // Step 1: given wasm file, extract the original function names.
    // This information is lost in wasmer's generated binary, but the
    // function names can be very useful in debugging & profiling.
    let function_names_json = {
        use wasmparser::{BinaryReader, Name, NameSectionReader, Parser, Payload};

        let mut name_mapping = HashMap::new();
        for payload in Parser::new(0).parse_all(&wasm) {
            match payload {
                Ok(Payload::CustomSection(s)) if s.name() == "name" => {
                    let name_reader =
                        NameSectionReader::new(BinaryReader::new(s.data(), s.data_offset()));
                    for name in name_reader {
                        if let Name::Function(name_map) = name.expect("name") {
                            for naming in name_map {
                                let naming = naming.expect("naming");
                                name_mapping.insert(naming.index, naming.name.to_string());
                            }
                        }
                    }
                }
                _ => (),
            }
        }

        // names might be sparse, we need more processing work here.
        let min_index = name_mapping.keys().copied().min().unwrap();
        let name_mapping: Vec<(usize, String)> = name_mapping
            .into_iter()
            .map(|(index, name)| ((index - min_index) as usize, name))
            .collect();
        let mut names = vec![None; name_mapping.iter().max_by_key(|(i, _)| i).unwrap().0 + 1];
        for (i, name) in name_mapping {
            names[i] = Some(name);
        }

        let names_json = serde_json::to_string_pretty(&names).expect("serde_json");

        let output = cli.output_folder.join("function_names.json");
        println!("Write wasm function names to {}", output.display());
        std::fs::write(output, &names_json).expect("write function_names.json");

        names_json
    };

    // Step 2: build wasmu (riscv64 target) file from replay.wasm,
    // using wasmer's LLVM compiler.
    let wasmu_binary = {
        let target = Target::new(Triple::from_str("riscv64").unwrap(), CpuFeature::set());
        let config = LLVM::new();
        let engine = EngineBuilder::new(config).set_target(Some(target)).engine();

        let store = Store::new(engine);
        let module = Module::new(&store, wasm).expect("wasm compilation");
        let wasmu_binary = module.serialize().expect("serialize module");

        let output = cli.output_folder.join("replay.wasmu");
        println!("Compile {} to {}", cli.replay_wasm, output.display());
        std::fs::write(output, &wasmu_binary).expect("write replay.wasmu");

        wasmu_binary
    };

    // Step 3: use generated data from previous steps to bootload SP1 program.
    {
        sp1_sdk::utils::setup_logger();

        let output = match std::env::var("DUMP_ELF_OUTPUT") {
            Ok(s) => s,
            Err(_) => {
                let output = cli.output_folder.join("dumped_replay_wasm.elf");
                unsafe { std::env::set_var("DUMP_ELF_OUTPUT", &output) };
                output.display().to_string()
            }
        };
        let _ = std::fs::remove_file(&output);
        assert!(!std::fs::exists(&output).unwrap());

        let program = Arc::new(Program::from(&PROGRAM_ELF).expect("parse elf"));
        let mut executor = MinimalExecutor::simple(program);
        executor.with_input(&wasmu_binary);
        executor.with_input(function_names_json.as_bytes());
        // The executed program expects an Arbitrum block, sending it an
        // empty buffer would fail. However, it does not matter here, since
        // all we need to do is the bootloading process, which should finish
        // before reading this input.
        executor.with_input(&[]);

        // The executor will fail after bootloading completes because
        // the empty input buffer cannot be parsed as an Arbitrum block.
        // This is expected — we only need the bootloading side-effect (ELF dump).
        let t0 = SystemTime::now();
        let _ = executor.execute_chunk();
        let time_secs = t0.elapsed().unwrap().as_secs_f64();

        if let Ok(true) = std::fs::exists(&output) {
            println!("SP1 bootloading process completed successfully.");
        } else {
            panic!(
                "SP1 bootloading failed: expected output at '{}' was not produced. \
                 Check logs above for the root cause.",
                output
            );
        }

        tracing::info!(
            "[PROFILE] bootloading: cycles={}, time_secs={:.3}",
            executor.global_clk(),
            time_secs,
        );

        println!("Bootloaded program is written to {}", output);
    }

    println!("All build processes are completed!");
}
