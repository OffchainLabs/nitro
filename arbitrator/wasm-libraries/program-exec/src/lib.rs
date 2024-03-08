// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn program_continue(response: u32) -> u32;
    fn program_call_main(module: u32, args_len: usize) -> u32;
}

#[link(wasm_import_module = "program_internal")]
extern "C" {
    fn set_done(status: u32) -> u32;
    fn args_len(module: u32) -> usize;
}

// This module works with user-host
// It has the calls from the main (go) module which transfer
// control to a cothread.
//
// In any time, user-host module's stack may have multiple
// co-threads waiting inside it, due to co-threads making
// to launch a new stylus program (=new cothread). This is
// o.k. because these thread calls are FIFO.
// the main go-module is not FIFO - i.e. we return to go
// while a cothread is waiting for a response - so
// all go-calls come here

// request_ids start above 0x100
// return status are 1 byte, so they don't mix
// if we got a return status - notify user-host
// user-host will generate an "execution done" request
fn check_program_done(mut req_id: u32) -> u32 {
    if req_id < 0x100 {
        unsafe {
            req_id = set_done(req_id);
        }
    }
    req_id
}

/// starts the program (in jit waits for first request)
/// module MUST match last module number returned from new_program
/// returns request_id for the first request from the program
#[no_mangle]
pub unsafe extern "C" fn programs__start_program(module: u32) -> u32 {
    // call the program
    let args_len = args_len(module);
    check_program_done(program_call_main(module, args_len))
}

// sends previos response and transfers control to program
// MUST be called right after set_response to the same id
// returns request_id for the next request
#[no_mangle]
pub unsafe extern "C" fn programs__send_response(req_id: u32) -> u32 {
    // call the program
    check_program_done(program_continue(req_id))
}
