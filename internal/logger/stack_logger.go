package logger

import (
	"unsafe"
)

// stack mimics the runtime.stack struct
type stack struct {
	lo uintptr
	hi uintptr
}

// g mimics the beginning of the runtime.g struct
// We only need the first field (stack) to calculate usage.
type g struct {
	stack stack
}

// getg is implemented in asm_amd64.s
func getg() *g

// StackStatus holds the calculated stack metrics
type StackStatus struct {
	CurrentSP uintptr
	StackLo   uintptr
	StackHi   uintptr
	Used      uintptr // Bytes used in current frame
	Remaining uintptr // Physical bytes left in current frame
}

// GetStackLeft calculates the available stack space in the current frame.
func GetStackLeft() StackStatus {
	// 1. Get the current Goroutine pointer (from assembly)
	gp := getg()

	// 2. Estimate Current Stack Pointer (SP)
	// We take the address of a local variable.
	var x byte
	sp := uintptr(unsafe.Pointer(&x))

	// 3. Calculate
	return StackStatus{
		CurrentSP: sp,
		StackLo:   gp.stack.lo,
		StackHi:   gp.stack.hi,
		Used:      gp.stack.hi - sp,
		Remaining: sp - gp.stack.lo,
	}
}

// PrintStackHealth prints a human-readable summary.
func PrintStackHealth(label string) {
	s := GetStackLeft()

	status := "HEALTHY"
	if s.Remaining < 900 {
		status = "CRITICAL (Split Imminent)"
	}

	Infof("[%s] Status: %s | Used: %d | Rem: %d | Size: %d",
		label, status, s.Used, s.Remaining, s.StackHi-s.StackLo)
}
