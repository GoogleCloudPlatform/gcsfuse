package util

// A Gate limits concurrency.
type Gate struct {
	c chan struct{}
}

// NewGate returns a new gate that will only permit max operations at once.
func NewGate(max int) *Gate {
	return &Gate{make(chan struct{}, max)}
}

// Start starts an operation, blocking until the gate has room.
func (g *Gate) Start() {
	g.c <- struct{}{}
}

// StartCh is the same as Start, but accepts a channel to cancel.
func (g *Gate) StartCh(stop <-chan struct{}) bool {
	select {
	case g.c <- struct{}{}:
		return true
	case <-stop:
		return false
	}
}

// Done finishes an operation.
func (g *Gate) Done() {
	select {
	case <-g.c:
	default:
		panic("Done called more than Start")
	}
}

