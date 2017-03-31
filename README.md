# DaZeus go client library

To get started, see the following example:

```go
package main

import (
    "os"

    "github.com/dazeus/dazeus-go"
)

func main() {
    connStr := "unix:/tmp/dazeus.sock"
    if len(os.Args) > 1 {
        connStr = os.Args[1]
    }

    dz, err := dazeus.Connect(connStr)
    if err != nil {
        panic(err)
    }

    _, err = dz.Subscribe(dazeus.EventPrivMsg, func(evt dazeus.Event, replier dazeus.Replier) {
        replier(evt.Params[3], dazeus.ReplyMessage, false)
    })
    if err != nil {
        panic(err)
    }

    dz.Listen()
}
```
