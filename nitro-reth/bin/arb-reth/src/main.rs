use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    // TODO: Re-enable node launch once ArbPrimitives-compatible pool/add-ons builders exist.
    // The Ethereum builders hard-code EthPrimitives which is incompatible with ArbPrimitives.
    info!(target: "reth::cli", "arb-reth binary placeholder — node builder integration pending");
}
