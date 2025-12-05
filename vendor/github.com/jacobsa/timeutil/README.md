[![GoDoc](https://godoc.org/github.com/jacobsa/timeutil?status.svg)](https://godoc.org/github.com/jacobsa/timeutil)

This package contains code that supplements the [time][] package from the Go
standard library. In particular:

*   A Clock interface, with a fake implementation that can be used in tests.
*   Implementations of [oglematchers.Matcher][] for time values.

See the [reference][] for more info.

[time]: http://godoc.org/time
[oglematchers.Matcher]: https://godoc.org/github.com/jacobsa/oglematchers#Matcher
[reference]: http://godoc.org/github.com/jacobsa/timeutil
