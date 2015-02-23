// Copyright 2015 Google Inc. All Rights Reserved.
// Author: jacobsa@google.com (Aaron Jacobs)
//
// A collection of tests for a file system backed by a GCS bucket, where in
// most cases we interact with the file system directly for creating and
// mofiying files (rather than through the side channel of the GCS bucket
// itself).
//
// These tests are registered by RegisterFSTests.

package fstesting

import (
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"time"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Read-write interaction
////////////////////////////////////////////////////////////////////////

type readWriteTest struct {
	fsTest
}

func (t *readWriteTest) OpenNonExistent_CreateFlagNotSet() {
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0700)
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *readWriteTest) OpenNonExistent_ReadOnly() {
	// Open the file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDONLY|os.O_CREATE,
		0700)

	AssertEq(nil, err)
	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Read its contents.
	fileContents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq("", string(fileContents))

	// Attempt to write.
	n, err := f.Write([]byte("taco"))

	AssertEq(0, n)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *readWriteTest) OpenNonExistent_WriteOnly() {
	// Open the file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_WRONLY|os.O_CREATE,
		0700)

	AssertEq(nil, err)
	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Reading should fail.
	_, err = ioutil.ReadAll(f)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("1"), 1)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(3, 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectEq(len("010222"), fi.Size())

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("010222", string(fileContents))
}

func (t *readWriteTest) OpenNonExistent_ReadWrite() {
	// Open the file.
	f, err := os.OpenFile(
		path.Join(t.mfs.Dir(), "foo"),
		os.O_RDWR|os.O_CREATE,
		0700)

	AssertEq(nil, err)
	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(0, fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("1"), 1)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(3, 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectEq(len("010222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(2, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("0222", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len("010222"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("010222", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents after opening anew for reading.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("010222", string(fileContents))
}

func (t *readWriteTest) OpenNonExistent_Append() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenExistingFile_ReadOnly() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file for reading.
	f, err := os.Open(path.Join(t.mfs.Dir(), "foo"))
	AssertEq(nil, err)

	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Read its contents.
	fileContents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := f.Write([]byte("taco"))

	AssertEq(0, n)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *readWriteTest) OpenExistingFile_WriteOnly() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_WRONLY, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Reading should fail.
	_, err = ioutil.ReadAll(f)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *readWriteTest) OpenExistingFile_ReadWrite() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file for reading.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents after opening anew for reading.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *readWriteTest) OpenExistingFile_Append() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file for reading.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Check its vitals.
	ExpectEq(path.Join(t.mfs.Dir(), "foo"), f.Name())

	fi, err := f.Stat()
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectEq(os.FileMode(0), fi.Mode() & ^os.ModePerm)
	ExpectLt(math.Abs(time.Since(fi.ModTime()).Seconds()), 10)
	ExpectFalse(fi.IsDir())

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = f.Seek(1, 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Check the size now.
	fi, err = f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents after opening anew for reading.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222", string(fileContents))
}

func (t *readWriteTest) TruncateExistingFile_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateExistingFile_Append() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_ReadWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenAlreadyOpenedFile_Append() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenReadOnlyFileForWrite() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) ReadOnlyMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file for reading.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDONLY, 0)
	AssertEq(nil, err)

	defer func() {
		ExpectEq(nil, f.Close())
	}()

	// Read its contents.
	fileContents, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectEq(contents, string(fileContents))

	// Attempt to write.
	n, err := f.Write([]byte("taco"))

	AssertEq(0, n)
	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))
}

func (t *readWriteTest) WriteOnlyMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_WRONLY, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Reading should fail.
	_, err = ioutil.ReadAll(f)

	AssertNe(nil, err)
	ExpectThat(err, Error(HasSubstr("bad file descriptor")))

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *readWriteTest) ReadWriteMode() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write to the start of the file using File.Write.
	_, err = f.Write([]byte("000"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Seek and write past the end of the file.
	_, err = f.Seek(int64(len(contents)), 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents after opening anew for reading.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("000o111ritoenchilada222", string(fileContents))
}

func (t *readWriteTest) AppendMode_ReadOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) AppendMode_WriteOnly() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) AppendMode_ReadWrite() {
	// Create a file.
	const contents = "tacoburritoenchilada"
	AssertEq(
		nil,
		ioutil.WriteFile(
			path.Join(t.mfs.Dir(), "foo"),
			[]byte(contents),
			os.FileMode(0644)))

	// Open the file.
	f, err := os.OpenFile(path.Join(t.mfs.Dir(), "foo"), os.O_RDWR|os.O_APPEND, 0)
	AssertEq(nil, err)

	defer func() {
		if f != nil {
			ExpectEq(nil, f.Close())
		}
	}()

	// Write using File.Write. This should go to the end of the file regardless
	// of whether we Seek somewhere else first.
	_, err = f.Seek(1, 0)
	AssertEq(nil, err)

	_, err = f.Write([]byte("222"))
	AssertEq(nil, err)

	// Write to the middle of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("111"), 4)
	AssertEq(nil, err)

	// Write well past the end of the file using File.WriteAt.
	_, err = f.WriteAt([]byte("333"), 100)
	AssertEq(nil, err)

	// Check the size now.
	fi, err := f.Stat()
	AssertEq(nil, err)
	ExpectEq(len(contents)+len("222333"), fi.Size())

	// Read some contents with Seek and Read.
	_, err = f.Seek(4, 0)
	AssertEq(nil, err)

	buf := make([]byte, 4)
	_, err = io.ReadFull(f, buf)

	AssertEq(nil, err)
	ExpectEq("111r", string(buf))

	// Read the full contents with ReadAt.
	buf = make([]byte, len(contents)+len("222333"))
	_, err = f.ReadAt(buf, 0)

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222333", string(buf))

	// Close the file.
	AssertEq(nil, f.Close())
	f = nil

	// Read back its contents after opening anew for reading.
	fileContents, err := ioutil.ReadFile(path.Join(t.mfs.Dir(), "foo"))

	AssertEq(nil, err)
	ExpectEq("taco111ritoenchilada222333", string(fileContents))
}

func (t *readWriteTest) WritePastEndOfFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Seek() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Stat() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Sync() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) Truncate() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) CreateEmptyDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) CreateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) OpenWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) TruncateWithinSubDirectory() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) ReadFromWriteOnlyFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) WriteToReadOnlyFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) StatUnopenedFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) LstatUnopenedFile() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) ListingsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) AdditionsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) RemovalsAreCached() {
	AssertTrue(false, "TODO")
}

func (t *readWriteTest) BufferedWritesFlushedOnUnmount() {
	AssertTrue(false, "TODO")
}
