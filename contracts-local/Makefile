.PHONY: build build-forge build-forge-sol build-forge-gas-dimensions build-forge-gas-dimensions-yul

install:
	forge install --no-git

build: build-forge

build-forge: build-forge-sol build-forge-gas-dimensions build-forge-gas-dimensions-yul

build-forge-sol:
	FOUNDRY_PROFILE=default forge build --skip *.yul --skip src/mocks/HostioTest.sol --skip src/mocks/ArbOS11To32UpgradeTest.sol --skip src/mocks/ArbNativeTokenManagerTest.sol && \
	FOUNDRY_PROFILE=solc824 forge build src/mocks/HostioTest.sol src/mocks/ArbOS11To32UpgradeTest.sol src/mocks/ArbNativeTokenManagerTest.sol && \
	FOUNDRY_PROFILE=default forge build src/precompiles --out out/precompiles

build-forge-gas-dimensions:
	FOUNDRY_PROFILE=gas-dimensions forge build

build-forge-gas-dimensions-yul:
	FOUNDRY_PROFILE=gas-dimensions-yul forge build --skip *.sol
