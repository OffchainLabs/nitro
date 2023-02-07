// Copyright 2020-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(dead_code)]

use std::fmt;

pub const RED: &str = "\x1b[31;1m";
pub const BLUE: &str = "\x1b[34;1m";
pub const YELLOW: &str = "\x1b[33;1m";
pub const PINK: &str = "\x1b[38;5;161;1m";
pub const MINT: &str = "\x1b[38;5;48;1m";
pub const GREY: &str = "\x1b[90m";
pub const RESET: &str = "\x1b[0;0m";

pub const LIME: &str = "\x1b[38;5;119;1m";
pub const LAVENDER: &str = "\x1b[38;5;183;1m";
pub const MAROON: &str = "\x1b[38;5;124;1m";
pub const ORANGE: &str = "\x1b[38;5;202;1m";

pub fn color<S: fmt::Display>(color: &str, text: S) -> String {
    format!("{}{}{}", color, text, RESET)
}

/// Colors text red.
pub fn red<S: fmt::Display>(text: S) -> String {
    color(RED, text)
}

/// Colors text blue.
pub fn blue<S: fmt::Display>(text: S) -> String {
    color(BLUE, text)
}

/// Colors text yellow.
pub fn yellow<S: fmt::Display>(text: S) -> String {
    color(YELLOW, text)
}

/// Colors text pink.
pub fn pink<S: fmt::Display>(text: S) -> String {
    color(PINK, text)
}

/// Colors text grey.
pub fn grey<S: fmt::Display>(text: S) -> String {
    color(GREY, text)
}

/// Colors text lavender.
pub fn lavender<S: fmt::Display>(text: S) -> String {
    color(LAVENDER, text)
}

/// Colors text mint.
pub fn mint<S: fmt::Display>(text: S) -> String {
    color(MINT, text)
}

/// Colors text lime.
pub fn lime<S: fmt::Display>(text: S) -> String {
    color(LIME, text)
}

/// Colors text orange.
pub fn orange<S: fmt::Display>(text: S) -> String {
    color(ORANGE, text)
}

/// Colors text maroon.
pub fn maroon<S: fmt::Display>(text: S) -> String {
    color(MAROON, text)
}

/// Color a bool one of two colors depending on its value.
pub fn color_if(cond: bool, true_color: &str, false_color: &str) -> String {
    match cond {
        true => color(true_color, &format!("{cond}")),
        false => color(false_color, &format!("{cond}")),
    }
}

/// Color a bool if true
pub fn when<S: fmt::Display>(cond: bool, text: S, when_color: &str) -> String {
    match cond {
        true => color(when_color, text),
        false => format!("{text}"),
    }
}
