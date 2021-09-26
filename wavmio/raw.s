#include "textflag.h"

TEXT ·getLastBlockHash(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readInboxMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·advanceInboxMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·resolvePreImage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·setLastBlockHash(SB), NOSPLIT, $0
  CallImport
  RET
