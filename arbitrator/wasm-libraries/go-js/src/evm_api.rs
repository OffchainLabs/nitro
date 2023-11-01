// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::JsValue;
use arbutil::evm::js::ApiValue;
use eyre::{bail, ErrReport};

impl From<ApiValue> for JsValue {
    fn from(value: ApiValue) -> Self {
        Self::new_uint8_array(value.0)
    }
}

impl TryFrom<JsValue> for ApiValue {
    type Error = ErrReport;

    fn try_from(value: JsValue) -> Result<Self, Self::Error> {
        match value {
            JsValue::Uint8Array(x) => Ok(ApiValue(x.lock().to_vec())),
            x => bail!("tried to make EVM API value from {x:?}"),
        }
    }
}
