package storage

import (
	"cloud.google.com/go/storage"
)

// ObjectWriter implements gcs.Writer interface and is used to write content
// to gcs object via resumable upload API.
type ObjectWriter struct {
	*storage.Writer
}

//func (e *ObjectWriter) ContentType() string {
//	return e.Writer.ContentType
//}

func (e *ObjectWriter) ObjectName() string {
	return e.Writer.Name
}

func (e *ObjectWriter) Attrs() *storage.ObjectAttrs {
	return e.Writer.Attrs()
}
