#!/usr/bin/env bash
# This script checks the prerequisites for building Arbitrum Nitro locally.

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

node_version_needed="v24"
rust_version_needed="1.83.0"
golangci_lint_version_needed="2.3.0"

if [[ -f go.mod ]]; then
    go_version_needed=$(grep "^go " go.mod | awk '{print $2}')
else
    if [[ -f ../go.mod ]]; then
        go_version_needed=$(grep "^go " ../go.mod | awk '{print $2}')
    else
        go_version_needed="unknown"
    fi
fi

# Documentation link for installation instructions
INSTALLATION_DOCS_URL="Refer to https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally for installation."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

EXIT_CODE=0

# Detect operating system
OS=$(uname -s)
echo -e "${BLUE}Detected OS: $OS${NC}"

# Check Docker Installation
if command_exists docker; then
    echo -e "${GREEN}Docker is installed.${NC}"
else
    echo -e "${RED}Docker is not installed.${NC}"
    EXIT_CODE=1
fi

# Check if Docker service is running
if [[ "$OS" == "Linux" ]] && ! pidof dockerd >/dev/null; then
    echo -e "${YELLOW}Docker service is not running on Linux. Start it with: sudo service docker start${NC}"
    EXIT_CODE=1
elif [[ "$OS" == "Darwin" ]] && ! docker info >/dev/null 2>&1; then
    echo -e "${YELLOW}Docker service is not running on macOS. Ensure Docker Desktop is started.${NC}"
   EXIT_CODE=1
else
    echo -e "${GREEN}Docker service is running.${NC}"
fi

# Check the version tag
VERSION_TAG=$(git tag --points-at HEAD | sed '/-/!s/$/_/' | sort -rV | sed 's/_$//' | head -n 1 | grep ^ || git show -s --pretty=%D | sed 's/, /\n/g' | grep -v '^origin/' | grep -v '^grafted\|HEAD\|master\|main$' || echo "")
if [[ -z "${VERSION_TAG}" ]]; then
    echo -e "${YELLOW}Untagged version of Nitro checked out, may not be fully tested.${NC}"
else
    echo -e "${GREEN}You are on Nitro version tag: $VERSION_TAG${NC}"
fi

# Check if submodules are properly initialized and updated
if git submodule status | grep -qE '^-|\+'; then
    echo -e "${YELLOW}Submodules are not properly initialized or updated. Run: git submodule update --init --recursive${NC}"
    EXIT_CODE=1
else
    echo -e "${GREEN}All submodules are properly initialized and up to date.${NC}"
fi

# Check if Nitro Docker Image is built
if docker images | grep -q "nitro-node"; then
    echo -e "${GREEN}Nitro Docker image is built.${NC}"
else
    echo -e "${YELLOW}Nitro Docker image is not built. Build it using: docker build . --tag nitro-node${NC}"
fi

# Check prerequisites for building binaries
prerequisites=(git go curl clang make cmake npm wasm2wat wasm-ld yarn gotestsum python3)
if [[ "$OS" == "Linux" ]]; then
    prerequisites+=()
else
    prerequisites+=()
fi

for pkg in "${prerequisites[@]}"; do
    if command_exists "$pkg"; then
        exists=true
    else
        exists=false
    fi
    [[ "$pkg" == "make" ]] && pkg="build-essential"
    [[ "$pkg" == "wasm2wat" ]] && pkg="wabt"
    [[ "$pkg" == "clang" ]] && pkg="llvm"
    [[ "$pkg" == "wasm-ld" ]] && pkg="lld"
    if $exists; then
        # There is no way to check for wabt / llvm directly, since they install multiple tools
        # So instead, we check for wasm2wat and clang, which are part of wabt and llvm respectively
        # and if they are installed, we assume wabt / llvm is installed else we ask the user to install wabt / llvm

        echo -e "${GREEN}$pkg is installed.${NC}"
    else
        echo -e "${RED}$pkg is not installed. Please install $pkg.${NC}"
        EXIT_CODE=1
    fi
done

# Check Node.js version
if command_exists node && node -v | grep -q "$node_version_needed"; then
    echo -e "${GREEN}Node.js version $node_version_needed is installed.${NC}"
else
    echo -e "${RED}Node.js version $node_version_needed not installed.${NC}"
    EXIT_CODE=1
fi

# Check Rust version
if command_exists rustc && rustc --version | grep -q "$rust_version_needed"; then
    echo -e "${GREEN}Rust version $rust_version_needed is installed.${NC}"
else
    echo -e "${RED}Rust version $rust_version_needed is not installed.${NC}"
    EXIT_CODE=1
fi

# Check Rust nightly toolchain
if rustup toolchain list | grep -q "nightly"; then
    echo -e "${GREEN}Rust nightly toolchain is installed.${NC}"
else
    echo -e "${RED}Rust nightly toolchain is not installed. Install it using: rustup toolchain install nightly${NC}"
    EXIT_CODE=1
fi

# Check Go version
if command_exists go && go version | grep -q "$go_version_needed"; then
    echo -e "${GREEN}Go version $go_version_needed is installed.${NC}"
else
    echo -e "${RED}Go version $go_version_needed not installed.${NC}"
    EXIT_CODE=1
fi

# Check Go Linter version
if command_exists golangci-lint && golangci-lint version | grep -q "$golangci_lint_version_needed"; then
    echo -e "${GREEN}golangci-lint version $golangci_lint_version_needed is installed.${NC}"
else
    echo -e "${RED}golangci-lint version $golangci_lint_version_needed not installed.${NC}"
    EXIT_CODE=1
fi

# Check Foundry installation
if command_exists foundryup; then
    echo -e "${GREEN}Foundry is installed.${NC}"
else
    echo -e "${RED}Foundry is not installed.${NC}"
    EXIT_CODE=1
fi


if [ $EXIT_CODE != 0 ]; then
    echo -e "${RED}One or more dependencies missing. $INSTALLATION_DOCS_URL${NC}"
else
    echo -e "${BLUE}Build readiness check passed.${NC}"
fi

exit $EXIT_CODE
