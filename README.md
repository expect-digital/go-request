# request [![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](https://pkg.go.dev/go.expect.digital/request) ![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/expect-digital/go-request/check.yml) ![GitHub](https://img.shields.io/github/license/expect-digital/go-request) [![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11144/badge)](https://www.bestpractices.dev/projects/11144)

Package `request` simplifies decoding HTTP request data (like path parameters, query strings, and request bodies) into Go structs. It leverages struct tags based on the [OpenAPI 3.1](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md) specification to reduce boilerplate code and make request handling cleaner and more declarative.

While code generation from an API specification is often preferred, this package is useful in scenarios where that's not feasible.

## Key Features

- **Declarative Decoding:** Use `oas` struct tags to define how request data maps to your struct fields.
- **Multiple Data Sources:** Decode data from URL path parameters, query strings, and the request body. (Header decoding is not yet implemented).
- **Flexible Query Parameters:** Supports various query parameter styles defined in the OpenAPI spec:
  - `form` (e.g., `id=3,4,5` or `id=3&id=4`)
  - `spaceDelimited` (e.g., `id=3 4 5`)
  - `pipeDelimited` (e.g., `id=3|4|5`)
  - `deepObject` for nested objects.
- **Content-Type Aware:** Automatically decodes JSON or XML request bodies based on the `Content-Type` header, or can be forced via a struct tag.
- **Framework Agnostic:** Works with the standard `net/http` library and can be easily integrated with popular frameworks like Chi, Gorilla Mux, and Gin.
- **Customizable:** Allows overriding default behaviors for path parameter extraction and query parsing.

## Requirements

Golang 1.24+ is required.

## Installation

```sh
go get go.expect.digital/request
```

## Usage

The core of the package is the `Decode` function, which takes an `*http.Request` and a pointer to a struct, and populates the struct's fields based on the `oas` tags.

### Basic Example (`net/http`)

Here's a simple handler that uses `request.Decode` to extract a path parameter and a query parameter.

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"go.expect.digital/request"
)

func main() {
	http.HandleFunc("GET /{id}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `oas:"id,path"`
			Format string `oas:"format,query"`
		}

		if err := request.Decode(r, &req); err != nil {
			// handle error
			return
		}

		fmt.Fprintf(w, "ID: %d, Format: %s", req.ID, req.Format)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

To test this, you can run the server and make a request:
`curl "http://localhost:8080/123?format=json"`

### The `oas` Struct Tag

The `oas` tag controls how a field is populated. It's a comma-separated string with the following format:

`oas:"<name>,<origin>,[options...]"`

- **`name`**: The name of the parameter in the request (e.g., the path parameter name, the query key).
  - **Default**: If empty or omitted, the lowercase field name is used (e.g., `FieldName` becomes `fieldname`).
- **`origin`**: Where to find the data. Must be one of:
  - `path`: URL path parameter.
  - `query`: URL query string parameter.
  - `body`: Request body.
  - `header`: Request header (not yet implemented).
  - **Default**: If `origin` is not specified, it defaults to `query`.
- **`options`** (optional): Additional decoding options.
  - `required`: The request is considered invalid if this parameter is missing.
  - For `query`: `form`, `spaceDelimited`, `pipeDelimited`, `deepObject`, `explode`, `implode`.
    - **Default**: If no style is specified, `form` with `explode` is used.
  - For `body`: `json`, `xml` to force a specific format.
    - **Default**: If no format is specified, the `Content-Type` header is used to determine the decoder (`application/json` for JSON, `application/xml` for XML).

### Framework Integrations

The package is designed to be flexible. By default, it uses `r.PathValue` (available in Go 1.22+) for path parameters. For other routers, you can provide a custom path value function.

#### Chi

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.expect.digital/request"
)

func main() {
	// Create a decoder that knows how to get path params from Chi.
	decode := request.NewDecoder(request.PathValue(chi.URLParam)).Decode

	r := chi.NewRouter()
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `oas:"id,path"`
		}

		if err := decode(r, &req); err != nil {
			// handle error
			return
		}
		// ...
	})

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
```

#### Gorilla Mux

```go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.expect.digital/request"
)

func main() {
	// Create a decoder that knows how to get path params from Gorilla Mux.
	decode := request.NewDecoder(
		request.PathValue(func(r *http.Request, name string) string {
			return mux.Vars(r)[name]
		}),
	).Decode

	r := mux.NewRouter()
	r.Path("/{id}").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `oas:"id,path"`
		}

		if err := decode(r, &req); err != nil {
			// handle error
			return
		}
		// ...
	})

	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
```

#### Gin

While Gin has its own data binding, you can still use this package if you prefer.

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
		// Create a decoder that knows how to get path params from Gin.
		decode := request.NewDecoder(
			request.PathValue(func(r *http.Request, name string) string {
				return c.Param(name)
			}),
		).Decode

		var req struct {
			ID int `oas:"id,path"`
		}

		if err := decode(c.Request, &req); err != nil {
			// handle error
			return
		}
		// ...
	})

	log.Fatal(r.Run())
}
```
