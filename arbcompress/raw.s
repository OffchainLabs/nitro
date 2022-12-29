// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

#include "textflag.h"

TEXT ·brotliCompress(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·brotliDecompress(SB), NOSPLIT, $0
  CallImport
  RET
