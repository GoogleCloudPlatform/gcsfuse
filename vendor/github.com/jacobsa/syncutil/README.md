[![GoDoc](https://godoc.org/github.com/jacobsa/syncutil?status.svg)](https://godoc.org/github.com/jacobsa/syncutil)

This package contains code that supplements the [sync][] package from the Go
standard library. In particular:

*   Bundle, which makes it easy to write code that spawns multiple
    cancellation-aware workers that may fail.
*   InvariantMutex, which makes it possible to automatically check your
    invariants at lock and unlock time.

See the [reference][] for more info.

[sync]: http://godoc.org/sync
[reference]: http://godoc.org/github.com/jacobsa/syncutil
