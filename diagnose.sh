#!/bin/bash

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Documentation link for installation instructions
INSTALLATION_DOCS_URL="Refer to https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally for installation."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect operating system
OS=$(uname -s)
echo -e "${BLUE}Detected OS: $OS${NC}"
echo -e "${BLUE}Checking prerequisites for building Nitro locally...${NC}"

# Step 1: Check Docker Installation
if command_exists docker; then
    echo -e "${GREEN}Docker is installed.${NC}"
else
    echo -e "${RED}Docker is not installed. $INSTALLATION_DOCS_URL${NC}"
fi

# Step 2: Check if Docker service is running
if [[ "$OS" == "Linux" ]] && ! sudo service docker status >/dev/null; then
    echo -e "${YELLOW}Docker service is not running on Linux. Start it with: sudo service docker start${NC}"
elif [[ "$OS" == "Darwin" ]] && ! docker info >/dev/null 2>&1; then
    echo -e "${YELLOW}Docker service is not running on macOS. Ensure Docker Desktop is started.${NC}"
else
    echo -e "${GREEN}Docker service is running.${NC}"
fi

# Step 3: Check if the correct Nitro branch is checked out and submodules are updated
EXPECTED_BRANCH="v3.2.1"
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$EXPECTED_BRANCH" ]; then
    echo -e "${YELLOW}Switch to the correct branch using: git fetch origin && git checkout $EXPECTED_BRANCH${NC}"
else
    echo -e "${GREEN}The Nitro repository is on the correct branch: $EXPECTED_BRANCH.${NC}"
fi

# Check if submodules are properly initialized and updated
if git submodule status | grep -qE '^-|\+'; then
    echo -e "${YELLOW}Submodules are not properly initialized or updated. Run: git submodule update --init --recursive --force${NC}"
else
    echo -e "${GREEN}All submodules are properly initialized and up to date.${NC}"
fi

# Step 4: Check if Nitro Docker Image is built
if docker images | grep -q "nitro-node"; then
    echo -e "${GREEN}Nitro Docker image is built.${NC}"
else
    echo -e "${RED}Nitro Docker image is not built. Build it using: docker build . --tag nitro-node${NC}"
fi

# Step 5: Check prerequisites for building binaries
echo -e "${BLUE}Checking prerequisites for building Nitro's binaries...${NC}"
if [[ "$OS" == "Linux" ]]; then
    prerequisites=(git curl build-essential cmake npm golang clang make gotestsum wasm2wat lld-13 python3 yarn)
else
    prerequisites=(git curl make cmake npm go gvm golangci-lint wasm2wat clang gotestsum yarn)
fi

for pkg in "${prerequisites[@]}"; do
    if command_exists "$pkg"; then
        [[ "$pkg" == "wasm2wat" ]] && pkg="wabt"
        [[ "$pkg" == "clang" ]] && pkg="llvm"

        # Check for specific symbolic links related to wasm-ld on Linux and macOS
        if [[ "$pkg" == "llvm" ]]; then
            if [[ "$OS" == "Linux" ]]; then
                if [ ! -L /usr/local/bin/wasm-ld ]; then
                    echo -e "${YELLOW}Creating symbolic link for wasm-ld on Linux.${NC}"
                    sudo ln -s /usr/bin/wasm-ld-13 /usr/local/bin/wasm-ld
                else
                    echo -e "${GREEN}Symbolic link for wasm-ld on Linux is already present.${NC}"
                fi
            elif [[ "$OS" == "Darwin" ]]; then
                if [ ! -L /usr/local/bin/wasm-ld ]; then
                    echo -e "${YELLOW}Creating symbolic link for wasm-ld on macOS.${NC}"
                    sudo mkdir -p /usr/local/bin
                    sudo ln -s /opt/homebrew/opt/llvm/bin/wasm-ld /usr/local/bin/wasm-ld
                else
                    echo -e "${GREEN}Symbolic link for wasm-ld on macOS is already present.${NC}"
                fi
            fi
        fi

        echo -e "${GREEN}$pkg is installed.${NC}"
    else
        [[ "$pkg" == "wasm2wat" ]] && pkg="wabt"
        [[ "$pkg" == "clang" ]] && pkg="llvm"
        echo -e "${RED}$pkg is not installed. Please install $pkg. $INSTALLATION_DOCS_URL${NC}"
    fi
done

# Step 6: Check Node.js version
if command_exists node && node -v | grep -q "v18"; then
    echo -e "${GREEN}Node.js version 18 is installed.${NC}"
else
    echo -e "${RED}Node.js version 18 not installed. $INSTALLATION_DOCS_URL${NC}"
fi

# Step 7: Check Rust version
if command_exists rustc && rustc --version | grep -q "1.80.1"; then
    echo -e "${GREEN}Rust version 1.80.1 is installed.${NC}"
else
    echo -e "${RED}Rust version 1.80.1 not installed. $INSTALLATION_DOCS_URL${NC}"
fi

# Step 8: Check Go version
if command_exists go && go version | grep -q "go1.23"; then
    echo -e "${GREEN}Go version 1.23 is installed.${NC}"
else
    echo -e "${RED}Go version 1.23 not installed. $INSTALLATION_DOCS_URL${NC}"
fi

# Step 9: Check Foundry installation
if command_exists foundryup; then
    echo -e "${GREEN}Foundry is installed.${NC}"
else
    echo -e "${RED}Foundry is not installed. $INSTALLATION_DOCS_URL${NC}"
fi

echo -e "${BLUE}Verification complete.${NC}"
echo -e "${YELLOW}Refer to https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally if the build fails for any other reason.${NC}"
