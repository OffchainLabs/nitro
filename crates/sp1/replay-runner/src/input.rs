use crate::cli::Cli;
use crate::execute_minimal;
use anyhow::{Context, Result};
use arbutil::Bytes32;
use sp1_core_executor::Program;
use sp1_sdk::SP1Stdin;
use std::collections::HashMap;
use std::ops::Deref;
use std::sync::Arc;
use stylus_compiler_program::CompileInput;
use validation::{UserWasm, ValidationInput};

/// Builds the validation payload from a recorded block: the rkyv-serialized `ValidationInput` and
/// turns it into a ready-to-consume SP1 stdin.
pub fn build_stdin(cli: &Cli) -> Result<SP1Stdin> {
    let request = cli.validation_request()?;
    let mut input = ValidationInput::from_request_allowing_missing_binaries(&request, "rv64")
        .map_err(anyhow::Error::msg)
        .context("build validation input")?;
    if let Some(wasms) = request.user_wasms.get("wasm") {
        compile_user_wasms(cli, wasms, &mut input, request.debug_chain)?;
    }
    let serialized = rkyv::to_bytes::<rkyv::rancor::Error>(&input)
        .context("rkyv-serialize validation input")?
        .to_vec();
    Ok(replay_io::send::validation_stdin(&serialized))
}

/// Goes over raw user wasm sources in `input` that are not already compiled to rv64 and compiles
/// them using SP1 Stylus compiler.
fn compile_user_wasms(
    cli: &Cli,
    wasms: &HashMap<Bytes32, UserWasm>,
    input: &mut ValidationInput,
    debug_chain: bool,
) -> Result<()> {
    let compiler = Arc::new(cli.stylus_compiler_program()?);
    for (module_hash, wasm) in wasms.iter() {
        if input.module_asms.contains_key(module_hash.deref()) {
            continue; // already compiled
        }
        let compiled = compile_in_sp1(
            compiler.clone(),
            wasm.as_ref(),
            cli.stylus_version,
            debug_chain,
        )?;
        input.module_asms.insert(**module_hash, compiled);
    }
    Ok(())
}

/// Compiles a Stylus wasm to a rv64 binary by running the stylus compiler inside SP1.
fn compile_in_sp1(
    compiler: Arc<Program>,
    wasm: &[u8],
    version: u16,
    debug: bool,
) -> Result<Vec<u8>> {
    let compile_input = CompileInput {
        version,
        debug,
        wasm: wasm.to_vec(),
    };

    let mut stdin = SP1Stdin::new();
    stdin.write(&compile_input);

    let compiled = execute_minimal(compiler, stdin).context("stylus compilation in SP1")?;
    bincode::deserialize(&compiled).context("deserialize compiled binary")
}
