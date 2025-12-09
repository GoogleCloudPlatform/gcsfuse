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
//
//go:nosplit
func getg() *g

// StackStatus holds the calculated stack metrics
type StackStatus struct {
	CurrentSP uintptr
	StackLo   uintptr
	StackHi   uintptr
	Used      uintptr // Bytes used in current frame
	Remaining uintptr // Physical bytes left in current frame
}

// AddStackLeft calculates the available stack space in the current frame.
//
//go:nosplit
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

// Capture captures the current stack usage and limit.
//
//go:nosplit
func (s *StackGrowthData) Capture() {
	status := GetStackLeft()
	s.Add(int(status.Used), int(status.Used+status.Remaining))
}
