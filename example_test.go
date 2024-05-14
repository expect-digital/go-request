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
		FilterType      []string `query:"filterType,imploded"`
		FilterClientIDs []int    `query:"filterClientIds,pipe,imploded"`
		ClientID        int      `qyery:"clientId"`

		// body
		Client struct {
			ID int `json:"id"`
		} `body:"json"`
	}

	_ = request.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {FilterType:[pending approved] FilterClientIDs:[1 2 3] ClientID:4 Client:{ID:1}}
}

func ExampleDecoder_Decode() {
	r := httptest.NewRequest(http.MethodPost, "/?ids=1,2,3", nil)

	var req struct {
		Ids []int
	}

	// set query values imploded "?ids=1,2,3" by default
	dec := request.NewDecoder(request.QueryImploded())

	_ = dec.Decode(r, &req)

	fmt.Printf("%+v\n", req)
	// Output:
	// {Ids:[1 2 3]}
}
