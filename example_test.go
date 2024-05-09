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
		"/?filterType=pending,approved&clientId=4&filterClientids=1|2|3",
		strings.NewReader(`{"id":1}`),
	)

	var req struct {
		// query params
		FilterType      []string `query:"filterType,imploded"`
		FilterClientIDs []int    `query:",pipe,imploded"`
		ClientID        int

		// body
		Client struct {
			ID int
		} `body:"json"`
	}

	_ = request.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {FilterType:[pending approved] ClientId:4 FilterClientIds:[1 2 3] Client:{Id:1}}
}

func ExampleDecoder_Decode() {
	r := httptest.NewRequest(http.MethodPost, "/?ids=1,2,3", nil)

	var req struct {
		Ids []int
	}

	dec := request.NewDecoder()
	// set query values imploded "?ids=1,2,3" by default
	dec.Query.Exploded = false

	_ = dec.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {Ids:[1 2 3]}
}
