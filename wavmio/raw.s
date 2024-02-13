// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//go:build js
// +build js

#include "textflag.h"

TEXT ·getGlobalStateBytes32(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·setGlobalStateBytes32(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·getGlobalStateU64(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·setGlobalStateU64(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readInboxMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readDelayedInboxMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·resolveTypedPreimage(SB), NOSPLIT, $0
  CallImport
  RET
