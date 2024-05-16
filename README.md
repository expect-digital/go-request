# request [![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](https://pkg.go.dev/go.expect.digital/request) ![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/expect-digital/go-request/workflow.yml) ![GitHub](https://img.shields.io/github/license/expect-digital/go-request)

godoc [go.expect.digital/request](https://pkg.go.dev/go.expect.digital/request)

Package request simplifies the decoding of HTTP requests (REST API) into Go structs for easier consumption.
It implements decoding based on the [OpenAPI 3.1](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md) specification.

In general, it is better to use code generation from the API specification,
e.g. OpenAPI spec to a server code in Golang. However, it's not always possible due to certain constraints.

Key Features:

- Decodes path parameters, query parameters, request headers (not yet implemented), and request body.
- Supports different query parameter styles: form, space-delimited, pipe-delimited,
  and deep (nested) objects.
- Allows customization of field names, required parameters, and decoding behavior through struct tags.
- Handles different body content types (JSON, XML) based on the Accept header or a specified field tag.

## Reading path value

By default, the Decoder reads a path value using [request.PathValue](https://pkg.go.dev/net/http#Request.PathValue).

Declare once and re-use in handlers.

### net/http

```go
package main

import (
  "net/http"

  "go.expect.digital/request"
)

func main() {
  http.HandleFunc("/{id}", func (w http.ResponseWriter, r *http.Request) {
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

We advise using [Gin data binding](https://gin-gonic.com/docs/examples/bind-uri/) implementation.

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
