package file

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	files    []string
	fileChan chan string
}

func NewWatcher(files []string) *Watcher {
	return &Watcher{
		files:    files,
		fileChan: make(chan string),
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating file system watcher: %s", err)
	}
	defer watcher.Close()

	for _, file := range w.files {
		watcher.Add(file)
	}

	defer close(w.fileChan)

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-watcher.Events:
			if ev.Has(fsnotify.Write) {
				w.fileChan <- ev.Name
			}
		case err := <-watcher.Errors:
			return fmt.Errorf("error watching file system: %s", err)
		}
	}
}

func (w *Watcher) FileChanges() <-chan string {
	return w.fileChan
}
