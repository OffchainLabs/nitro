//! Arbitrum Eth API builder.

use std::marker::PhantomData;

use arb_primitives::ArbPrimitives;
use reth_chainspec::{EthereumHardforks, Hardforks};
use reth_evm::ConfigureEvm;
use reth_node_api::{FullNodeComponents, HeaderTy, TxTy};
use reth_node_builder::{
    rpc::{EthApiBuilder, EthApiCtx},
    NodeTypes,
};
use reth_rpc_convert::{RpcConvert, RpcConverter, RpcTypes, SignableTxRequest};
use reth_rpc_eth_api::{
    helpers::pending_block::BuildPendingEnv,
    FromEvmError,
};
use reth_rpc_eth_types::EthApiError;

use crate::header::ArbHeaderConverter;
use crate::receipt::ArbReceiptConverter;
use crate::response::ArbRpcTxConverter;
use crate::types::ArbRpcTypes;

/// Type alias for the Arbitrum RPC converter.
pub type ArbRpcConvert<N> = RpcConverter<
    ArbRpcTypes,
    <N as FullNodeComponents>::Evm,
    ArbReceiptConverter,
    ArbHeaderConverter,
    (),
    (),
    ArbRpcTxConverter,
>;

/// Type alias for the Arbitrum EthApi.
pub type ArbEthApi<N> = reth_rpc::EthApi<N, ArbRpcConvert<N>>;

/// Builder for the Arbitrum Eth API.
#[derive(Debug)]
pub struct ArbEthApiBuilder<NetworkT = ArbRpcTypes> {
    _nt: PhantomData<NetworkT>,
}

impl<NetworkT> Default for ArbEthApiBuilder<NetworkT> {
    fn default() -> Self {
        Self {
            _nt: PhantomData,
        }
    }
}

impl<N, NetworkT> EthApiBuilder<N> for ArbEthApiBuilder<NetworkT>
where
    N: FullNodeComponents<
        Types: NodeTypes<ChainSpec: Hardforks + EthereumHardforks, Primitives = ArbPrimitives>,
        Evm: ConfigureEvm<NextBlockEnvCtx: BuildPendingEnv<HeaderTy<N::Types>>>,
    >,
    NetworkT: RpcTypes<TransactionRequest: SignableTxRequest<TxTy<N::Types>>>,
    ArbRpcConvert<N>: RpcConvert<
        Primitives = ArbPrimitives,
        Error = EthApiError,
        Network = NetworkT,
        Evm = N::Evm,
    >,
    EthApiError: FromEvmError<N::Evm>,
{
    type EthApi = reth_rpc::EthApi<N, ArbRpcConvert<N>>;

    async fn build_eth_api(self, ctx: EthApiCtx<'_, N>) -> eyre::Result<Self::EthApi> {
        let rpc_converter =
            RpcConverter::<ArbRpcTypes, N::Evm, ArbReceiptConverter>::new(ArbReceiptConverter)
                .with_header_converter(ArbHeaderConverter)
                .with_rpc_tx_converter(ArbRpcTxConverter);

        Ok(ctx
            .eth_api_builder()
            .with_rpc_converter(rpc_converter)
            .build())
    }
}
