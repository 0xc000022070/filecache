package filecache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileCache struct {
	namespace string

	mu       sync.Mutex
	wg       sync.WaitGroup
	keyItem  map[string]*item
	pipe     chan string
	shutdown chan struct{}
	closed   bool

	pipeSize      uint
	maxItems      uint
	maxSize       int64
	ttl           time.Duration
	checkInterval time.Duration
}

// Some useful size constants.
const (
	Kilobyte = 1024
	Megabyte = 1024 * 1024
	Gigabyte = 1024 * 1024 * 1024
)

const (
	defaultTTL           = time.Minute * 5
	defaultMaxSize       = Megabyte * 16
	defaultCheckInterval = time.Minute
	defaultMaxItems      = 32
	defaultPipeSize      = 4
)

var (
	ErrIsDirectory = fmt.Errorf("item is a directory")
	ErrTooLarge    = fmt.Errorf("item is too large")
	ErrNotFound    = fmt.Errorf("item not found")
	ErrInvalidKey  = fmt.Errorf("invalid key")
)

func GetDecoded[T any](fc *FileCache, key string) (T, error) {
	var t T

	data, err := fc.Get(key)
	if err != nil {
		return t, err
	}

	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&t)
	if err != nil {
		return t, fmt.Errorf("failed to decode gob: %w", err)
	}

	return t, nil
}

func SetEncoded[T any](fc *FileCache, key string, v T) error {
	b := new(bytes.Buffer)

	err := gob.NewEncoder(b).Encode(v)
	if err != nil {
		return fmt.Errorf("failed to encode value as gob: %w", err)
	}

	return fc.Set(key, b.Bytes())
}

func New(namespace string, options ...fileCacheOptFn) *FileCache {
	fc := FileCache{
		checkInterval: defaultCheckInterval,
		pipeSize:      defaultPipeSize,
		maxItems:      defaultMaxSize,
		maxSize:       defaultMaxSize,
		ttl:           defaultTTL,
	}

	for _, opt := range options {
		opt(&fc)
	}

	fc.pipe = make(chan string, fc.pipeSize)
	fc.keyItem = make(map[string]*item, 0)
	fc.shutdown = make(chan struct{}, 1)
	fc.namespace = namespace

	go fc.vacuum()

	return &fc
}

func (fc *FileCache) Get(key string) ([]byte, error) {
	item, err := fc.getItem(key)
	if err != nil {
		return nil, err
	}

	return item.Access(), nil
}

func (fc *FileCache) Exists(key string) bool {
	_, err := fc.getItem(key)

	return err == nil
}

func (fc *FileCache) Set(key string, content []byte) error {
	path := fc.KeyToPath(key)

	item, err := setCacheItem(path, content, fc.maxSize)
	if err != nil {
		return err
	}

	fc.mu.Lock()
	fc.keyItem[key] = item
	fc.mu.Unlock()

	return nil
}

func (fc *FileCache) Delete(key string) error {
	path := fc.KeyToPath(key)

	fc.mu.Lock()
	delete(fc.keyItem, key)
	fc.mu.Unlock()

	return deleteCacheItem(path)
}

func (fc *FileCache) SizeInMemory() int {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	return len(fc.keyItem)
}

// Alias for `(*FileSystem).Shutdown`. Created to implement the io.Closer interface.
func (fc *FileCache) Close() error {
	fc.Shutdown()

	return nil
}

// Removes the in-memory cache. The filesystem cache is not changed because
// maybe the program will initialize a new cache with the same namespace in
// a near future.
//
// Unnecessary calls if `(*FileSystem).Destroy` was already called.
func (fc *FileCache) Shutdown() {
	close(fc.pipe)
	close(fc.shutdown)
	<-time.After(time.Microsecond)

	fc.mu.Lock()

	for key := range fc.keyItem {
		delete(fc.keyItem, key)
	}

	fc.keyItem = nil

	fc.mu.Unlock()

	fc.wg.Wait()
}

// Destroys the in-memory cache and the filesystem cache.
func (fc *FileCache) Destroy() error {
	if !fc.closed {
		fc.Shutdown()
	}

	dir := fc.getNamespaceDir()

	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}

	return nil
}

func (fc *FileCache) KeyToPath(key string) string {
	return filepath.Join(fc.getNamespaceDir(), key)
}

func (fc *FileCache) getNamespaceDir() string {
	return filepath.Join(os.TempDir(), "fc", fmt.Sprintf("%s-namespace", fc.namespace))
}

func (fc *FileCache) getItem(key string) (*item, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if item, ok := fc.keyItem[key]; ok {
		return item, nil
	}

	path := fc.KeyToPath(key)

	item, err := getCacheItem(path, fc.maxSize)
	if err != nil {
		return nil, err
	}

	fc.keyItem[key] = item

	return item, nil
}

func (fc *FileCache) removeItem(key string, onlyMemory bool) {
	_, err := fc.getItem(key)
	if err == nil {
		fc.mu.Lock()
		delete(fc.keyItem, key)
		fc.mu.Unlock()

		if !onlyMemory {
			path := fc.KeyToPath(key)
			deleteCacheItem(path)
		}
	}
}

func (fc *FileCache) removeOldest(force bool) error {
	var lastAccessedAt time.Time

	oldestKey := ""

	for key, item := range fc.keyItem {
		if force && oldestKey != "" {
			lastAccessedAt = item.AccesedAt
			oldestKey = key
		} else if item.AccesedAt.Before(lastAccessedAt) {
			lastAccessedAt = item.AccesedAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		fc.removeItem(oldestKey, true)
	}

	return nil
}

func (fc *FileCache) vacuum() {
	if fc.checkInterval < 1 {
		return
	}

	fc.wg.Add(1)

	for {
		select {
		case _ = <-fc.shutdown:
			fc.wg.Done()
			return
		case <-time.After(fc.checkInterval):
			for key, item := range fc.keyItem {
				if item.Duration() > fc.ttl {
					fc.removeItem(key, false)
				}
			}
		}
	}
}
