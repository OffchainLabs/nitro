#![cfg(feature = "docker")]

use crate::{error::HarnessError, node::NodeStartCtx, Result};

pub struct DockerNitro {
    _container_id: String,
}

impl DockerNitro {
    pub fn start(_ctx: &NodeStartCtx) -> Result<Self> {
        Err(HarnessError::NotImplemented {
            what: "DockerNitro::start",
        })
    }
}
