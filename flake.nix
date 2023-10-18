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
              targets = [ "wasm32-unknown-unknown" "wasm32-wasi" ];
            };
          in
          # pkgs.mkShell.override { stdenv = pkgs.llvmPackages_16.stdenv; } {
          pkgs.mkShell {

            # By default clang does not find its resource dir. See
            # https://discourse.nixos.org/t/why-is-the-clang-resource-dir-split-in-a-separate-package/34114
            CPATH = "${pkgs.llvmPackages_16.libclang.lib}/lib/clang/16/include";
            packages = with pkgs; [
              stableToolchain

              llvmPackages_16.clang-unwrapped # provides clang without wrapper
              llvmPackages_16.bintools-unwrapped # provides wasm-ld
              llvmPackages_16.llvm

              go
              # goimports, godoc, etc.
              gotools
              golangci-lint

              # Node
              nodejs
              yarn

              rust-cbindgen
              # cmake
              # wabt
              # libiconv
              # cargo

              # Docker
              docker-compose # provides the `docker-compose` command
              docker-buildx
              docker-credential-helpers # for `docker-credential-osxkeychain` command
            ] ++ lib.optionals stdenv.isDarwin [
              darwin.libobjc
              darwin.IOKit
              darwin.apple_sdk.frameworks.CoreFoundation
              # darwin.apple_sdk.Libsystem
            ];
            # With the unwrapped clang first in the path we can run `make build-wasm-libs` but
            # it breaks `cargo build --manifest-path arbitrator/Cargo.toml --release --lib -p prover`
            # because it doesn't find the standard library when compiling with clang.
            shellHook = ''
              export PATH="${pkgs.llvmPackages_16.clang-unwrapped}/bin:$PATH"

              # Prevent cargo aliases from using programs in `~/.cargo` to avoid conflicts
              # with rustup installations.
              export CARGO_HOME=$HOME/.cargo-nix

              # Fix docker-buildx command on OSX. Can we do this in a cleaner way?
              mkdir -p ~/.docker/cli-plugins
              # Check if the file exists, otherwise symlink
              test -f $HOME/.docker/cli-plugins/docker-buildx || ln -sn $(which docker-buildx) $HOME/.docker/cli-plugins
            '';
            RUST_SRC_PATH = "${stableToolchain}/lib/rustlib/src/rust/library";
          };
      });
}
