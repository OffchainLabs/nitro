// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::color::Color;
use std::fmt::Display;

#[must_use]
pub fn commas<T, U>(items: U) -> String
where
    T: Display,
    U: IntoIterator<Item = T>,
{
    let items: Vec<_> = items.into_iter().map(|x| format!("{x}")).collect();
    items.join(&", ".grey())
}
