RISCV            := $(HOME)/.sp1/riscv

.PHONY: help
help: # Show help for each of the Makefile recipes.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

.PHONY: install-sp1
install-sp1: # Install the SP1 RISC-V toolchain.
	@curl -L https://sp1up.succinct.xyz | bash
	@$(HOME)/.sp1/bin/sp1up -c
	@$(RISCV)/bin/riscv64-unknown-elf-gcc --version
