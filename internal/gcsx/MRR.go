package gcsx

type MRR struct {
}

func (mrr *MRR) GetData(start int64, size int64) []byte {
	// Can this be served from the existing buffer?
	// If yes, serve it.

	// If not, make a request to GCS
	// For making request to GCS, get the range to be downloaded. or can be hardcoded.
}
