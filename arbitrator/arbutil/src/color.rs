// Copyright 2020-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(dead_code)]

use std::fmt::{Debug, Display};

pub const BLUE: &str = "\x1b[34;1m";
pub const DIM: &str = "\x1b[2m";
pub const GREY: &str = "\x1b[0;0m\x1b[90m";
pub const MINT: &str = "\x1b[38;5;48;1m";
pub const PINK: &str = "\x1b[38;5;161;1m";
pub const RED: &str = "\x1b[31;1m";
pub const CLEAR: &str = "\x1b[0;0m";
pub const WHITE: &str = "\x1b[0;1m";
pub const YELLOW: &str = "\x1b[33;1m";
pub const ORANGE: &str = "\x1b[0;33m";

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
    fn orange(&self) -> String;
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
    fn orange(&self) -> String { self.color(ORANGE) }
}

pub fn when<T: Display>(cond: bool, text: T, when_color: &str) -> String {
    match cond {
        true => text.color(when_color),
        false => format!("{text}"),
    }
}

pub trait DebugColor {
    fn debug_color(&self, color: &str) -> String;

    fn debug_blue(&self) -> String;
    fn debug_dim(&self) -> String;
    fn debug_clear(&self) -> String;
    fn debug_grey(&self) -> String;
    fn debug_mint(&self) -> String;
    fn debug_pink(&self) -> String;
    fn debug_red(&self) -> String;
    fn debug_white(&self) -> String;
    fn debug_yellow(&self) -> String;
    fn debug_orange(&self) -> String;
}

#[rustfmt::skip]
impl<T> DebugColor for T where T: Debug {

    fn debug_color(&self, color: &str) -> String {
        format!("{}{:?}{}", color, self, CLEAR)
    }

    fn debug_blue(&self)   -> String { self.debug_color(BLUE)   }
    fn debug_dim(&self)    -> String { self.debug_color(DIM)    }
    fn debug_clear(&self)  -> String { self.debug_color(CLEAR)  }
    fn debug_grey(&self)   -> String { self.debug_color(GREY)   }
    fn debug_mint(&self)   -> String { self.debug_color(MINT)   }
    fn debug_pink(&self)   -> String { self.debug_color(PINK)   }
    fn debug_red(&self)    -> String { self.debug_color(RED)    }
    fn debug_white(&self)  -> String { self.debug_color(WHITE)  }
    fn debug_yellow(&self) -> String { self.debug_color(YELLOW) }
    fn debug_orange(&self) -> String { self.debug_color(ORANGE) }
}
