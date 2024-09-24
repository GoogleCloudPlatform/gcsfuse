package cmd

import (
	"os"
)

type CrashWriter struct {
	fileName string
}

func (w *CrashWriter) Write(p []byte) (n int, err error) {
	f, err := os.OpenFile(w.fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
  defer f.Close()

	n, err = f.Write(p)

	return
}
