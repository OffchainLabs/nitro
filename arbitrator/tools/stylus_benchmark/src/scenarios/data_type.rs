// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use strum_macros::{Display, EnumString};
use strum;

#[derive(Debug, Display, EnumString)]
#[strum(serialize_all = "lowercase")]
pub enum DataType {
    I32,
    I64
}
