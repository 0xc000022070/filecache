# File cache

A simple modern file-based cache implementation written in Go.

Inspired by [gokyle/filecache](https://github.com/gokyle/filecache).

## Install

```shell
# Requires Go 1.18 or later
 $ go get -u github.com/0xc000022070/filecache@latest
```

## API

### Examples

#### Simple

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/0xc000022070/filecache"
)

func main() {
    cache := filecache.New("showcase-1",
        filecache.WithTTL(time.Hour),
        filecache.WithMaxSize(filecache.Megabyte*100),
        filecache.WithCheckInterval(time.Minute*3),
    )

    // if you don't want to destroy it from the FS
    // use `cache.Shutdown()`.
    defer cache.Destroy()

    // Try to handle ErrToLarge, ErrInvalidKey, etc.
    err := cache.Set("my-showcase-content.txt", []byte("heeey!"))
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    content, err := cache.Get("my-showcase-content.txt")
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    fmt.Fprintf(os.Stdout, "%s\n", content) // heeey!
}

```

#### Using generics

```go
package main

import (
    "fmt"
    "os"
    "time"

    "github.com/0xc000022070/filecache"
)

func main() {
    cache := filecache.New("showcase-2",
        filecache.WithTTL(time.Minute*5),
        filecache.WithMaxSize(filecache.Gigabyte),
        filecache.WithCheckInterval(time.Minute),
    )

    // It will not be destroyed from the FS, so it can be used in another program call.
    defer cache.Shutdown()

    type UserData struct {
        Name  string
        Email string
    }

    users := []UserData{
        {
            Name:  "John Doe",
            Email: "john@doe.com",
        },
        {
            Name:  "Jane Doe",
            Email: "jane@doe.com",
        },
    }

    for _, user := range users {
        key := fmt.Sprintf("user-%s", user.Email)

        // (encoded as glob)
        if err := filecache.SetEncoded(cache, key, user); err != nil {
            fmt.Fprintln(os.Stderr, err)
        }

        user2, err := filecache.GetDecoded[UserData](cache, key)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }

        fmt.Fprintf(os.Stdout, "%s <%s>\n", user2.Name, user2.Email)
    }

    if !cache.Exists(fmt.Sprintf("user-%s", users[0].Email)) {
        fmt.Fprintln(os.Stderr, "user not found")
        os.Exit(1)
    }
}
```

## License

This project is licensed under the ISC License - see the [LICENSE](./LICENSE.md) file for details.
