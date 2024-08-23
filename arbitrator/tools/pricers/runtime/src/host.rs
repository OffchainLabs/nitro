// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::time::{Duration, Instant};
use stylus::env::MeterData;
use thiserror::Error;
use wasmer::FunctionEnvMut;

#[derive(Error, Debug)]
pub enum Escape {
    #[error("done")]
    Done,
    #[error("incomplete")]
    Incomplete,
}

pub type MaybeEscape = Result<(), Escape>;

pub type PricerEnvMut<'a> = FunctionEnvMut<'a, PricerEnv>;

#[derive(Debug, Default)]
pub struct PricerEnv {
    pub timer: Option<Instant>,
    pub elapsed: Option<Duration>,
    pub cycles_start: Option<u64>,
    pub cycles_total: Option<u64>,
    pub meter: Option<MeterData>,
}

impl PricerEnv {
    pub fn meter(&self) -> &MeterData {
        self.meter.as_ref().expect("not metered")
    }

    pub fn meter_mut(&mut self) -> &mut MeterData {
        self.meter.as_mut().expect("not metered")
    }
}

pub fn toggle_timer(mut env: PricerEnvMut) -> MaybeEscape {
    let env = env.data_mut();
    if let Some(timer) = env.timer {
        env.elapsed = Some(timer.elapsed());
        env.cycles_total = Some(cpu_cycles().wrapping_sub(env.cycles_start.unwrap()));
        return Err(Escape::Done);
    }
    env.timer = Some(Instant::now());
    env.cycles_start = Some(cpu_cycles());
    Ok(())
}

pub fn memory_grow(env: PricerEnvMut, _: u32) -> MaybeEscape {
    toggle_timer(env)
}

#[inline(always)]
pub fn cpu_cycles() -> u64 {
    #[cfg(target_arch = "x86_64")]
    unsafe { core::arch::x86_64::_rdtsc() }

    #[cfg(not(target_arch = "x86_64"))]
    0
}
