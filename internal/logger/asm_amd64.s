#include "textflag.h"

// func getg() unsafe.Pointer
// This reads the Thread Local Storage (TLS) to get the current 'g' structure.
TEXT ·getg(SB), NOSPLIT, $0-8
    MOVQ (TLS), AX
    MOVQ AX, ret+0(FP)
    RET
