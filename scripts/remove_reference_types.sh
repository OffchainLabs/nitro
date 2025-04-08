#!/usr/bin/env bash

# This script removes reference types from a wasm file

wasm2wat "$1" > "$1.wat"
wat2wasm "$1.wat" -o "$1"
