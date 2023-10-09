{
  description = "A Nix-flake-based Go 1.20 development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.flake-compat.url = "github:edolstra/flake-compat";
  inputs.flake-compat.flake = false;
  inputs.rust-overlay.url = "github:oxalica/rust-overlay";


  # Closes commit in foundry.nix to forge 3b1129b used in CI.
  inputs.foundry.url = "github:shazow/foundry.nix/fef36a77f0838fe278cc01ccbafbab8cd38ad26f";


  outputs = { self, flake-utils, nixpkgs, foundry, rust-overlay, ... }:
    let
      goVersion = 20; # Change this to update the whole stack
      overlays = [
        (import rust-overlay)
        (final: prev: {
          go = prev."go_1_${toString goVersion}";
          # Overlaying nodejs here to ensure nodePackages use the desired
          # version of nodejs.
          nodejs = prev.nodejs-16_x;
          pnpm = prev.nodePackages.pnpm;
          yarn = prev.nodePackages.yarn;
        })
        foundry.overlay
      ];
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit overlays system;
          config = {
            permittedInsecurePackages = [ "nodejs-16.20.2" ];
          };
        };
      in
      {
        devShells.default = 
        let
            stableToolchain = pkgs.rust-bin.stable.latest.minimal.override {
              extensions = [ "rustfmt" "clippy" "llvm-tools-preview" "rust-src" ];
              targets = ["wasm32-unknown-unknown" "wasm32-wasi"];
            };
        in 
        pkgs.mkShell{
          packages = with pkgs; [
            go
            # goimports, godoc, etc.
            gotools
            golangci-lint

            # Node
            nodejs
            pnpm
            yarn

            rust-cbindgen
            cmake
            wabt
            llvmPackages_15.clang
            libiconv
            cargo
            clang
            stableToolchain


            # Docker
            docker-compose # provides the `docker-compose` command
          ] ++ lib.optionals stdenv.isDarwin [
            darwin.libobjc
            darwin.IOKit
            darwin.apple_sdk.frameworks.CoreFoundation
          ];
          shellHook = ''
                # Prevent cargo aliases from using programs in `~/.cargo` to avoid conflicts
                # with rustup installations.
                export CARGO_HOME=$HOME/.cargo-nix
              '';
          RUST_SRC_PATH = "${stableToolchain}/lib/rustlib/src/rust/library";
        };
      });
}
