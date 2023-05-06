// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::js::{ApiValue, JsCallIntoGo};

#[link(wasm_import_module = "go_stub")]
extern "C" {
    fn run_stylus_closure(
        func: u32,
        data: *const *const u8,
        lens: *const usize,
        count: usize,
    ) -> usize;
    fn read_closure_lens(func: u32, lens: *mut usize);
    fn drop_closure_outs(func: u32, data: *const *mut u8);
}

pub(crate) struct ApiCaller {}

impl ApiCaller {
    pub fn new() -> Self {
        Self {}
    }
}

impl JsCallIntoGo for ApiCaller {
    fn call_go(&mut self, func: u32, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let mut data = vec![];
        let mut lens = vec![];
        for arg in &args {
            data.push(arg.0.as_ptr());
            lens.push(arg.0.len());
        }

        unsafe {
            let count = run_stylus_closure(func, data.as_ptr(), lens.as_ptr(), args.len());
            let mut lens = vec![0_usize; count];
            read_closure_lens(func, lens.as_mut_ptr());

            let mut outs: Vec<Vec<u8>> = lens.into_iter().map(|x| vec![0; x]).collect();
            let data: Vec<_> = outs.iter_mut().map(Vec::as_mut_ptr).collect();
            drop_closure_outs(func, data.as_ptr());

            outs.into_iter().map(ApiValue).collect()
        }
    }
}
