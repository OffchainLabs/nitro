// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{JsEnv, JsState, JsValue};
use arbutil::evm::{api::EvmApiMethod, js::ApiValue};
use eyre::{bail, ErrReport, Result, WrapErr};

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

impl JsState {
    pub fn call_stylus_func(
        &self,
        api_id: u32,
        method: EvmApiMethod,
        args: Vec<ApiValue>,
        js_env: &mut dyn JsEnv,
    ) -> Result<Vec<ApiValue>> {
        let field = &format!("api{api_id}");
        let api = self.get_globals().get_path(&["stylus", field]);

        // get the callback into Go
        let array = match api.clone() {
            JsValue::Array(array) => array,
            x => bail!("bad EVM api type for {api_id}: {x:?}"),
        };
        let array = array.lock();
        let func = match array.get(method as usize) {
            Some(JsValue::Function(func)) => func,
            x => bail!("bad EVM api func for {method:?}, {api_id}: {x:?}"),
        };

        // call into go
        let args = args.into_iter().map(Into::into).collect();
        let outs = func.call(js_env, api, args).wrap_err("EVM api failed")?;

        // send the outputs
        let outs = match outs {
            JsValue::Array(outs) => outs.lock().clone().into_iter(),
            x => bail!("bad EVM api result for {method:?}: {x:?}"),
        };
        outs.map(TryInto::try_into).collect::<Result<_, _>>()
    }
}
