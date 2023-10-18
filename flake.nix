{
  description = "A Nix-flake-based Go 1.20 development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.flake-compat.url = "github:edolstra/flake-compat";
  inputs.flake-compat.flake = false;
  inputs.rust-overlay.url = "github:oxalica/rust-overlay";
  inputs.foundry.url = "github:shazow/foundry.nix/monthly";

  outputs = { flake-utils, nixpkgs, foundry, rust-overlay, ... }:
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
        stableToolchain = pkgs.rust-bin.stable.latest.minimal.override {
          extensions = [ "rustfmt" "clippy" "llvm-tools-preview" "rust-src" ];
          targets = [ "wasm32-unknown-unknown" "wasm32-wasi" ];
        };
        shellHook = ''
          # Prevent cargo aliases from using programs in `~/.cargo` to avoid conflicts
          # with rustup installations.
          export CARGO_HOME=$HOME/.cargo-nix
        ''
        + pkgs.lib.optionalString pkgs.stdenv.isDarwin ''
          # Fix docker-buildx command on OSX. Can we do this in a cleaner way?
          mkdir -p ~/.docker/cli-plugins
          # Check if the file exists, otherwise symlink
          test -f $HOME/.docker/cli-plugins/docker-buildx || ln -sn $(which docker-buildx) $HOME/.docker/cli-plugins
        '';
      in
      {
        devShells =
          {
            # This shell is only used for one make recipe because the other
            # shell is not able to build one recipe and we haven't managed to
            # come up with a dev shell that works for everything.
            #
            #    nix develop .#wasm -c make build-wasm-libs
            #
            # After that, the other shell can be used to run `make build`.
            wasm = pkgs.mkShell {
              # By default clang-unwrapped does not find its resource dir. See
              # https://discourse.nixos.org/t/why-is-the-clang-resource-dir-split-in-a-separate-package/34114
              CPATH = "${pkgs.llvmPackages_16.libclang.lib}/lib/clang/16/include";
              packages = with pkgs; [
                stableToolchain

                llvmPackages_16.clang-unwrapped # provides clang without wrapper
                llvmPackages_16.bintools # provides wasm-ld

                # Docker
                docker-compose # provides the `docker-compose` command
                docker-buildx
                docker-credential-helpers # for `docker-credential-osxkeychain` command
              ];

              # Ensure the unwrapped clang is used by default.
              shellHook = shellHook + ''
                export PATH="${pkgs.llvmPackages_16.clang-unwrapped}/bin:$PATH"
              '';
            };
            default = pkgs.mkShell {

              packages = with pkgs; [
                stableToolchain

                # llvmPackages_16.clang # provides clang without wrapper
                # llvmPackages_16.bintools # provides wasm-ld

                go
                # goimports, godoc, etc.
                gotools
                golangci-lint
                gotestsum

                # Node
                nodejs
                yarn

                # wasm
                rust-cbindgen
                wabt

                # Docker
                docker-compose # provides the `docker-compose` command
                docker-buildx
                docker-credential-helpers # for `docker-credential-osxkeychain` command

                foundry-bin
              ] ++ lib.optionals stdenv.isDarwin [
                darwin.libobjc
                darwin.IOKit
                darwin.apple_sdk.frameworks.CoreFoundation
              ];
              inherit shellHook;
              RUST_SRC_PATH = "${stableToolchain}/lib/rustlib/src/rust/library";
            };
          };
      });
}
