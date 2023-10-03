// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::redundant_closure_call)]

use crate::{env::Escape, native::NativeInstance};
use arbutil::evm::api::EvmApi;
use arbutil::evm::user::UserOutcome;
use eyre::{eyre, Result};
use prover::machine::Machine;
use prover::programs::{prelude::*, STYLUS_ENTRY_POINT};

pub trait RunProgram {
    fn run_main(&mut self, args: &[u8], config: StylusConfig, ink: u64) -> Result<UserOutcome>;
}

impl RunProgram for Machine {
    fn run_main(&mut self, args: &[u8], config: StylusConfig, ink: u64) -> Result<UserOutcome> {
        macro_rules! call {
            ($module:expr, $func:expr, $args:expr) => {
                call!($module, $func, $args, |error| UserOutcome::Failure(error))
            };
            ($module:expr, $func:expr, $args:expr, $error:expr) => {{
                match self.call_function($module, $func, $args) {
                    Ok(value) => value[0].try_into().unwrap(),
                    Err(error) => return Ok($error(error)),
                }
            }};
        }

        // push the args
        let args_len = (args.len() as u32).into();
        let push_vec = vec![
            args_len,
            config.version.into(),
            config.max_depth.into(),
            config.pricing.ink_price.into(),
        ];
        let args_ptr = call!("user_test", "prepare", push_vec);
        let user_host = self.find_module("user_test")?;
        self.write_memory(user_host, args_ptr, args)?;

        self.set_ink(ink);
        self.set_stack(config.max_depth);

        let status: u32 = call!("user", STYLUS_ENTRY_POINT, vec![args_len], |error| {
            if self.ink_left() == MachineMeter::Exhausted {
                return UserOutcome::OutOfInk;
            }
            if self.stack_left() == 0 {
                return UserOutcome::OutOfStack;
            }
            UserOutcome::Failure(error)
        });

        let outs_ptr = call!("user_test", "get_outs_ptr", vec![]);
        let outs_len = call!("user_test", "get_outs_len", vec![]);
        let outs = self.read_memory(user_host, outs_ptr, outs_len)?.to_vec();

        Ok(match status {
            0 => UserOutcome::Success(outs),
            _ => UserOutcome::Revert(outs),
        })
    }
}

impl<E: EvmApi> RunProgram for NativeInstance<E> {
    fn run_main(&mut self, args: &[u8], config: StylusConfig, ink: u64) -> Result<UserOutcome> {
        use UserOutcome::*;

        self.set_ink(ink);
        self.set_stack(config.max_depth);

        let store = &mut self.store;
        let env = self.env.as_mut(store);
        env.args = args.to_owned();
        env.outs.clear();
        env.config = Some(config);

        if env.evm_data.tracing {
            let args_len = args.len() as u32;
            env.evm_api
                .capture_hostio(STYLUS_ENTRY_POINT, &args_len.to_be_bytes(), &[], ink, ink);
        }

        let exports = &self.instance.exports;
        let main = exports.get_typed_function::<u32, u32>(store, STYLUS_ENTRY_POINT)?;
        let status = match main.call(store, args.len() as u32) {
            Ok(status) => status,
            Err(outcome) => {
                if self.stack_left() == 0 {
                    return Ok(OutOfStack);
                }
                if self.ink_left() == MachineMeter::Exhausted {
                    return Ok(OutOfInk);
                }

                let escape: Escape = match outcome.downcast() {
                    Ok(escape) => escape,
                    Err(error) => return Ok(Failure(eyre!(error).wrap_err("hard user error"))),
                };
                return Ok(match escape {
                    Escape::OutOfInk => OutOfInk,
                    Escape::Memory(error) => UserOutcome::Failure(error.into()),
                    Escape::Internal(error) | Escape::Logical(error) => UserOutcome::Failure(error),
                });
            }
        };

        let env = self.env();
        if env.evm_data.tracing {
            env.evm_api
                .capture_hostio("user_returned", &[], &status.to_be_bytes(), ink, ink);
        }

        let outs = self.env().outs.clone();
        Ok(match status {
            0 => UserOutcome::Success(outs),
            _ => UserOutcome::Revert(outs),
        })
    }
}
