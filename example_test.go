package request_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"go.expect.digital/request"
)

func ExampleDecode() {
	r := httptest.NewRequest(
		http.MethodPost,
		"/?filterType=pending,approved&clientId=4&filterClientIds=1|2|3",
		strings.NewReader(`{"id":1}`),
	)

	var req struct {
		// query params
		FilterType      []string `oas:"filterType,query,implode"`
		FilterClientIDs []int    `oas:"filterClientIds,query,pipeDelimited,implode"`
		ClientID        int      `oas:"clientId,query"`

		// body
		Client struct {
			ID int `json:"id"`
		} `oas:",body,json"`
	}

	_ = request.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {FilterType:[pending approved] FilterClientIDs:[1 2 3] ClientID:4 Client:{ID:1}}
}

func ExampleDecoder_Decode() {
	r := httptest.NewRequest(http.MethodPost, "/?ids=1,2,3", nil)

	var req struct {
		IDs []int
	}

	// set query values imploded "?ids=1,2,3" by default
	dec := request.NewDecoder(request.QueryImplode())

	_ = dec.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {IDs:[1 2 3]}
}
