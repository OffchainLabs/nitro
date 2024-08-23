// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use activate::{util, Trial};
use arbutil::{color::Color, format};
use eyre::Result;
use std::{
    fs::File,
    io::{BufRead, BufReader},
    path::Path,
    time::Duration,
};

pub fn verify(path: &Path) -> Result<()> {
    verify_impl(path, "parse", true, |trial| {
        let pred = trial.pred_parse_us() as f64;
        let actual = trial.parse_time.as_micros() as f64;
        (pred, actual)
    })?;

    verify_impl(path, "module", true, |trial| {
        let pred = trial.pred_module_us() as f64;
        let actual = trial.module_time.as_micros() as f64;
        (pred, actual)
    })?;

    verify_impl(path, "asm", true, |trial| {
        let pred = trial.pred_asm_us() as f64;
        let actual = trial.asm_time.as_micros() as f64;
        (pred, actual)
    })?;

    verify_impl(path, "brotli", true, |trial| {
        let pred = trial.pred_brotli_us() as f64;
        let actual = trial.brotli_time.as_micros() as f64;
        (pred, actual)
    })?;

    verify_impl(path, "hash", true, |trial| {
        let pred = trial.pred_hash_us() as f64;
        let actual = trial.hash_time.as_micros() as f64;
        (pred, actual)
    })?;

    verify_impl(path, "disk", false, |trial| {
        let pred = trial.pred_asm_len() as f64;
        let actual = trial.asm_len as f64;
        (pred, actual)
    })
}

pub fn verify_impl(path: &Path, name: &str, gas: bool, apply: fn(Trial) -> (f64, f64)) -> Result<()> {
    let file = BufReader::new(File::open(path)?);

    let mut high: f64 = f64::MIN;
    let mut avg: f64 = 0.;
    let mut low: f64 = f64::MAX;
    let mut count = 0;

    let mut naive: u64 = 0;
    let mut model: u64 = 0;
    let mut load: u64 = 0;

    for line in file.lines() {
        let trial: Trial = line?.parse()?;

        let (pred, actual) = apply(trial);

        high = high.max(actual - pred);
        low = low.min(actual - pred);
        avg += 100. * (actual - pred) / actual;
        count += 1;

        naive = naive.max(actual as u64);
        model += pred as u64;
        load += actual as u64;

        if actual > pred {
            println!("pred {} {}", actual.red(), pred.red());
        }
    }

    avg = avg / count as f64;
    model = model / count;
    load = load / count;

    println!("{name} prediction");
    println!("high:  {high}");
    println!("low:   {low}");
    println!("avg:   {:.1}%", avg);
    println!("count: {count}");

    let print = |x| match gas {
        true => format::gas(util::gas(Duration::from_micros(x), 2.)),
        false => {
            let wei = 50. * x as f64 * 0.1 * 1e9;
            let usd = 2300. * wei / 1e18;
            format!("${usd:.2}")
        },
    };
    
    println!(
        "naive: {}", print(naive)
    );
    println!(
        "model: {}", print(model));

    if model < naive {
        println!(
            "saved: {} ^.^", print(naive - model)
        );
    }
    println!(
        "oppt:  {}", print(model - load)
    );
    println!();
    Ok(())
}
