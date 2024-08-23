/*// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{host::Escape, runtime::Runtime, wasm};
use arbutil::{format, operator::OperatorCode};
use eyre::{bail, ErrReport, Result};
use prover::programs::prelude::{CompileConfig, CountingMachine, StylusConfig};
use rand::Rng;
use std::{
    fmt::Display,
    fs::File,
    io::{BufRead, BufReader, Seek, SeekFrom},
    path::Path,
    str::FromStr,
    time::Duration,
};

pub const OP_COUNT: usize = 256;

type Ops = [usize; OP_COUNT];

pub struct Trial {
    pub time: Duration,
    pub cycles: u64,
    pub ops: Ops,
}

impl Trial {
    pub fn try_new(config: StylusConfig, ink: u64) -> Result<Self> {
        let mut rng = rand::thread_rng();
        let module = wasm::random(rng.gen_range(0..1024))?;
        wasm::validate(&module)?;

        let mut compile = CompileConfig::version(1, true);
        compile.debug.count_ops = true;

        let mut runtime = Runtime::new(&module, compile)?;
        match runtime.run(config, ink)? {
            Escape::Incomplete => bail!("incomplete"),
            Escape::Done => {}
        };

        let mut ops = [0; OP_COUNT];
        for (op, count) in runtime.operator_counts()? {
            ops[op.seq()] = count;
        }
        if ops.iter().sum::<usize>() < 10000 {
            bail!("too few ops")
        }
        let (time, cycles) = runtime.time();
        Ok(Self { time, ops, cycles })
    }

    pub fn sample(config: StylusConfig, ink: u64) -> Result<Self> {
        for i in 0.. {
            let error = match Self::try_new(config, ink) {
                Ok(trial) => return Ok(trial),
                Err(error) => error,
            };
            if i > 10_000 {
                bail!("failed to generate trial: {error}")
            }
        }
        unreachable!()
    }

    #[allow(dead_code)]
    pub fn short_stats(&self) -> String {
        let sum: usize = self.ops.into_iter().sum();
        let time = format::time(self.time);
        let per = self.time.as_nanos() as f64 / sum as f64;
        format!("{time} {sum} => {per:.3}ns")
    }

    #[allow(dead_code)]
    pub fn long_stats(&self) {
        println!("Trial: {}", self.short_stats());
        let mut top: Vec<_> = self.ops.into_iter().enumerate().collect();
        top.sort_by_key(|(_, x)| *x);
        top.reverse();
        for (seq, count) in top.iter().take(16) {
            let name = format!("{}", OperatorCode::from_seq(*seq));
            println!("{name:<13} {count}");
        }
    }
}

impl Display for Trial {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} {}", self.time.as_nanos(), self.cycles)?;
        for (seq, count) in self.ops.iter().enumerate() {
            if *count != 0 {
                write!(f, " {seq} {count}")?;
            }
        }
        Ok(())
    }
}

impl FromStr for Trial {
    type Err = ErrReport;

    fn from_str(s: &str) -> std::result::Result<Self, Self::Err> {
        let data: Vec<_> = s.trim().split(' ').collect();
        let mut ops = [0; OP_COUNT];

        let time = Duration::from_nanos(data[0].parse()?);
        let cycles = data[1].parse()?;
        for i in (2..data.len() - 1).step_by(2) {
            let seq: usize = data[i].parse()?;
            ops[seq] = data[i + 1].parse()?;
        }
        Ok(Self { time, ops, cycles })
    }
}

pub struct Feed {
    file: BufReader<File>,
}

impl Feed {
    pub fn new(path: &Path) -> Result<Self> {
        let file = BufReader::new(File::open(path)?);
        Ok(Self { file })
    }

    pub fn next(&mut self) -> Result<Trial> {
        let mut line = String::default();
        if let Ok(0) = self.file.read_line(&mut line) {
            // if EOF
            line.clear();
            self.file.seek(SeekFrom::Start(0))?;
            self.file.read_line(&mut line)?; // try once more
        }
        line.parse()
    }
}
*/
