use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    // Node types and engine types are now wired up (ArbNode, ArbEngineTypes).
    // Full node launch requires implementing the Node trait with:
    // - ArbPoolBuilder (transaction pool)
    // - ArbNetworkBuilder (P2P networking)
    // - ArbPayloadServiceBuilder (block building)
    // These are tracked as follow-up work.
    info!(target: "reth::cli", "arb-reth node types configured, full launch pending pool/network builders");
}
