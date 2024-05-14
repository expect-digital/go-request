# request [![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](https://pkg.go.dev/go.expect.digital/request) ![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/expect-digital/go-request/workflow.yml) ![GitHub](https://img.shields.io/github/license/expect-digital/go-request)

Package request implements simple decoding of http request - queries, headers and body - into golang struct
for easier consumption, resulting in less code boilerplate.

godoc [go.expect.digital/request](https://pkg.go.dev/go.expect.digital/request)

## Reading path value

By default, Decoder reads a path value using [Request.Pathvalue](https://pkg.go.dev/net/http#Request.PathValue).

Declare once and re-use in handlers.

### http

```go
package main

import (
  "net/http"

  "go.expect.digital/request"
)

func main() {
  http.HandleFunc("/{id}", func (r *http.Request, w http.Response) {
    var req struct {
      ID int `path:"id"`
    }

    if err := request.Decode(r, &req); err != nil {
      return
    }
  })

  http.ListenAndServe(":8080", nil)
}
```

### Chi

```go
package main

import (
  "net/http"

  "github.com/go-chi/chi/v5"
)

func main() {
  decode := NewDecoder(
    WithPathValue(
      func (r *http.Request, name string) string {
        return chi.URLParam(r, name)
      },
    ),
  ).Decode

  r := chi.NewRouter()
  r.Get("/{id}", func handler(r *http.Request, w http.Response) {
    var req struct {
      ID int `path:"id"`
    }

    if err := decode(r, &req); err != nil {
      return
    }
  })

  http.ListenAndServe(":8080", r)
}


```

### Gorilla

```go
package main

import (
  "net/http"

  "github.com/gorilla/mux"
)

func main() {
  decode := NewDecoder(
    WithPathValue(func (r *http.Request, name string) string {
      return mux.Vars(r)[name]
    })
  ).Decode

  http.HandleFunc("/{id}", func (r *http.Request, w http.Response) {
    var req struct {
      ID int `path:"id"`
    }

    if err := decode(r, &req); err != nil {
      return
    }
  })

  http.ListenAndServe(":8080", nil)
}
```

### Gin

```go
import (
  "net/http"

  "github.com/gin-gonic/gin"
)

func main() {
  r := gin.Default()

  r.GET("/:id", func (c *gin.Context) {
    decoder := NewDecoder(
      WithPathValue(func (r *http.Request, name string) string {
        return c.Param(name)
      })
    )

    if err := decode(r, &req); err != nil {
      return
    }
  })

  r.Run()
}

```
