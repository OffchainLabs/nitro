#include "textflag.h"

TEXT ·getLastBlockHash(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readInboxMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·readDelayedInboxMessage(SB), NOSPLIT, $0
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

TEXT ·getPositionWithinMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·setPositionWithinMessage(SB), NOSPLIT, $0
  CallImport
  RET

TEXT ·getInboxPosition(SB), NOSPLIT, $0
  CallImport
  RET
