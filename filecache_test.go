package filecache_test

import (
	"errors"
	"testing"
	"time"

	"github.com/0xc000022070/filecache"
)

func TestBasic(t *testing.T) {
	cache := filecache.New("basic-test",
		filecache.WithMaxItems(10),
		filecache.WithTTL(time.Second*3),
		filecache.WithCheckInterval(time.Second),
	)

	defer cache.Destroy()

	type BasicTestData struct {
		Why string
	}

	data := BasicTestData{
		Why: "I'm a life",
	}

	const key = "internal/basic-test.gob"

	if err := filecache.SetEncoded(cache, key, data); err != nil {
		t.Fatal(err)
	}

	ticker := time.NewTicker(time.Second)
	ticks := 0

	t.Log("The cache data should be removed after 3 seconds")

	for range ticker.C {
		if ticks >= 5 {
			t.Log("The cache data wasn't removed")
			t.Fail()

			break
		}

		ticks++
		t.Logf("tick n° %d", ticks)

		data, err := filecache.GetDecoded[BasicTestData](cache, key)
		if err != nil {
			if errors.Is(err, filecache.ErrNotFound) {
				if ticks < 3 {
					t.Log("the cache data was removed but that was not expected")
					t.Fail()
				} else {
					t.Log("as expected, the cache data was removed")
				}

				break
			}
		}

		t.Log("stored data is", data)
	}
}

func TestDestroy(t *testing.T) {
	t.Parallel()

	cache := filecache.New("destroy-test")
	defer cache.Destroy()

	if err := cache.Set("a-misty-value", []byte("I'm a misty value")); err != nil {
		t.Errorf("failed to set a value in file-based cache: %v", err)
		t.FailNow()
	}

	if err := cache.Destroy(); err != nil {
		t.Errorf("failed to destroy file-based cache: %v", err)
		t.FailNow()
	}

	cache = filecache.New("destroy-test")

	data, err := cache.Get("a-misty-value")
	if err != nil {
		if errors.Is(err, filecache.ErrNotFound) {
			t.Log("as expected, the value was removed from the cache")
			return
		}

		t.Errorf("failed to retrieve the value from the cache: %v", err)
		t.FailNow()
	}

	t.Errorf("failed to remove the value from the cache: %v", err)
	t.Logf("a-misty-value: %s", data)
	t.Fail()
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	cache := filecache.New("shutdown-test")

	if err := cache.Set("extraordinary", []byte("the extraordinary value")); err != nil {
		t.Errorf("failed to set a value in file-based cache: %v", err)
		t.FailNow()
	}

	cache.Shutdown()

	cache = filecache.New("shutdown-test")
	defer cache.Destroy()

	data, err := cache.Get("extraordinary")
	if err != nil {
		t.Errorf("failed to get the value from the cache when it was suposed to be in file-system: %v", err)
		t.FailNow()
	}

	t.Log("as expected, the value was removed from the cache")
	t.Logf("extraordinary: %s", data)
}
