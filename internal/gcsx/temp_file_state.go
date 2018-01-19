package gcsx

import (
	"github.com/jacobsa/gcloud/gcs"

	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type tempFileStat struct {
	Name       string
	Synced     bool
	Generation int64
}

type TempFileSate struct {
	mu        sync.Mutex
	stateFile string

	bucket gcs.Bucket
}

func NewTempFileSate(cacheDir string, b gcs.Bucket) *TempFileSate {
	return &TempFileSate{
		stateFile: path.Join(cacheDir, "status.json"),
		bucket:    b,
	}
}

func (p *TempFileSate) getStatusFile() (*os.File, map[string]tempFileStat, error) {
	file, err := os.OpenFile(p.stateFile, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return nil, nil, err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, nil, err
	}

	tmpFileStats := map[string]tempFileStat{}
	if len(bytes) > 0 {
		if err := json.Unmarshal(bytes, &tmpFileStats); err != nil {
			return nil, nil, err
		}
	}
	return file, tmpFileStats, nil
}

func (p *TempFileSate) writeStatusFile(file *os.File, st map[string]tempFileStat) error {
	bytes, err := json.Marshal(st)
	if err != nil {
		return err
	}

	file.Truncate(0)
	file.Seek(0, 0)

	if _, err = file.Write(bytes); err != nil {
		return err
	}
	file.WriteString("\n")
	return nil
}

func (p *TempFileSate) MarkForUpload(tmpFile, dstPath string, generation int64) error {
	return p.update(func(m map[string]tempFileStat) {
		m[tmpFile] = tempFileStat{
			Name:       dstPath,
			Generation: generation,
		}
	})
}

func (p *TempFileSate) MarkUploaded(tmpFile string) error {
	return p.update(func(m map[string]tempFileStat) {
		s := m[tmpFile]
		s.Synced = true
		m[tmpFile] = s
	})
}

func (p *TempFileSate) DeleteFileStatus(tmpFile string) error {
	return p.update(func(m map[string]tempFileStat) {
		delete(m, tmpFile)
	})
}

func (p *TempFileSate) UpdatePaths(oldPath, newPath string) error {
	return p.update(func(m map[string]tempFileStat) {
		for t, s := range m {
			if strings.HasPrefix(s.Name, oldPath) {
				p2 := strings.TrimPrefix(s.Name, oldPath)
				newName := path.Join(newPath, p2)
				if strings.HasSuffix(t, "/") {
					newName = newName + "/"
				}
				s.Name = newName
				m[t] = s
			}
		}
	})
}

func (p *TempFileSate) update(update func(m map[string]tempFileStat)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, st, err := p.getStatusFile()
	if err != nil {
		return err
	}
	defer file.Close()
	update(st)

	return p.writeStatusFile(file, st)
}

func (p *TempFileSate) UploadUnsynced(ctx context.Context) error {
	p.mu.Lock()
	file, st, err := p.getStatusFile()
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.mu.Unlock()
	defer file.Close()

	go func() {
		for t, f := range st {
			if !f.Synced {
				log.Println("local cache file sync.", t, f.Name)
				if err := p.uploadTmpFile(ctx, t, f); err != nil {
					log.Println("local cache file sync failed.", t, f.Name, err)
					continue
				}
				log.Println("local cache file sync done.", t, f.Name)
			} else {
				log.Println("local cache file already synced.", t, f.Name)
			}
			if err := os.Remove(t); err != nil {
				log.Println("failed to remove local cache file.", t, f.Name)
			}

			p.mu.Lock()
			file, st, err := p.getStatusFile()
			if err != nil {
				log.Println("failed to open cache state file.", t, f.Name)
				p.mu.Unlock()
				continue
			}
			delete(st, t)
			if err = p.writeStatusFile(file, st); err != nil {
				log.Println("failed to write cache state file.", t, f.Name)
			}
			file.Close()
			p.mu.Unlock()
		}
	}()
	return nil
}

func (p *TempFileSate) CreateIfEmpty() error {
	csf, err := os.OpenFile(p.stateFile, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return err
	}
	defer csf.Close()
	size, err := csf.Seek(0, 2)
	if err != nil {
		return err
	}
	if size == 0 {
		_, err := csf.WriteString("{}\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *TempFileSate) uploadTmpFile(ctx context.Context, tmpFile string, f tempFileStat) error {
	tfile, err := os.Open(tmpFile)
	defer tfile.Close()
	if err != nil {
		return err
	}
	req := &gcs.CreateObjectRequest{
		Name: f.Name,
		GenerationPrecondition: &f.Generation,
		Contents:               tfile,
		Metadata: map[string]string{
			"gcsfuse_mtime": time.Now().Format(time.RFC3339Nano),
		},
	}
	_, err = p.bucket.CreateObject(ctx, req)
	if err == nil {
		return nil
	}
	if _, ok := err.(*gcs.NotFoundError); ok {
		var gen int64 = 0
		req.GenerationPrecondition = &gen
		_, err = p.bucket.CreateObject(ctx, req)
	}
	return err
}
