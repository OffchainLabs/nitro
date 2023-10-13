// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

#include "textflag.h"

TEXT ·activateWasmRustImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·callUserWasmRustImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readRustVecLenImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·rustVecIntoSliceImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·rustConfigImpl(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·rustEvmDataImpl(SB), NOSPLIT, $0
  CallImport
  RET
