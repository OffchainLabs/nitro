### Internal
- Fix Docker image build failure (`forge: No such file or directory` during `make build-solidity`): the new foundryup-init 2.0.0 installer served at foundry.paradigm.xyz no longer adds `~/.foundry/bin` to `~/.bashrc`, so the `contracts-builder` stage now sets `PATH` explicitly instead of sourcing `~/.bashrc`.
