// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn program_continue(response: u32, ignored: u32) -> u32;
    fn program_call_main(module: u32, args_len: usize) -> u32;
}

#[link(wasm_import_module = "program_internal")]
extern "C" {
    fn set_done(status: u32) -> u32;
    fn args_len(module: u32) -> usize;
}


fn check_program_done(mut req_id: u32) -> u32 {
    if req_id < 0x100 {
        unsafe {
            req_id = set_done(req_id);
        }
    }
    req_id
}


#[no_mangle]
pub unsafe extern "C" fn programs__start_program(
    module: u32,
) -> u32 {
    // call the program
    let args_len = args_len(module);
    check_program_done(program_call_main(module, args_len))
}

#[no_mangle]
pub unsafe extern "C" fn programs__send_response(
    req_id: u32,
) -> u32 {
    // call the program
    check_program_done(program_continue(req_id, 0))
}
