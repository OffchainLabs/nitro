// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{format, Color, DebugColor, PreimageType};
use eyre::{Context, Result};
use fnv::{FnvHashMap as HashMap, FnvHashSet as HashSet};
use prover::{
    machine::{GlobalState, InboxIdentifier, Machine, MachineStatus, PreimageResolver, ProofInfo},
    utils::{hash_preimage, Bytes32, CBytes},
    wavm::Opcode,
};
use std::sync::Arc;
use std::{convert::TryInto, io::BufWriter};
use std::{
    fs::File,
    io::{BufReader, ErrorKind, Read, Write},
    path::{Path, PathBuf},
};
use structopt::StructOpt;

#[derive(StructOpt)]
#[structopt(name = "arbitrator-prover")]
struct Opts {
    binary: PathBuf,
    #[structopt(short, long)]
    libraries: Vec<PathBuf>,
    #[structopt(short, long)]
    output: Option<PathBuf>,
    #[structopt(short = "b", long)]
    proving_backoff: bool,
    #[structopt(long)]
    allow_hostapi: bool,
    #[structopt(long)]
    inbox_add_stub_headers: bool,
    #[structopt(long)]
    always_merkleize: bool,
    /// profile output instead of generting proofs
    #[structopt(short = "p", long)]
    profile_run: bool,
    /// simple summary of hot opcodes
    #[structopt(long)]
    profile_sum_opcodes: bool,
    /// simple summary of hot functions
    #[structopt(long)]
    profile_sum_funcs: bool,
    /// profile written in "folded" format (use as input for e.g. inferno-flamegraph)
    #[structopt(long)]
    profile_output: Option<PathBuf>,
    #[structopt(short = "i", long, default_value = "1")]
    proving_interval: u64,
    #[structopt(short = "s", long, default_value = "0")]
    proving_start: u64,
    #[structopt(long, default_value = "0")]
    delayed_inbox_position: u64,
    #[structopt(long, default_value = "0")]
    inbox_position: u64,
    #[structopt(long, default_value = "0")]
    position_within_message: u64,
    #[structopt(long)]
    last_block_hash: Option<String>,
    #[structopt(long)]
    last_send_root: Option<String>,
    #[structopt(long)]
    inbox: Vec<PathBuf>,
    #[structopt(long)]
    delayed_inbox: Vec<PathBuf>,
    #[structopt(long)]
    preimages: Option<PathBuf>,
    /// Require that the machine end in the Finished state
    #[structopt(long)]
    require_success: bool,
    /// Generate WAVM binary, until host io state, and module root and exit
    #[structopt(long)]
    generate_binaries: Option<PathBuf>,
    #[structopt(long)]
    skip_until_host_io: bool,
    #[structopt(long)]
    max_steps: Option<u64>,
}

fn file_with_stub_header(path: &Path, headerlength: usize) -> Result<Vec<u8>> {
    let mut msg = vec![0u8; headerlength];
    File::open(path).unwrap().read_to_end(&mut msg)?;
    Ok(msg)
}

fn decode_hex_arg(arg: &Option<String>, name: &str) -> Result<Bytes32> {
    if let Some(arg) = arg {
        let mut arg = arg.as_str();
        if arg.starts_with("0x") {
            arg = &arg[2..];
        }
        let mut bytes32 = Bytes32::default();
        hex::decode_to_slice(arg, &mut bytes32.0)
            .wrap_err_with(|| format!("failed to parse {} contents", name))?;
        Ok(bytes32)
    } else {
        Ok(Bytes32::default())
    }
}

#[derive(Debug, Default, Copy, Clone, PartialEq, Eq)]
struct SimpleProfile {
    count: u64,
    total_cycles: u64,
    local_cycles: u64,
}

const INBOX_HEADER_LEN: usize = 40; // also in test-case's host-io.rs & contracts's OneStepProverHostIo.sol
const DELAYED_HEADER_LEN: usize = 112; // also in test-case's host-io.rs & contracts's OneStepProverHostIo.sol

