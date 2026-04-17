use alloy_primitives::address;
use arb_stylus::pages::{
    add_stylus_pages, get_stylus_pages, get_stylus_program_count, pop_stylus_program,
    push_stylus_program, reset_stylus_pages, set_stylus_pages_open,
};

#[test]
fn reset_clears_state() {
    reset_stylus_pages();
    add_stylus_pages(10);
    let addr = address!("AAAA000000000000000000000000000000000000");
    push_stylus_program(addr);
    reset_stylus_pages();
    assert_eq!(get_stylus_pages(), (0, 0));
    assert_eq!(get_stylus_program_count(addr), 0);
}

#[test]
fn add_pages_advances_open_and_ever() {
    reset_stylus_pages();
    let (prev_open, prev_ever) = add_stylus_pages(5);
    assert_eq!(prev_open, 0);
    assert_eq!(prev_ever, 0);
    let (open, ever) = get_stylus_pages();
    assert_eq!(open, 5);
    assert_eq!(ever, 5);

    add_stylus_pages(3);
    let (open, ever) = get_stylus_pages();
    assert_eq!(open, 8);
    assert_eq!(ever, 8);
}

#[test]
fn add_pages_saturates_on_overflow() {
    reset_stylus_pages();
    add_stylus_pages(u16::MAX - 5);
    add_stylus_pages(100);
    let (open, _) = get_stylus_pages();
    assert_eq!(open, u16::MAX);
}

#[test]
fn ever_is_high_water_mark() {
    reset_stylus_pages();
    add_stylus_pages(10);
    set_stylus_pages_open(2);
    let (_, ever) = get_stylus_pages();
    assert_eq!(ever, 10);
    add_stylus_pages(5);
    let (_, ever) = get_stylus_pages();
    assert_eq!(ever, 10);
    add_stylus_pages(20);
    let (open, ever) = get_stylus_pages();
    assert_eq!(open, 27);
    assert_eq!(ever, 27);
}

#[test]
fn push_returns_true_on_reentrant_call() {
    reset_stylus_pages();
    let addr = address!("BBBB000000000000000000000000000000000000");
    assert!(!push_stylus_program(addr));
    assert!(push_stylus_program(addr));
    assert!(push_stylus_program(addr));
}

#[test]
fn push_distinct_addresses_are_not_reentrant() {
    reset_stylus_pages();
    let a = address!("CCCC000000000000000000000000000000000000");
    let b = address!("DDDD000000000000000000000000000000000000");
    assert!(!push_stylus_program(a));
    assert!(!push_stylus_program(b));
}

#[test]
fn pop_decrements_count_and_removes_at_zero() {
    reset_stylus_pages();
    let addr = address!("EEEE000000000000000000000000000000000000");
    push_stylus_program(addr);
    push_stylus_program(addr);
    assert_eq!(get_stylus_program_count(addr), 2);
    pop_stylus_program(addr);
    assert_eq!(get_stylus_program_count(addr), 1);
    pop_stylus_program(addr);
    assert_eq!(get_stylus_program_count(addr), 0);
    pop_stylus_program(addr);
    assert_eq!(get_stylus_program_count(addr), 0);
}
