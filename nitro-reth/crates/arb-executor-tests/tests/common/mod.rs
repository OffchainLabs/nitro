//! Shared helpers for the precompile account-read access tests.

use alloy_primitives::Address;

/// Runtime that calls `precompile` with `selector` + the left-padded `read`
/// address (forwarding `msg.value` when `forward_value`), then makes an empty
/// `CALL` to `call`.
pub fn read_then_call(
    precompile: u8,
    selector: [u8; 4],
    read: Address,
    call: Address,
    forward_value: bool,
) -> Vec<u8> {
    let mut c = Vec::new();
    // mem[0..4] = selector, mem[4..36] = read address (left-padded)
    c.push(0x63);
    c.extend_from_slice(&selector);
    c.extend_from_slice(&[0x60, 0xE0, 0x1b, 0x60, 0x00, 0x52]); // PUSH1 224; SHL; PUSH1 0; MSTORE
    c.push(0x73);
    c.extend_from_slice(read.as_slice());
    c.extend_from_slice(&[0x60, 0x04, 0x52]); // PUSH1 4; MSTORE

    // CALL precompile: retLen, retOff, argLen=36, argOff pushed in reverse order.
    c.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x24, 0x60, 0x00]);
    if forward_value {
        c.push(0x34); // CALLVALUE
    } else {
        c.extend_from_slice(&[0x60, 0x00]); // PUSH1 0
    }
    c.extend_from_slice(&[0x60, precompile, 0x5a, 0xf1, 0x50]); // precompile; GAS; CALL; POP

    // CALL target: empty calldata, no value.
    c.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00]);
    c.push(0x73);
    c.extend_from_slice(call.as_slice());
    c.extend_from_slice(&[0x5a, 0xf1, 0x50]); // GAS; CALL; POP
    c.push(0x00); // STOP
    c
}
