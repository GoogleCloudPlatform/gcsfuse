package refactoring

import "context"

type Reader interface {
	Read(ctx context.Context, dst []byte, offset int64) (n int, err error)
}
