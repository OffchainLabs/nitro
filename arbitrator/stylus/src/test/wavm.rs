// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::test::{new_test_machine, test_compile_config};
use eyre::Result;
use prover::{programs::prelude::*, Machine};

#[test]
fn test_ink() -> Result<()> {
    let mut compile = test_compile_config();
    compile.pricing.costs = super::expensive_add;

    let machine = &mut new_test_machine("tests/add.wat", &compile)?;
    let call = |mech: &mut Machine, v: u32| mech.call_function("user", "add_one", vec![v.into()]);

    macro_rules! exhaust {
        ($ink:expr) => {
            machine.set_ink($ink);
            assert_eq!(machine.ink_left(), MachineMeter::Ready($ink));
            assert!(call(machine, 32).is_err());
            assert_eq!(machine.ink_left(), MachineMeter::Exhausted);
        };
    }

    exhaust!(0);
    exhaust!(50);
    exhaust!(99);

    let mut ink_left = 500;
    machine.set_ink(ink_left);
    while ink_left > 0 {
        assert_eq!(machine.ink_left(), MachineMeter::Ready(ink_left));
        assert_eq!(call(machine, 64)?, vec![65_u32.into()]);
        ink_left -= 100;
    }
    assert!(call(machine, 32).is_err());
    assert_eq!(machine.ink_left(), MachineMeter::Exhausted);
    Ok(())
}

#[test]
fn test_depth() -> Result<()> {
    // in depth.wat
    //    the `depth` global equals the number of times `recurse` is called
    //    the `recurse` function calls itself
    //    the `recurse` function has 1 parameter and 2 locals
    //    comments show that the max depth is 3 words

    let machine = &mut new_test_machine("tests/depth.wat", &test_compile_config())?;
    let call = |mech: &mut Machine| mech.call_function("user", "recurse", vec![0_u64.into()]);

    let program_depth: u32 = machine.get_global("depth")?.try_into()?;
    assert_eq!(program_depth, 0);

    let mut check = |space: u32, expected: u32| -> Result<()> {
        machine.set_global("depth", 0_u32.into())?;
        machine.set_stack(space);
        assert_eq!(machine.stack_left(), space);

        assert!(call(machine).is_err());
        assert_eq!(machine.stack_left(), 0);

        let program_depth: u32 = machine.get_global("depth")?.try_into()?;
        assert_eq!(program_depth, expected);
        Ok(())
    };

    let locals = 2;
    let depth = 3;
    let fixed = 4;

    let frame_size = locals + depth + fixed;

    check(frame_size, 0)?; // should immediately exhaust (space left <= frame)
    check(frame_size + 1, 1)?;
    check(2 * frame_size, 1)?;
    check(2 * frame_size + 1, 2)?;
    check(4 * frame_size, 3)?;
    check(4 * frame_size + frame_size / 2, 4)
}

#[test]
fn test_start() -> Result<()> {
    // in start.wat
    //     the `status` global equals 10 at initialization
    //     the `start` function increments `status`
    //     by the spec, `start` must run at initialization

    fn check(machine: &mut Machine, value: u32) -> Result<()> {
        let status: u32 = machine.get_global("status")?.try_into()?;
        assert_eq!(status, value);
        Ok(())
    }

    let compile = test_compile_config();
    let mut machine = &mut new_test_machine("tests/start.wat", &compile)?;
    check(machine, 10)?;

    let call = |mech: &mut Machine, name: &str| mech.call_function("user", name, vec![]);

    call(machine, "move_me")?;
    call(machine, "stylus_start")?;
    check(&mut machine, 12)
}
