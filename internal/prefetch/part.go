package prefetch

import "bytes"

type Part struct {
	startOffset uint64
	endOffset   uint64
	buff        *bytes.Buffer
}
