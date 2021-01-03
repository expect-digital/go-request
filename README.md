# request

[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/expect-digital/go-request)

Package request implements simple decoding of http request - queries, headers and body - into golang struct
for easier consumption, resulting in less code boilerplate.

Implementation is based on OpenAPI 3 specification [https://swagger.io/docs/specification/about/](https://swagger.io/docs/specification/about/).

```go
func (r *http.Request, w *http.Response) {
	var req struct {
		// query params
		ExplodedIds []int `query:"id"`           // ?id=1&id=2&id=3
		ImplodedIds []int `query:"ids,imploded"` // ?ids=1,2,3
		Search string                            // ?search=foobar

		// body
		Client Client `body:"json"`
	}

	if err := request.Decode(r, &req); err != nil {
		// ...
	}
}
```

## Functions

### func [Decode](/request.go#L76)

`func Decode(r *http.Request, i interface{}) error`

Decode decodes http request into golang struct using defaults of OpenAPI 3 specification.

```golang
package main

import (
	"fmt"
	"go.expect.digital/request"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	r := httptest.NewRequest(
		http.MethodPost,
		"/?filterType=pending,approved&clientId=4&filterClientids=1|2|3",
		strings.NewReader(`{"id":1}`),
	)

	var req struct {
		// query params
		FilterType      []string `query:"filterType,imploded"`
		ClientId        int
		FilterClientIds []int `query:",pipe,imploded"`

		// body
		Client struct {
			Id int
		} `body:"json"`
	}

	_ = request.Decode(r, &req)

	fmt.Printf("%+v\n", req)
}

```

 Output:

```
{FilterType:[pending approved] ClientId:4 FilterClientIds:[1 2 3] Client:{Id:1}}
```

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
