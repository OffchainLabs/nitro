//! Ideally, we should just use arbitrator/prover. This crate has no
//! reason to exist. But at the moment, nitro uses wasmer 4.x, while SP1
//! requires wasmer 6.1.x. The 2 wasmer versions are hardly compatible.
//! On the other hand, we would want to borrow common definitions in
//! arbitrator/prover so we don't have to define them twice.
//! This crate serves as a workaround till we can upgrade wasmer in nitro.
#![allow(unexpected_cfgs)]

#[cfg(feature = "native")]
#[path = "../../../arbitrator/prover/src/binary.rs"]
pub mod binary;
#[cfg(feature = "native")]
#[path = "../../../arbitrator/prover/src/programs/mod.rs"]
pub mod programs;
#[cfg(feature = "native")]
#[path = "../../../arbitrator/prover/src/value.rs"]
pub mod value;

#[cfg(feature = "native")]
#[path = "../../../arbitrator/arbutil/src/operator.rs"]
pub mod operator;

pub mod binary_input;
