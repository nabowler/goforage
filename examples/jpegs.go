package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/nabowler/goforage"
)

var (
	mutex = sync.Mutex{}
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage:\n\t%s /path/to/dir\n", os.Args[0])
		return
	}
	baseDir := os.Args[1]

	panic(goforage.Scanner{
		Forager: findJpegs,
	}.ScanForFiles(context.Background(), baseDir))

}

func findJpegs(ctx context.Context, fname string) {
	fbytes, err := readFile(fname)
	if err != nil {
		// fmt.Printf("Can't read %s duye to %s\n", fname, err)
		return
	}

	// I'd use a library like github.com/gabriel-vasile/mimetype for this,
	// but I don't want to introduce a dependency just for an example.
	// Instead, I'm just going to check for the first three bytes
	// of the possible jpeg magic bytes per
	// https://en.wikipedia.org/wiki/List_of_file_signatures
	if fbytes[0] == 0xFF && fbytes[1] == 0xD8 && fbytes[2] == 0xFF {
		fmt.Printf("%s is probably a jpeg\n", fname)
	}
}

func readFile(fname string) ([]byte, error) {
	// avoid Too Many Files Open errors
	// weighted semaphores could allow more concurrent IO
	mutex.Lock()
	defer mutex.Unlock()

	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 3)
	_, err = io.ReadFull(f, buf)
	return buf, err
}
