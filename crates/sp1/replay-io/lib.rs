// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Host-guest input protocol for the SP1 replay pipeline. The guest reads, in order: (1) the
//! wasmu, raw; (2) the function names JSON, raw — both baked in by bootloading — then (3) the
//! validation payload: the bootload sentinel or the validation input.

const SP1_BOOTLOAD_SENTINEL: &[u8] = b"SP1_BOOTLOAD_ONLY";

/// Sending data from host to SP1 guest.
#[cfg(not(target_os = "zkvm"))]
pub mod send {
    use sp1_core_executor::{ExecutionMode, MinimalExecutor};

    /// Sends input for bootloading.
    pub fn bootload_mode(
        executor: &mut MinimalExecutor<impl ExecutionMode>,
        wasmu: &[u8],
        function_names_json: &str,
    ) {
        executor.with_input(wasmu);
        executor.with_input(function_names_json.as_bytes());
        executor.with_input(super::SP1_BOOTLOAD_SENTINEL);
    }

    /// Sends the data for actual validation (already after bootloading).
    pub fn validation_mode(executor: &mut MinimalExecutor<impl ExecutionMode>, payload: &[u8]) {
        executor.with_input(payload);
    }
}

/// Reading data in SP1 guest.
pub mod recv {
    /// The inputs consumed during bootload.
    pub struct BootloadInputs {
        pub wasmu: &'static [u8],
        pub function_names_json: &'static [u8],
    }

    /// Reads the bootload inputs.
    pub fn bootload_inputs() -> BootloadInputs {
        BootloadInputs {
            wasmu: read_raw(),
            function_names_json: read_raw(),
        }
    }

    /// The guest's third input, with the sentinel already recognized.
    pub enum ValidationPayload {
        /// Bootloading: the guest should halt cleanly; the ELF dump is the product.
        Bootload,
        /// A real validation input.
        Input(&'static [u8]),
    }

    /// Reads the validation payload (either a sentinel or actual input).
    pub fn validation_payload() -> ValidationPayload {
        let payload = read_raw();
        if payload == super::SP1_BOOTLOAD_SENTINEL {
            ValidationPayload::Bootload
        } else {
            ValidationPayload::Input(payload)
        }
    }

    /// Reads an input in place, without copying.
    fn read_raw() -> &'static [u8] {
        let sp1_zkvm::ReadVecResult { ptr, len, .. } = sp1_zkvm::read_vec_raw();
        assert!(!ptr.is_null());
        // SAFETY: the executor wrote `len` bytes at `ptr`; never deallocated.
        unsafe { core::slice::from_raw_parts(ptr, len) }
    }
}
