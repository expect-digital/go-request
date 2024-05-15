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

  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Chi

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.expect.digital/request"
)

func main() {
	decode := request.NewDecoder(
		request.PathValue(
			func(r *http.Request, name string) string {
				return chi.URLParam(r, name)
			},
		),
	).Decode

	r := chi.NewRouter()

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `path:"id"`
		}

		if err := decode(r, &req); err != nil {
			return
		}
	})

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
```

### Gorilla

```go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.expect.digital/request"
)

func main() {
	decode := request.NewDecoder(
		request.PathValue(func(r *http.Request, name string) string {
			return mux.Vars(r)[name]
		}),
	).Decode

	r := mux.NewRouter()

	r.Path("/{id}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `path:"id"`
		}

		if err := decode(r, &req); err != nil {
			return
		}
	})

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
```

### Gin

We advise using Gin binding [implementation](https://gin-gonic.com/docs/examples/bind-uri/).

Example of using the package in Gin:

```go
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.expect.digital/request"
)

func main() {
	r := gin.Default()

	r.GET("/:id", func(c *gin.Context) {
		decode := request.NewDecoder(
			request.PathValue(func(r *http.Request, name string) string {
				return c.Param(name)
			}),
		).Decode

		var req struct {
			ID int `path:"id"`
		}

		if err := decode(c.Request, &req); err != nil {
			return
		}
	})

	log.Fatal(r.Run())
}
```
