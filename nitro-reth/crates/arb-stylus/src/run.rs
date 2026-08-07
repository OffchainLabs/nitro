use arbos::programs::types::UserOutcome;

use crate::{
    config::StylusConfig,
    error::StylusError,
    evm_api::EvmApi,
    ink::Ink,
    meter::{DepthCheckedMachine, MachineMeter, MeteredMachine, STYLUS_ENTRY_POINT},
    native::NativeInstance,
};

/// Trait for running Stylus WASM programs.
pub trait RunProgram {
    fn run_main(
        &mut self,
        args: &[u8],
        config: StylusConfig,
        ink: Ink,
    ) -> Result<UserOutcome, StylusError>;
}

impl<E: EvmApi> RunProgram for NativeInstance<E> {
    fn run_main(
        &mut self,
        args: &[u8],
        config: StylusConfig,
        ink: Ink,
    ) -> Result<UserOutcome, StylusError> {
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
            let main = exports
                .get_typed_function::<u32, u32>(store, STYLUS_ENTRY_POINT)
                .map_err(|e| StylusError::MissingGlobal(e.to_string()))?;
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

                    let ink_left_now = match self.ink_left() {
                        MachineMeter::Ready(i) => i.0,
                        MachineMeter::Exhausted => 0,
                    };
                    if std::env::var("STYLUS_HOSTIO_TRACE").is_ok() {
                        let consumed = ink.0.saturating_sub(ink_left_now);
                        eprintln!(
                            "[hostio] wasm_trap ink_start={} ink_left={ink_left_now} ink_consumed_total={consumed} stack_left={}",
                            ink.0,
                            self.stack_left(),
                        );
                    }
                    tracing::warn!(target: "stylus",
                        ink = ?self.ink_left(), stack = self.stack_left(),
                        "WASM trap");
                    let escape: StylusError = match outcome.downcast() {
                        Ok(escape) => escape,
                        Err(error) => {
                            tracing::warn!(target: "stylus", err = %error, "WASM trap detail");
                            return Ok(UserOutcome::Failure);
                        }
                    };
                    match escape {
                        StylusError::OutOfInk => return Ok(UserOutcome::OutOfInk),
                        StylusError::Exit(status) => status,
                        _ => return Ok(UserOutcome::Failure),
                    }
                }
            }
        };

        self.sync_meter_from_globals();

        if std::env::var("STYLUS_HOSTIO_TRACE").is_ok() {
            let ink_left = match self.ink_left() {
                MachineMeter::Ready(i) => i.0,
                MachineMeter::Exhausted => 0,
            };
            let consumed = ink.0.saturating_sub(ink_left);
            eprintln!(
                "[hostio] user_returned status={status} ink_start={} ink_left={ink_left} ink_consumed_total={consumed}",
                ink.0,
            );
        }

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
