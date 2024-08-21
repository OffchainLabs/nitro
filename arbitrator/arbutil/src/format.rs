// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::color::{Color, GREY, MINT, RED, YELLOW};
use std::{
    fmt::{Debug, Display},
    time::Duration,
};

#[must_use]
pub fn time(span: Duration) -> String {
    let mut span = span.as_nanos() as f64;
    let mut unit = 0;
    let units = [
        "ns", "μs", "ms", "s", "min", "h", "d", "w", "mo", "yr", "dec", "cent", "mill", "eon",
    ];
    let scale = [
        1000., 1000., 1000., 60., 60., 24., 7., 4.34, 12., 10., 10., 10., 1_000_000.,
    ];
    let colors = [MINT, MINT, YELLOW, RED, RED, RED];
    while span >= scale[unit] && unit < scale.len() {
        span /= scale[unit];
        unit += 1;
    }
    format!("{:6}", format!("{:.1}{}", span, units[unit])).color(colors[unit])
}

#[must_use]
pub fn gas(gas: u64) -> String {
    let mut gas = gas as f64;
    let mut unit = 0;
    let units = vec!["", "k", "m", "b"];
    let scale = [1000., 1000., 1000.];
    let colors = [MINT, MINT, YELLOW, RED];
    while gas >= scale[unit] && unit < scale.len() {
        gas /= scale[unit];
        unit += 1;
    }
    format!("{:6}", format!("{:.1}{}", gas, units[unit])).color(colors[unit])
}

#[must_use]
pub fn wei(wei: u64) -> String {
    let mut value = wei as f64;
    let mut unit = 0;
    let units = [
        "wei", "kwei", "mwei", "gwei", "szabo", "finney", "Ξ", "kΞ", "mΞ", "gΞ", "tΞ",
    ];
    let scale = [
        1000., 1000., 1000., 1000., 1000., 1000., 1000., 1000., 1000., 1000., 1000.,
    ];
    let colors = [
        MINT, MINT, MINT, MINT, MINT, MINT, YELLOW, YELLOW, RED, RED, RED, RED,
    ];
    while value >= scale[unit] && unit < scale.len() {
        value /= scale[unit];
        unit += 1;
    }
    format!("{:6}", format!("{:.1}{}", value, units[unit])).color(colors[unit])
}

#[must_use]
pub fn bytes(bytes: usize) -> String {
    let mut space = bytes as f64;
    let mut unit = 0;
    let units = vec!["", "kb", "mb", "gb"];
    let scale = [1024., 1024., 1024.];
    let colors = [GREY, GREY, RED];
    while space >= scale[unit] && unit < scale.len() {
        space /= scale[unit];
        unit += 1;
    }
    format!("{:6}", format!("{:.1}{}", space, units[unit])).color(colors[unit])
}

#[must_use]
pub fn commas<T, U>(items: U) -> String
where
    T: Display,
    U: IntoIterator<Item = T>,
{
    let items: Vec<_> = items.into_iter().map(|x| format!("{x}")).collect();
    items.join(&", ".grey())
}

pub fn hex_fmt<T: AsRef<[u8]>>(data: T, f: &mut std::fmt::Formatter) -> std::fmt::Result {
    f.write_str(&hex::encode(data))
}

pub trait DebugBytes {
    fn debug_bytes(self) -> Vec<u8>;
}

impl<T: Debug> DebugBytes for T {
    fn debug_bytes(self) -> Vec<u8> {
        format!("{:?}", self).as_bytes().to_vec()
    }
}

pub trait Utf8OrHex {
    fn from_utf8_or_hex(data: impl Into<Vec<u8>>) -> String;
}

impl Utf8OrHex for String {
    fn from_utf8_or_hex(data: impl Into<Vec<u8>>) -> String {
        match String::from_utf8(data.into()) {
            Ok(string) => string,
            Err(error) => hex::encode(error.as_bytes()),
        }
    }
}
