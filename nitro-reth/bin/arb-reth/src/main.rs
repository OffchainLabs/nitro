//! MOCK (NIT-5205): stand-in for the real `arb-reth` binary until the
//! arbitrum-reth sources are migrated. Supports just enough CLI surface
//! (--version) for the spec-test plumbing to validate the built artifact.

fn main() {
    if std::env::args().any(|arg| arg == "--version") {
        println!("arb-reth {} (mock)", env!("CARGO_PKG_VERSION"));
    }
}
