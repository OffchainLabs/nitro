// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::{color::Color, evm::EvmData, format};
use eyre::{bail, ErrReport, Result};
use prover::{binary::WasmBinary, machine::Module, programs::prelude::CompileConfig};
use serde::{Deserialize, Serialize};
use std::{
    str::FromStr,
    time::{Duration, Instant},
};
use stylus::native::NativeInstance;
use util::FakeEvm;

pub mod util;
pub mod wasm;

#[derive(Serialize, Deserialize)]
pub struct Trial {
    pub parse_time: Duration,
    pub module_time: Duration,
    pub hash_time: Duration,
    pub brotli_time: Duration,
    pub asm_time: Duration,
    pub init_time: Duration,
    pub wasm_len: usize,
    pub module_len: usize,
    pub brotli_len: usize,
    pub asm_len: usize,
    pub info: WasmInfo,
}

#[derive(Serialize, Deserialize)]
pub struct WasmInfo {
    pub funcs: usize,
    pub code: usize,
    pub data: usize,
    pub mem_size: usize,
}

impl WasmInfo {
    pub fn new(bin: &WasmBinary) -> Self {
        let code = bin.codes.iter().map(|x| x.expr.len()).sum();
        let data = bin.datas.iter().map(|x| x.data.len()).sum();
        let mem_size = bin.memories[0].initial as usize;
        Self {
            funcs: bin.codes.len(),
            code,
            data,
            mem_size,
        }
    }
}

impl Trial {
    pub fn new(wasm: &[u8]) -> Result<Trial> {
        let mut compile = CompileConfig::version(1, false);
        compile.debug.count_ops = false;

        // parse the binary
        let timer = Instant::now();
        let (bin, stylus_data, _) = WasmBinary::parse_user(&wasm, 128, &compile)?;
        let parse_time = timer.elapsed();

        // create a module
        let timer = Instant::now();
        let module = Module::from_user_binary(&bin, false, Some(stylus_data))?;
        let module_time = timer.elapsed();

        // hash the module
        let timer = Instant::now();
        module.hash();
        let hash_time = timer.elapsed();

        // compress the module (only done by validators)
        let timer = Instant::now();
        let module = module.into_bytes();
        let brotli = util::compress(&module, 2)?;
        let brotli_time = timer.elapsed();

        // create an asm
        let timer = Instant::now();
        let asm = stylus::native::module(&wasm, compile.clone()).expect("asm divergence");
        let asm_time = timer.elapsed();

        // init an asm
        let timer = Instant::now();
        unsafe {
            let evm = FakeEvm;
            let evm_data = EvmData::default();
            NativeInstance::deserialize(&asm, compile, evm, evm_data).expect("can't init");
        }
        let init_time = timer.elapsed();

        Ok(Trial {
            parse_time,
            module_time,
            hash_time,
            brotli_time,
            asm_time,
            init_time,
            wasm_len: wasm.len(),
            module_len: module.len(),
            brotli_len: brotli.len(),
            asm_len: asm.len(),
            info: WasmInfo::new(&bin),
        })
    }

    pub fn check_model(&self, margin: f64) -> Result<()> {
        macro_rules! check {
            ($name:expr, $pred:ident, $actual:ident) => {
                let pred = self.$pred() as f64;
                let actual = self.$actual.as_micros() as f64;
                if pred * margin < actual {
                    let pred = format!("{:.0}", pred * margin).red();
                    bail!("missed {}, {} > {pred}", $name, actual.red());
                }
            };
        }

        check!("parse", pred_parse_us, parse_time);
        check!("module", pred_module_us, module_time);
        check!("hash", pred_hash_us, hash_time);
        check!("brotli", pred_brotli_us, brotli_time);
        check!("asm", pred_asm_us, asm_time);
        Ok(())
    }

    pub fn accuracy(&self) -> (&str, i64) {
        let mut diff: i64 = i64::MIN;
        let mut name = "";

        macro_rules! check {
            ($name:expr, $pred:ident, $actual:ident) => {
                let pred = self.$pred() as i64;
                let actual = self.$actual.as_micros() as i64;
                let specific = actual as i64 - pred as i64;
                if specific > diff {
                    diff = specific;
                    name = $name;
                }
            };
        }

        check!("parse", pred_parse_us, parse_time);
        check!("module", pred_module_us, module_time);
        check!("hash", pred_hash_us, hash_time);
        check!("brotli", pred_brotli_us, brotli_time);
        check!("asm", pred_asm_us, asm_time);
        (name, diff)
    }

    pub fn pred_parse_us(&self) -> u64 {
        let s = 17656.;
        let l = 0.03573263;
        let pred = s + l * self.wasm_len as f64;
        pred.ceil() as u64
    }

    pub fn pred_module_us(&self) -> u64 {
        let info = &self.info;
        let s = 11000.;
        let f = 11.57;
        let c = 0.222;
        let l = 25000. / (256. * 1024.);

        let pred = s + f * info.funcs as f64 + c * info.code as f64 + l * self.wasm_len as f64;
        pred.ceil() as u64
    }

    pub fn pred_asm_us(&self) -> u64 {
        let info = &self.info;
        let s = 15000.;
        let f = 5.693;
        let c = 0.313;
        let d = 8.5;
        let l = 0.04;

        let pred = s
            + f * info.funcs as f64
            + c * info.code as f64
            + d * info.data as f64
            + l * self.wasm_len as f64;
        pred.ceil() as u64
    }

    pub fn pred_brotli_us(&self) -> u64 {
        let s = 14599.75;
        let l = 0.05746767;
        let pred = s + l * self.wasm_len as f64;
        pred.ceil() as u64
    }

    pub fn pred_hash_us(&self) -> u64 {
        let info = &self.info;
        let s = 1000.;
        let d = 8.87621;
        let l = 0.0887621;
        let m = 2216.903;

        let pred = s + m * info.mem_size as f64 + d * info.data as f64 + l * self.wasm_len as f64;
        pred.ceil() as u64
    }
    
    pub fn pred_asm_len(&self) -> u64 {
        let info = &self.info;
        let s = 393216.;
        let m = 1.679;
        let f = 432.32;
        let l = 3.952;

        let pred = s + m * self.module_len as f64 + f * info.funcs as f64 + l * self.wasm_len as f64;
        pred.ceil() as u64
    }

    pub fn print(&self) {
        macro_rules! stat {
            ($name:expr, $time:expr, $margin:expr) => {
                let gas = util::gas($time, $margin);
                println!("{} {}  {}", $name, format::time($time), format::gas(gas));
            };
        }

        stat!("parse ", self.parse_time, 2.);
        stat!("module", self.module_time, 2.);
        stat!("hash  ", self.hash_time, 3.);
        stat!("brotli", self.brotli_time, 3.);
        stat!("asm   ", self.asm_time, 2.);
        stat!("init  ", self.init_time, 2.);

        //println!("------ {}  {:.1} {:.1} {:.1} {:.1}", format::bytes(wasm.len()), parse_ratio, module_ratio, hash_ratio, brotli_ratio);
        println!()
    }

    pub fn to_hex(&self) -> Result<String> {
        Ok(hex::encode(bincode::serialize(self)?))
    }
}

impl FromStr for Trial {
    type Err = ErrReport;

    fn from_str(s: &str) -> Result<Self> {
        Ok(bincode::deserialize(&hex::decode(s)?)?)
    }
}
