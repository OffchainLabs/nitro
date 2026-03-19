use arbos::programs::types::UserOutcome;
use eyre::Result;

use crate::{
    config::StylusConfig,
    error::Escape,
    evm_api::EvmApi,
    ink::Ink,
    meter::{DepthCheckedMachine, MachineMeter, MeteredMachine, STYLUS_ENTRY_POINT},
    native::NativeInstance,
};

/// Trait for running Stylus WASM programs.
pub trait RunProgram {
    fn run_main(&mut self, args: &[u8], config: StylusConfig, ink: Ink) -> Result<UserOutcome>;
}

impl<E: EvmApi> RunProgram for NativeInstance<E> {
    fn run_main(&mut self, args: &[u8], config: StylusConfig, ink: Ink) -> Result<UserOutcome> {
        self.set_ink(ink);
        self.set_stack(config.max_depth);

        {
            let store = &mut self.store;
            let env = self.env.as_mut(store);
            env.args = args.to_owned();
            env.outs.clear();
            env.config = Some(config);

            if env.evm_data.tracing {
                let args_len = args.len() as u32;
                env.evm_api.capture_hostio(
                    STYLUS_ENTRY_POINT,
                    &args_len.to_be_bytes(),
                    &[],
                    ink,
                    ink,
                );
            }
        }

        self.sync_meter_to_globals();

        let status = {
            let store = &mut self.store;
            let exports = &self.instance.exports;
            let main = exports.get_typed_function::<u32, u32>(store, STYLUS_ENTRY_POINT)?;
            match main.call(store, args.len() as u32) {
                Ok(status) => status,
                Err(outcome) => {
                    self.sync_meter_from_globals();

                    if self.stack_left() == 0 {
                        return Ok(UserOutcome::OutOfStack);
                    }
                    if self.ink_left() == MachineMeter::Exhausted {
                        return Ok(UserOutcome::OutOfInk);
                    }

                    let escape: Escape = match outcome.downcast() {
                        Ok(escape) => escape,
                        Err(_error) => {
                            return Ok(UserOutcome::Failure);
                        }
                    };
                    match escape {
                        Escape::OutOfInk => return Ok(UserOutcome::OutOfInk),
                        Escape::Memory(_) | Escape::Internal(_) | Escape::Logical(_) => {
                            return Ok(UserOutcome::Failure);
                        }
                        Escape::Exit(status) => status,
                    }
                }
            }
        };

        self.sync_meter_from_globals();

        let env = self.env_mut();
        if env.evm_data.tracing {
            env.evm_api
                .capture_hostio("user_returned", &[], &status.to_be_bytes(), ink, ink);
        }

        Ok(match status {
            0 => UserOutcome::Success,
            _ => UserOutcome::Revert,
        })
    }
}
