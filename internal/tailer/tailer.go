package tailer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Tailer struct {
	path    string
	lines   chan string
	watcher *fsnotify.Watcher
}

func New(path string) (*Tailer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("watcher: %w", err)
	}
	return &Tailer{
		path:    path,
		lines:   make(chan string, 10000),
		watcher: watcher,
	}, nil
}

func (t *Tailer) Lines() <-chan string {
	return t.lines
}

// Start tailing from `offset` bytes (0 = beginning, -1 = end/live only)
func (t *Tailer) Start(offset int64) error {
	f, err := os.Open(t.path)
	if err != nil {
		return fmt.Errorf("open %s: %w", t.path, err)
	}

	// Seek to saved offset
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			log.Printf("[tailer] seek failed, starting from beginning: %v", err)
			f.Seek(0, io.SeekStart)
		}
	} else if offset == -1 {
		// Live only — go to end
		f.Seek(0, io.SeekEnd)
	}

	if err := t.watcher.Add(t.path); err != nil {
		return fmt.Errorf("watch %s: %w", t.path, err)
	}

	go t.tail(f)
	return nil
}

func (t *Tailer) tail(f *os.File) {
	defer f.Close()
	reader := bufio.NewReaderSize(f, 64*1024)

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			t.lines <- line
		}
		if err == nil {
			continue
		}
		if err != io.EOF {
			log.Printf("[tailer] read error: %v", err)
			return
		}

		// EOF — wait for more data via fsnotify
		select {
		case event, ok := <-t.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				// Log rotation — reopen
				log.Printf("[tailer] rotation detected, reopening %s", t.path)
				time.Sleep(200 * time.Millisecond)
				newF, err := os.Open(t.path)
				if err != nil {
					log.Printf("[tailer] reopen error: %v", err)
					return
				}
				f.Close()
				f = newF
				reader = bufio.NewReaderSize(f, 64*1024)
				t.watcher.Add(t.path)
			}
		case err := <-t.watcher.Errors:
			log.Printf("[tailer] watcher error: %v", err)
		case <-time.After(500 * time.Millisecond):
			// Poll — file grow hua ho toh
		}
	}
}

func (t *Tailer) Close() {
	t.watcher.Close()
}
