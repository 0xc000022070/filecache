package filecache

import "time"

type fileCacheOptFn func(*FileCache)

func WithMaxItems(maxItems uint) fileCacheOptFn {
	return func(fc *FileCache) {
		fc.maxItems = maxItems
	}
}

func WithMaxSize(maxSize int64) fileCacheOptFn {
	return func(fc *FileCache) {
		fc.maxSize = maxSize
	}
}

func WithTTL(expiresIn time.Duration) fileCacheOptFn {
	return func(fc *FileCache) {
		fc.ttl = expiresIn
	}
}

func WithCheckInterval(checkEvery time.Duration) fileCacheOptFn {
	return func(fc *FileCache) {
		fc.checkInterval = checkEvery
	}
}

func WithPipeSize(pipeSize uint) fileCacheOptFn {
	return func(fc *FileCache) {
		fc.pipeSize = pipeSize
	}
}
