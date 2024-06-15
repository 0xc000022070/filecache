package filecache

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type item struct {
	content []byte
	mu      sync.Mutex

	AccesedAt  time.Time
	ModifiedAt time.Time
}

func (i *item) Duration() time.Duration {
	i.mu.Lock()
	defer i.mu.Unlock()

	return time.Since(i.ModifiedAt)
}

func (i *item) Access() []byte {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.AccesedAt = time.Now()

	return i.content
}

func getCacheItem(path string, maxSize int64) (*item, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, ErrNotFound
	} else if info.IsDir() {
		return nil, ErrIsDirectory
	} else if info.Size() > maxSize {
		return nil, ErrTooLarge
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	item := &item{
		content:    content,
		ModifiedAt: info.ModTime(),
	}

	return item, nil
}

func setCacheItem(path string, content []byte, maxSize int64) (*item, error) {
	if int64(len(content)) > maxSize {
		return nil, ErrTooLarge
	}

	item := &item{
		content: content,
	}

	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
		return nil, ErrInvalidKey
	}

	err := os.WriteFile(path, content, os.FileMode(0o644))
	if err != nil {
		return nil, err
	}

	item.ModifiedAt = time.Now()

	return item, nil
}

func deleteCacheItem(path string) error {
	return os.Remove(path)
}
