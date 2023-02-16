// Copyright 2020-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(dead_code)]

use std::fmt::Display;

pub const BLUE: &str = "\x1b[34;1m";
pub const DIM: &str = "\x1b[2m";
pub const GREY: &str = "\x1b[0;0m\x1b[90m";
pub const MINT: &str = "\x1b[38;5;48;1m";
pub const PINK: &str = "\x1b[38;5;161;1m";
pub const RED: &str = "\x1b[31;1m";
pub const CLEAR: &str = "\x1b[0;0m";
pub const WHITE: &str = "\x1b[0;1m";
pub const YELLOW: &str = "\x1b[33;1m";

pub trait Color {
    fn color(&self, color: &str) -> String;

    fn blue(&self) -> String;
    fn dim(&self) -> String;
    fn clear(&self) -> String;
    fn grey(&self) -> String;
    fn mint(&self) -> String;
    fn pink(&self) -> String;
    fn red(&self) -> String;
    fn white(&self) -> String;
    fn yellow(&self) -> String;
}

#[rustfmt::skip]
impl<T> Color for T where T: Display {

    fn color(&self, color: &str) -> String {
        format!("{}{}{}", color, self, CLEAR)
    }

    fn blue(&self)   -> String { self.color(BLUE)   }
    fn dim(&self)    -> String { self.color(DIM)    }
    fn clear(&self)  -> String { self.color(CLEAR)  }
    fn grey(&self)   -> String { self.color(GREY)   }
    fn mint(&self)   -> String { self.color(MINT)   }
    fn pink(&self)   -> String { self.color(PINK)   }
    fn red(&self)    -> String { self.color(RED)    }
    fn white(&self)  -> String { self.color(WHITE)  }
    fn yellow(&self) -> String { self.color(YELLOW) }
}

pub fn when<T: Display>(cond: bool, text: T, when_color: &str) -> String {
    match cond {
        true => text.color(when_color),
        false => format!("{text}"),
    }
}
