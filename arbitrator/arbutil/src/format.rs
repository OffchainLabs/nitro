// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::color::Color;
use std::{
    fmt::{Debug, Display},
    time::Duration,
};

#[must_use]
pub fn time(span: Duration) -> String {
    use crate::color::{MINT, RED, YELLOW};

    let mut span = span.as_nanos() as f64;
    let mut unit = 0;
    let units = vec![
        "ns", "μs", "ms", "s", "min", "h", "d", "w", "mo", "yr", "dec", "cent", "mill", "eon",
    ];
    let scale = vec![
        1000., 1000., 1000., 60., 60., 24., 7., 4.34, 12., 10., 10., 10., 1_000_000.,
    ];
    let colors = vec![MINT, MINT, YELLOW, RED, RED, RED];
    while span >= scale[unit] && unit < scale.len() {
        span /= scale[unit];
        unit += 1;
    }
    format!("{:6}", format!("{:.1}{}", span, units[unit])).color(colors[unit])
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
