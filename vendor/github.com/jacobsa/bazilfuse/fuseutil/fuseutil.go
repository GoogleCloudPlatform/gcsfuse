package fuseutil

import (
	"github.com/jacobsa/bazilfuse"
)

// HandleRead handles a read request assuming that data is the entire file content.
// It adjusts the amount returned in resp according to req.Offset and req.Size.
func HandleRead(req *bazilfuse.ReadRequest, resp *bazilfuse.ReadResponse, data []byte) {
	if req.Offset >= int64(len(data)) {
		data = nil
	} else {
		data = data[req.Offset:]
	}
	if len(data) > req.Size {
		data = data[:req.Size]
	}
	n := copy(resp.Data[:req.Size], data)
	resp.Data = resp.Data[:n]
}
