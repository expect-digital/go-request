package request

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestDecodePointerToStruct(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	assert.EqualError(t, Decode(r, struct{}{}), "call of Decode passes non-pointer as second argument")

	var i int

	assert.EqualError(t, Decode(r, &i), "call of Decode passes pointer to non-struct as second argument")
}

func TestDecodeQueryBool(t *testing.T) {
	assert.NoError(t, quick.Check(func(v bool) bool {
		var req struct {
			Value bool `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatBool(v))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQuerySlice(t *testing.T) {
	assert.NoError(t, quick.Check(func(v []string) bool {
		var req struct {
			Value []string `query:"value"`
		}

		queries := url.Values{}
		for i := range v {
			queries.Add("value", v[i])
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return assert.ElementsMatch(t, v, req.Value)
	}, nil))
}

func TestDecodeQueryString(t *testing.T) {
	assert.NoError(t, quick.Check(func(v string) bool {
		var req struct {
			Value string `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", v)

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryInt8(t *testing.T) {
	assert.NoError(t, quick.Check(func(v int8) bool {
		var req struct {
			Value int8 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.Itoa(int(v)))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryInt16(t *testing.T) {
	assert.NoError(t, quick.Check(func(v int16) bool {
		var req struct {
			Value int16 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.Itoa(int(v)))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryInt32(t *testing.T) {
	assert.NoError(t, quick.Check(func(v int32) bool {
		var req struct {
			Value int32 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.Itoa(int(v)))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryInt64(t *testing.T) {
	assert.NoError(t, quick.Check(func(v int64) bool {
		var req struct {
			Value int64 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatInt(v, 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryInt(t *testing.T) {
	assert.NoError(t, quick.Check(func(v int) bool {
		var req struct {
			Value int `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.Itoa(v))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryUint8(t *testing.T) {
	assert.NoError(t, quick.Check(func(v uint8) bool {
		var req struct {
			Value uint8 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatUint(uint64(v), 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryUint16(t *testing.T) {
	assert.NoError(t, quick.Check(func(v uint16) bool {
		var req struct {
			Value uint16 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatUint(uint64(v), 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryUint32(t *testing.T) {
	assert.NoError(t, quick.Check(func(v uint32) bool {
		var req struct {
			Value uint32 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatUint(uint64(v), 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryUint64(t *testing.T) {
	assert.NoError(t, quick.Check(func(v uint64) bool {
		var req struct {
			Value uint64 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatUint(v, 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryUint(t *testing.T) {
	assert.NoError(t, quick.Check(func(v uint) bool {
		var req struct {
			Value uint `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatUint(uint64(v), 10))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryFloat32(t *testing.T) {
	assert.NoError(t, quick.Check(func(v float32) bool {
		var req struct {
			Value float32 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatFloat(float64(v), 'f', -1, 32))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryFloat64(t *testing.T) {
	assert.NoError(t, quick.Check(func(v float64) bool {
		var req struct {
			Value float64 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatFloat(v, 'f', -1, 64))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryComplex64(t *testing.T) {
	assert.NoError(t, quick.Check(func(v complex64) bool {
		var req struct {
			Value complex64 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatComplex(complex128(v), 'f', -1, 64))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryComplex128(t *testing.T) {
	assert.NoError(t, quick.Check(func(v complex128) bool {
		var req struct {
			Value complex128 `query:"value"`
		}

		queries := url.Values{}
		queries.Set("value", strconv.FormatComplex(v, 'f', -1, 128))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return req.Value == v
	}, nil))
}

func TestDecodeQueryStringSliceImploded(t *testing.T) {
	assert.NoError(t, quick.Check(func(v []string) bool {
		var req struct {
			Value []string `query:"value,imploded"`
		}

		// remove all commas
		for i := range v {
			v[i] = strings.ReplaceAll(v[i], ",", "")
		}

		queries := url.Values{}
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, QueryDelimiterComma))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return assert.ElementsMatch(t, v, req.Value)
	}, nil))
}

func TestDecodeQueryStringSliceExpanded(t *testing.T) {
	assert.NoError(t, quick.Check(func(v []string) bool {
		var req struct {
			Default []string `query:"value"`
			Value   []string `query:"value,expanded"`
		}

		queries := url.Values{}
		for i := range v {
			queries.Add("value", v[i])
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return assert.ElementsMatch(t, v, req.Value) && assert.ElementsMatch(t, v, req.Default)
	}, nil))
}

func TestDecodeQueryStringSliceSpace(t *testing.T) {
	assert.NoError(t, quick.Check(func(v []string) bool {
		var req struct {
			Value []string `query:"value,space"`
		}

		// remove all delimiters
		for i := range v {
			v[i] = strings.ReplaceAll(v[i], QueryDelimiterSpace, "")
		}

		queries := url.Values{}
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, QueryDelimiterSpace))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return assert.ElementsMatch(t, v, req.Value)
	}, nil))
}

func TestDecodeQueryStringSlicePipe(t *testing.T) {
	assert.NoError(t, quick.Check(func(v []string) bool {
		var req struct {
			Value []string `query:"value,pipe"`
		}

		for i := range v {
			v[i] = strings.ReplaceAll(v[i], QueryDelimiterPipe, "")
		}

		queries := url.Values{}
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, QueryDelimiterPipe))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		assert.NoError(t, Decode(r, &req))
		return assert.ElementsMatch(t, v, req.Value)
	}, nil))
}

func TestDecodeQueryStringSliceEmpty(t *testing.T) {
	var req struct {
		Fields []string
	}

	r := httptest.NewRequest(http.MethodGet, "/?fields=", nil)

	assert.NoError(t, Decode(r, &req))
	assert.Equal(t, []string{""}, req.Fields)
}

func TestDecodeQueryOptional(t *testing.T) {
	var req struct {
		Field bool `query:"field"`
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)

	assert.NoError(t, Decode(r, &req))
	assert.False(t, req.Field)
}

func TestDecodeQueryRequired(t *testing.T) {
	var req struct {
		Field bool `query:"field,required"`
	}

	queries := url.Values{}

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	assert.EqualError(t, Decode(r, &req), "query param 'field' is required")
}

func TestDecodeQueryFieldName(t *testing.T) {
	type req struct {
		FieldOne   string
		FieldTwo   string `query:",required"`
		FieldThree []string
	}

	expected := req{
		FieldOne:   "foo",
		FieldTwo:   "bar",
		FieldThree: []string{"fuzz", "bazz"},
	}

	queries := url.Values{}
	queries.Set("fIeLdOnE", expected.FieldOne)
	queries.Set("fieldTwo", expected.FieldTwo)
	queries.Add("fieldthree", "fuzz")
	queries.Add("FIELDTHREE", "bazz")

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	var actual req

	assert.NoError(t, Decode(r, &actual))
	assert.Equal(t, expected.FieldOne, actual.FieldOne)
	assert.Equal(t, expected.FieldTwo, actual.FieldTwo)
	assert.ElementsMatch(t, expected.FieldThree, actual.FieldThree)
}

func TestDecodeQueryIgnore(t *testing.T) {
	var req struct {
		Field string `query:"-"`
	}

	queries := url.Values{}
	queries.Set("field", "foobar")

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	assert.NoError(t, Decode(r, &req))
	assert.Empty(t, req.Field)
}

func TestDecodeQueryDeep(t *testing.T) {
	type Filter struct {
		Search string `query:"find"`
		Gt     byte
	}

	assert.NoError(t, quick.Check(func(v Filter) bool {
		query := url.Values{}
		query.Set("filter[find]", v.Search)
		query.Set("filter[gt]", strconv.Itoa(int(v.Gt)))

		r := httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)

		var req struct {
			Filter `query:",deep"`
		}

		return assert.NoError(t, Decode(r, &req)) && assert.Equal(t, v, req.Filter)
	}, nil))
}

func TestDecodeJSONBody(t *testing.T) {
	var req struct {
		Body struct {
			Id int
		} `body:"json"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":9}`))

	assert.NoError(t, Decode(r, &req))
	assert.Equal(t, 9, req.Body.Id)
}

func TestDecodeXMLBody(t *testing.T) {
	var req struct {
		Body struct {
			Id int
		} `body:"xml"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`<Body><Id>1</Id></Body>`))

	assert.NoError(t, Decode(r, &req))
	assert.Equal(t, 1, req.Body.Id)
}

func TestDecoder_DecodePath(t *testing.T) {
	assert.NoError(t, quick.Check(func(id int) bool {
		var req struct {
			ClientId int `path:"id"`
		}

		r := httptest.NewRequest(http.MethodGet, "/clients/"+strconv.Itoa(id), nil)

		dec := NewDecoder()
		// assume we return path parameter from path like /clients/{id}
		dec.Path.Get = func(r *http.Request, name string) string {
			v := strings.Split(r.URL.Path, "/")
			return v[len(v)-1]
		}

		return assert.NoError(t, dec.Decode(r, &req)) && assert.Equal(t, id, req.ClientId)
	}, nil))
}
