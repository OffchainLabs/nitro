//! Arbitrum RPC add-ons and builder types.

use arb_payload::ArbEngineTypes;
use arb_primitives::ArbPrimitives;
use reth_chainspec::ChainSpec;
use reth_node_builder::{
    rpc::PayloadValidatorBuilder,
    AddOnsContext, FullNodeComponents, NodeTypes,
};

use crate::validator::ArbPayloadValidator;

/// Builder for the Arbitrum payload validator.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbPayloadValidatorBuilder;

impl<N> PayloadValidatorBuilder<N> for ArbPayloadValidatorBuilder
where
    N: FullNodeComponents<
        Types: NodeTypes<
            ChainSpec = ChainSpec,
            Primitives = ArbPrimitives,
            Payload = ArbEngineTypes,
        >,
    >,
{
    type Validator = ArbPayloadValidator;

    async fn build(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::Validator> {
        Ok(ArbPayloadValidator::new(ctx.config.chain.clone()))
    }
}
