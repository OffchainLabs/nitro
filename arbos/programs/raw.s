// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

#include "textflag.h"

TEXT ·compileUserWasmRustImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·callUserWasmRustImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readRustVecImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·freeRustVecImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·rustParamsImpl(SB), NOSPLIT, $0
  CallImport
  RET
