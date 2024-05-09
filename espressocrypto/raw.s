//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

//go:build js
// +build js

#include "textflag.h"

TEXT ·verifyNamespace(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·verifyMerkleProof(SB), NOSPLIT, $0
  CallImport
  RET
