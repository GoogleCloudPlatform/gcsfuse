package gcsx

// This instance of MRR is specific to random_reader.go. That means only one buffer is active
// at any point of time.
// Once the random_reader.go calls GetData, this file will check if the existing buffer can serve the data
// if not a new request is made (Add and callback are in file itself) and data is served.

type MRR struct {
	localBuffer []byte
	// start and end buffer offsets
	start int64
	end   int64
}

func (mrr *MRR) TryServingFromMRR(offset int64, len int64) (bool, []byte) {
	if offset >= mrr.start && len+offset <= mrr.end {
		return true, mrr.localBuffer[offset:len]
	}

	return false, nil
}

func (mrr *MRR) GetData(start int64, end int64, len int64) ([]byte, error) {
	// Make a request to GCS
	// For making request to GCS use [start, end]
}
