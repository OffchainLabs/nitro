// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build js
// +build js

#include "textflag.h"

TEXT ·debugPoolHash(SB), NOSPLIT, $0
  CallImport
  RET
