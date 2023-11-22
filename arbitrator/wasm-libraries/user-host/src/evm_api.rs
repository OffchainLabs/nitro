// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::{
    api::EvmApiMethod,
    js::{ApiValue, JsCallIntoGo},
};

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn wavm_set_error_policy(status: u32);
}

#[link(wasm_import_module = "go_stub")]
extern "C" {
    fn run_api_closure(
        api_id: u32,
        method: EvmApiMethod,
        data: *const *const u8,
        lens: *const usize,
        num_args: usize,
    ) -> usize;
    fn read_api_result_lens(lens: *mut usize);
    fn move_api_result_data(data: *const *mut u8);
}

pub(crate) struct ApiCaller {
    api_id: u32,
}

impl ApiCaller {
    pub fn new(api_id: u32) -> Self {
        Self { api_id }
    }
}

impl JsCallIntoGo for ApiCaller {
    fn call_go(&mut self, method: EvmApiMethod, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let mut data = vec![];
        let mut lens = vec![];
        for arg in &args {
            data.push(arg.0.as_ptr());
            lens.push(arg.0.len());
        }

        let api_id = self.api_id;
        unsafe {
            wavm_set_error_policy(0); // disable error recovery

            let count = run_api_closure(api_id, method, data.as_ptr(), lens.as_ptr(), args.len());
            let mut lens = vec![0_usize; count];
            read_api_result_lens(lens.as_mut_ptr());

            let mut outs: Vec<Vec<u8>> = lens.into_iter().map(|x| vec![0; x]).collect();
            let data: Vec<_> = outs.iter_mut().map(Vec::as_mut_ptr).collect();
            move_api_result_data(data.as_ptr());

            let outs = outs.into_iter().map(ApiValue).collect();
            wavm_set_error_policy(1); // re-enable error recovery
            outs
        }
    }
}