fn main() -> Result<()> {
    let opts = Opts::from_args();

    let mut inbox_contents = HashMap::default();
    let mut inbox_position = opts.inbox_position;
    let mut delayed_position = opts.delayed_inbox_position;
    let inbox_header_len;
    let delayed_header_len;
    if opts.inbox_add_stub_headers {
        inbox_header_len = INBOX_HEADER_LEN;
        delayed_header_len = DELAYED_HEADER_LEN + 1;
    } else {
        inbox_header_len = 0;
        delayed_header_len = 0;
    }

    for path in opts.inbox {
        inbox_contents.insert(
            (InboxIdentifier::Sequencer, inbox_position),
            file_with_stub_header(&path, inbox_header_len)?,
        );
        println!("read file {:?} to seq. inbox {}", &path, inbox_position);
        inbox_position += 1;
    }
    for path in opts.delayed_inbox {
        inbox_contents.insert(
            (InboxIdentifier::Delayed, delayed_position),
            file_with_stub_header(&path, delayed_header_len)?,
        );
        delayed_position += 1;
    }

    let mut preimages: HashMap<PreimageType, HashMap<Bytes32, CBytes>> = HashMap::default();
    if let Some(path) = opts.preimages {
        let mut file = BufReader::new(File::open(path)?);
        loop {
            let mut ty_buf = [0u8; 1];
            match file.read_exact(&mut ty_buf) {
                Ok(()) => {}
                Err(e) if e.kind() == ErrorKind::UnexpectedEof => break,
                Err(e) => return Err(e.into()),
            }
            let preimage_ty: PreimageType = ty_buf[0].try_into()?;

            let mut size_buf = [0u8; 8];
            file.read_exact(&mut size_buf)?;
            let size = u64::from_le_bytes(size_buf) as usize;
            let mut buf = vec![0u8; size];
            file.read_exact(&mut buf)?;

            let hash = hash_preimage(&buf, preimage_ty)?;
            preimages
                .entry(preimage_ty)
                .or_default()
                .insert(hash.into(), buf.as_slice().into());
        }
    }
    let preimage_resolver =
        Arc::new(move |_, ty, hash| preimages.get(&ty).and_then(|m| m.get(&hash)).cloned())
            as PreimageResolver;

    let last_block_hash = decode_hex_arg(&opts.last_block_hash, "--last-block-hash")?;
    let last_send_root = decode_hex_arg(&opts.last_send_root, "--last-send-root")?;

    let global_state = GlobalState {
        u64_vals: [opts.inbox_position, opts.position_within_message],
        bytes32_vals: [last_block_hash, last_send_root],
    };

    let mut mach = Machine::from_paths(
        &opts.libraries,
        &opts.binary,
        true,
        opts.always_merkleize,
        opts.allow_hostapi,
        global_state,
        inbox_contents,
        preimage_resolver,
    )?;
    if let Some(output_path) = opts.generate_binaries {
        let mut module_root_file = File::create(output_path.join("module-root.txt"))?;
        writeln!(module_root_file, "0x{}", mach.get_modules_root())?;
        module_root_file.flush()?;

        mach.serialize_binary(output_path.join("machine.wavm.br"))?;
        while !mach.next_instruction_is_host_io() {
            mach.step_n(1)?;
        }
        mach.serialize_state(output_path.join("until-host-io-state.bin"))?;

        return Ok(());
    }

    println!("Starting machine hash: {}", mach.hash());

    let mut proofs: Vec<ProofInfo> = Vec::new();
    let mut seen_states = HashSet::default();
    let mut proving_backoff: HashMap<(Opcode, u64), usize> = HashMap::default();
    let mut opcode_profile: HashMap<Opcode, SimpleProfile> = HashMap::default();
    let mut func_profile: HashMap<(usize, usize), SimpleProfile> = HashMap::default();
    let mut func_stack: Vec<(usize, usize, SimpleProfile)> = Vec::default();
    let mut backtrace_stack: Vec<(usize, usize)> = Vec::default();
    let mut cycles_measured_total: u64 = 0;
    let mut profile_backtrace_counts: HashMap<Vec<(usize, usize)>, u64> = HashMap::default();
    let cycles_bigloop_start: u64;
    let cycles_bigloop_end: u64;
    #[cfg(target_arch = "x86_64")]
    unsafe {
        cycles_bigloop_start = core::arch::x86_64::_rdtsc();
    }
    mach.step_n(opts.proving_start)?;
    if opts.skip_until_host_io && !opts.profile_run {
        while !mach.next_instruction_is_host_io() {
            mach.step_n(1)?;
        }
    }
    let mut skipping_profiling = opts.skip_until_host_io;
    while !mach.is_halted() {
        if let Some(max_steps) = opts.max_steps {
            if mach.get_steps() >= max_steps {
                break;
            }
        }

        let next_inst = mach.get_next_instruction().unwrap();
        let next_opcode = next_inst.opcode;

        if opts.proving_backoff {
            let mut extra_data = 0;
            if matches!(next_opcode, Opcode::ReadInboxMessage | Opcode::ReadPreImage) {
                extra_data = next_inst.argument_data;
            }
            let count_entry = proving_backoff
                .entry((next_opcode, extra_data))
                .or_insert(0);
            *count_entry += 1;
            let count = *count_entry;
            // Apply an exponential backoff to how often to prove an instruction;
            let prove =
                count < 5 || (count < 25 && count % 5 == 0) || (count < 125 && count % 25 == 0);
            if !prove {
                mach.step_n(1)?;
                continue;
            }
        }

        if opts.profile_run {
            skipping_profiling = skipping_profiling && !mach.next_instruction_is_host_io();
            let start: u64;
            let end: u64;
            let pc = mach.get_pc().unwrap();
            #[cfg(target_arch = "x86_64")]
            unsafe {
                start = core::arch::x86_64::_rdtsc();
            }
            mach.step_n(1)?;
            #[cfg(target_arch = "x86_64")]
            unsafe {
                end = core::arch::x86_64::_rdtsc();
            }
            #[cfg(not(target_arch = "x86_64"))]
            {
                start = 0;
                end = 1;
            }
            let profile_time = end - start;

            if !skipping_profiling {
                cycles_measured_total += profile_time;
            }

            if opts.profile_sum_opcodes && !skipping_profiling {
                let opprofile = opcode_profile.entry(next_opcode).or_default();
                opprofile.count += 1;
                opprofile.total_cycles += profile_time;
            }

            if pc.inst == 0 {
                func_stack.push((pc.module(), pc.func(), SimpleProfile::default()));
                backtrace_stack.push((pc.module(), pc.func()));
            }
            let this_func_profile = &mut func_stack.last_mut().unwrap().2;
            if !skipping_profiling {
                this_func_profile.count += 1;
                this_func_profile.total_cycles += profile_time;
                this_func_profile.local_cycles += profile_time;
            }
            if next_opcode == Opcode::Return {
                let (module, func, profile) = func_stack.pop().unwrap();

                if opts.profile_sum_funcs && !skipping_profiling {
                    if let Some(parent_func) = &mut func_stack.last_mut() {
                        parent_func.2.count += profile.count;
                        parent_func.2.total_cycles += profile.total_cycles;
                    }
                    let func_profile_entry = func_profile.entry((module, func)).or_default();
                    func_profile_entry.count += profile.count;
                    func_profile_entry.total_cycles += profile.total_cycles;
                    func_profile_entry.local_cycles += profile.local_cycles;
                }

                if opts.profile_output.is_some() && !skipping_profiling {
                    *profile_backtrace_counts
                        .entry(backtrace_stack.clone())
                        .or_default() += profile.local_cycles;
                }
                backtrace_stack.pop();
            }
        } else {
            let values = mach.get_data_stack();
            if !values.is_empty() {
                println!("{} {}", "Machine stack".grey(), format::commas(values));
            }
            print!(
                "Generating proof {} (inst {}) for {}{}",
                proofs.len().blue(),
                mach.get_steps().blue(),
                next_opcode.debug_mint(),
                match next_inst.argument_data {
                    0 => "".into(),
                    v => format!(" with data 0x{v:x}"),
                }
            );
            std::io::stdout().flush().unwrap();
            let before = mach.hash();
            if !seen_states.insert(before) {
                break;
            }
            let proof = mach.serialize_proof();
            mach.step_n(1)?;
            let after = mach.hash();
            println!(" - done");
            proofs.push(ProofInfo {
                before: before.to_string(),
                proof: hex::encode(proof),
                after: after.to_string(),
            });
            mach.step_n(opts.proving_interval.saturating_sub(1))?;
        }
    }
    #[cfg(target_arch = "x86_64")]
    unsafe {
        cycles_bigloop_end = core::arch::x86_64::_rdtsc();
    }
    #[cfg(not(target_arch = "x86_64"))]
    {
        cycles_bigloop_start = 0;
        cycles_bigloop_end = 0;
    }

    let cycles_bigloop = cycles_bigloop_end - cycles_bigloop_start;

    if !proofs.is_empty() && mach.is_halted() {
        let hash = mach.hash();
        proofs.push(ProofInfo {
            before: hash.to_string(),
            proof: hex::encode(mach.serialize_proof()),
            after: hash.to_string(),
        });
    }

    println!("End machine status: {:?}", mach.get_status());
    println!("End machine hash: {}", mach.hash());
    println!("End machine stack: {:?}", mach.get_data_stack());
    println!("End machine backtrace:");
    for (module, func, pc) in mach.get_backtrace() {
        let func = rustc_demangle::demangle(&func);
        println!("  {} {} @ {}", module, func.mint(), pc.blue());
    }

    if let Some(out) = opts.output {
        let out = File::create(out)?;
        serde_json::to_writer_pretty(out, &proofs)?;
    }

    if opts.profile_run {
        let mut sum = SimpleProfile::default();
        while let Some((module, func, profile)) = func_stack.pop() {
            sum.total_cycles += profile.total_cycles;
            sum.count += profile.count;
            let entry = func_profile.entry((module, func)).or_default();
            entry.count += sum.count;
            entry.total_cycles += sum.total_cycles;
            entry.local_cycles += profile.local_cycles;
        }

        println!(
            "Total cycles measured {} out of {} in loop ({}%)",
            cycles_measured_total,
            cycles_bigloop,
            (cycles_measured_total as f64) * 100.0 / (cycles_bigloop as f64)
        );

        if opts.profile_sum_opcodes {
            println!("\n===Operations:");
            let mut ops_vector: Vec<_> = opcode_profile.iter().collect();
            ops_vector.sort_by(|a, b| b.1.total_cycles.cmp(&a.1.total_cycles));
            let mut printed = 0;
            for (opcode, profile) in ops_vector {
                println!(
                    "Opcode {:?}: steps: {} cycles: {} ({}%)",
                    opcode,
                    profile.count,
                    profile.total_cycles,
                    (profile.total_cycles as f64) * 100.0 / (cycles_measured_total as f64),
                );
                printed += 1;
                if printed > 20 {
                    break;
                }
            }
        }

        let opts_binary = opts.binary;
        let opts_libraries = opts.libraries;
        let format_pc = |module_num: usize, func_num: usize| -> (String, String) {
            let names = match mach.get_module_names(module_num) {
                Some(n) => n,
                None => {
                    return (
                        format!("[unknown {}]", module_num),
                        format!("[unknown {}]", func_num),
                    );
                }
            };
            let module_name = if module_num == 0 {
                names.module.clone()
            } else if module_num == &opts_libraries.len() + 1 {
                opts_binary.file_name().unwrap().to_str().unwrap().into()
            } else {
                opts_libraries[module_num - 1]
                    .file_name()
                    .unwrap()
                    .to_str()
                    .unwrap()
                    .into()
            };
            let func_idx = func_num as u32;
            let mut name = names
                .functions
                .get(&func_idx)
                .cloned()
                .unwrap_or_else(|| format!("[unknown {}]", func_idx));
            name = rustc_demangle::demangle(&name).to_string();
            (module_name, name)
        };

        if opts.profile_sum_funcs {
            println!("\n===Functions:");
            let mut func_vector: Vec<_> = func_profile.iter().collect();
            func_vector.sort_by(|a, b| b.1.total_cycles.cmp(&a.1.total_cycles));
            let mut printed = 0;
            for (&(module_num, func), profile) in func_vector {
                let (name, module_name) = format_pc(module_num, func);
                let percent =
                    (profile.total_cycles as f64) * 100.0 / (cycles_measured_total as f64);
                println!(
                    "module {}: function: {} {} steps: {} cycles: {} ({}%)",
                    module_name, func, name, profile.count, profile.total_cycles, percent,
                );
                printed += 1;
                if printed > 20 && percent < 3.0 {
                    break;
                }
            }
        }

        if let Some(out) = opts.profile_output {
            let mut out = BufWriter::new(File::create(out)?);
            for (backtrace, count) in profile_backtrace_counts {
                let mut path = String::new();
                let mut last_module = None;
                for (module, func) in backtrace {
                    let (module_name, func_name) = format_pc(module, func);
                    if last_module != Some(module) {
                        path += "[module] ";
                        path += &module_name;
                        path += ";";
                        last_module = Some(module);
                    }
                    path += &func_name;
                    path += ";";
                }
                path.pop(); // remove trailing ';'
                writeln!(out, "{} {}", path, count)?;
            }
            out.flush()?;
        }
    }

    if opts.require_success && mach.get_status() != MachineStatus::Finished {
        eprintln!("Machine didn't finish: {}", mach.get_status().red());
        std::process::exit(1);
    }

    Ok(())
}
