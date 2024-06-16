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
