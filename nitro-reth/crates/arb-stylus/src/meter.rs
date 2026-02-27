use crate::config::PricingParams;
use crate::error::Escape;
use crate::ink::Ink;

/// Names of the WASM globals used for ink metering.
pub const STYLUS_INK_LEFT: &str = "stylus_ink_left";
pub const STYLUS_INK_STATUS: &str = "stylus_ink_status";

/// Names of the WASM globals used for stack depth checking.
pub const STYLUS_STACK_LEFT: &str = "stylus_stack_left";

/// The Stylus program entry point function name.
pub const STYLUS_ENTRY_POINT: &str = "user_entrypoint";

/// Default host I/O ink cost.
pub const HOSTIO_INK: Ink = Ink(8400);

/// State of the ink meter.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MachineMeter {
    Ready(Ink),
    Exhausted,
}

impl MachineMeter {
    pub fn ink(&self) -> Ink {
        match self {
            Self::Ready(ink) => *ink,
            Self::Exhausted => Ink(0),
        }
    }

    pub fn status(&self) -> u32 {
        match self {
            Self::Ready(_) => 0,
            Self::Exhausted => 1,
        }
    }
}

/// Trait for machines that track ink consumption.
pub trait MeteredMachine {
    fn ink_left(&self) -> MachineMeter;
    fn set_meter(&mut self, meter: MachineMeter);

    fn ink_ready(&self) -> Result<Ink, Escape> {
        match self.ink_left() {
            MachineMeter::Ready(ink) => Ok(ink),
            MachineMeter::Exhausted => Escape::out_of_ink(),
        }
    }

    fn set_ink(&mut self, ink: Ink) {
        self.set_meter(MachineMeter::Ready(ink));
    }

    fn buy_ink(&mut self, ink: Ink) -> Result<(), Escape> {
        let current = self.ink_ready()?;
        if current < ink {
            self.set_meter(MachineMeter::Exhausted);
            return Escape::out_of_ink();
        }
        self.set_meter(MachineMeter::Ready(current - ink));
        Ok(())
    }
}

/// Trait for machines that can convert between gas and ink.
pub trait GasMeteredMachine: MeteredMachine {
    fn pricing(&self) -> PricingParams;

    fn buy_gas(&mut self, gas: u64) -> Result<(), Escape> {
        let ink = self.pricing().gas_to_ink(crate::ink::Gas(gas));
        self.buy_ink(ink)
    }
}

/// Trait for machines that track stack depth.
pub trait DepthCheckedMachine {
    fn stack_left(&mut self) -> u32;
    fn set_stack(&mut self, size: u32);
}
