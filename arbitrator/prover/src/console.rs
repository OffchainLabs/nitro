// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(dead_code)]

use std::fmt;

pub struct Color;

impl Color {
    pub const RED: &'static str = "\x1b[31;1m";
    pub const BLUE: &'static str = "\x1b[34;1m";
    pub const YELLOW: &'static str = "\x1b[33;1m";
    pub const PINK: &'static str = "\x1b[38;5;161;1m";
    pub const MINT: &'static str = "\x1b[38;5;48;1m";
    pub const GREY: &'static str = "\x1b[90m";
    pub const RESET: &'static str = "\x1b[0;0m";

    pub const LIME: &'static str = "\x1b[38;5;119;1m";
    pub const LAVENDER: &'static str = "\x1b[38;5;183;1m";
    pub const MAROON: &'static str = "\x1b[38;5;124;1m";
    pub const ORANGE: &'static str = "\x1b[38;5;202;1m";

    pub fn color<S: fmt::Display>(color: &str, text: S) -> String {
        format!("{}{}{}", color, text, Color::RESET)
    }

    /// Colors text red. Typically used for compile errors.
    pub fn red<S: fmt::Display>(text: S) -> String {
        Color::color(Color::RED, text)
    }

    /// Colors text blue. Typically used for display lines.
    pub fn blue<S: fmt::Display>(text: S) -> String {
        Color::color(Color::BLUE, text)
    }

    /// Colors text yellow. Typically used for compile warnings.
    pub fn yellow<S: fmt::Display>(text: S) -> String {
        Color::color(Color::YELLOW, text)
    }

    /// Colors text pink. Typically used for warnings promoted to errors.
    pub fn pink<S: fmt::Display>(text: S) -> String {
        Color::color(Color::PINK, text)
    }

    /// Colors text grey.
    pub fn grey<S: fmt::Display>(text: S) -> String {
        Color::color(Color::GREY, text)
    }

    /// Colors text lavender. Often used for human-readable values.
    pub fn lavender<S: fmt::Display>(text: S) -> String {
        Color::color(Color::LAVENDER, text)
    }

    /// Colors text mint. Often used for meta-markup in the optimizer.
    pub fn mint<S: fmt::Display>(text: S) -> String {
        Color::color(Color::MINT, text)
    }

    /// Colors text lime.
    pub fn lime<S: fmt::Display>(text: S) -> String {
        Color::color(Color::LIME, text)
    }

    /// Colors text orange.
    pub fn orange<S: fmt::Display>(text: S) -> String {
        Color::color(Color::ORANGE, text)
    }

    /// Colors text maroon.
    pub fn maroon<S: fmt::Display>(text: S) -> String {
        Color::color(Color::MAROON, text)
    }

    /// Color a bool one of two colors depending on its value.
    pub fn color_if(cond: bool, true_color: &str, false_color: &str) -> String {
        match cond {
            true => Color::color(true_color, &format!("{}", cond)),
            false => Color::color(false_color, &format!("{}", cond)),
        }
    }
}
