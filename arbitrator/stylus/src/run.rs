// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::api::EvmApi;
use crate::{env::Escape, native::NativeInstance};
use eyre::{ensure, eyre, Result};
use prover::machine::Machine;
use prover::programs::{prelude::*, STYLUS_ENTRY_POINT, USER_HOST};

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
            config.pricing.hostio_ink.into(),
        ];
        let args_ptr = call!("user_host", "push_program", push_vec);
        let user_host = self.find_module(USER_HOST)?;
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

        let outs_ptr = call!(USER_HOST, "get_output_ptr", vec![]);
        let outs_len = call!(USER_HOST, "get_output_len", vec![]);
        let outs = self.read_memory(user_host, outs_ptr, outs_len)?.to_vec();

        let num_progs: u32 = call!(USER_HOST, "pop_program", vec![]);
        ensure!(num_progs == 0, "dirty user_host");

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
        let mut env = self.env.as_mut(store);
        env.args = args.to_owned();
        env.outs.clear();
        env.config = Some(config);

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
                    Escape::Memory(error) => UserOutcome::revert(error.into()),
                    Escape::Internal(error) | Escape::Logical(error) => UserOutcome::revert(error),
                });
            }
        };

        let outs = self.env().outs.clone();
        Ok(match status {
            0 => UserOutcome::Success(outs),
            _ => UserOutcome::Revert(outs),
        })
    }
}
