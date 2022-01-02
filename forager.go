package goforage

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"
)

type (
	// Scanner scans a directory and calls `Forager` for every new file
	Scanner struct {
		// Forager will forage files
		Forager Forager

		// Cache will cache discovered file names. (optional)
		Cache FileCache
		// FileInactivityCutoff is how long a file must stop
		// being modified before it is foraged. (optional)
		FileInactvityCutoff time.Duration
	}

	Forager func(ctx context.Context, fname string)

	FileCache interface {
		Add(string) error
		Contains(string) (bool, error)
	}

	mapCache map[string]struct{}
)

const (
	DefaultFileInactivityCutoff = 5 * time.Second
)

// ScanForFiles periodically scans `scanDir` looking for new files.
// each new file has a goroutine created to `watch` it.
// `Forager` will be called for every "new" file, and may be called on the same file
// multiple times based on the `Cache`.
// If `Cache` is nil, a `FileCache` will be created, though no guarantees
// are made about performance or other characteristics.
func (s Scanner) ScanForFiles(ctx context.Context, scanDir string) error {
	forager := s.Forager
	if forager == nil {
		return fmt.Errorf("Forager is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	existingFiles := s.Cache
	if existingFiles == nil {
		existingFiles = mapCache{}
	}

	for ctx.Err() == nil {
		finfos, err := ioutil.ReadDir(scanDir)
		if err != nil {
			return err
		}

		for _, finfo := range finfos {
			if !finfo.IsDir() {
				fullPath := path.Join(scanDir, finfo.Name())
				known, err := existingFiles.Contains(fullPath)
				if err != nil {
					return err
				}
				if !known {
					existingFiles.Add(fullPath)
					go s.watch(ctx, forager, fullPath)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}

	return ctx.Err()
}

//watch watches a single file for when it appears to have stopped
//being modified.
func (s Scanner) watch(ctx context.Context, forager Forager, fname string) {
	fileInactivityCutoff := s.FileInactvityCutoff
	if fileInactivityCutoff <= 0 {
		fileInactivityCutoff = DefaultFileInactivityCutoff
	}

	finfo, err := os.Stat(fname)
	for {
		if err != nil {
			// the file was probably deleted
			return
		}
		if ctx.Err() != nil {
			return
		}
		if time.Since(finfo.ModTime()) > fileInactivityCutoff {
			break
		}
		time.Sleep(1 * time.Second)
		finfo, err = os.Stat(fname)
	}

	forager(ctx, fname)
}

func (m mapCache) Add(s string) error {
	m[s] = struct{}{}
	return nil
}

func (m mapCache) Contains(s string) (bool, error) {
	_, ok := m[s]
	return ok, nil
}
