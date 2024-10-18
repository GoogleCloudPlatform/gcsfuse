package gcsx

// This instance of MRR is specific to random_reader.go. That means only one buffer is active
// at any point of time.
// Once the random_reader.go calls GetData, this file will check if the existing buffer can serve the data
// if not a new request is made (Add and callback are in file itself) and data is served.

type MRR struct {
}

func (mrr *MRR) GetData(start int64, size int64) []byte {
	// Can this be served from the existing buffer?
	// If yes, serve it.

	// If not, make a request to GCS
	// For making request to GCS, get the range to be downloaded. or can be hardcoded.
}
