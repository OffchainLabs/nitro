// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::js::{ApiValue, JsCallIntoGo};

pub(crate) struct ApiCaller {}

impl ApiCaller {
    pub fn new() -> Self {
        Self {}
    }
}

impl JsCallIntoGo for ApiCaller {
    fn call_go(&mut self, func: u32, args: Vec<ApiValue>) -> Vec<ApiValue> {
        todo!()
    }
}
