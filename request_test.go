package request

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
)

func testQuery[T comparable](t *testing.T) {
	t.Helper()

	err := quick.Check(func(v T) bool {
		var req struct {
			Value T `oas:"value,query"`
		}

		queries := make(url.Values)
		queries.Set("value", fmt.Sprint(v))

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return req.Value == v
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodePointerToStruct(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/", nil)

	want := "call of Decode passes non-pointer as second argument"

	err := Decode(r, struct{}{})
	if err == nil || err.Error() != want {
		t.Errorf(`want "%s", got "%s"`, want, err)
	}

	var i int

	want = "call of Decode passes pointer to non-struct as second argument"

	err = Decode(r, &i)
	if err == nil || err.Error() != want {
		t.Errorf(`want "%s", got "%s"`, want, err)
	}
}

func TestDecodeQuery(t *testing.T) {
	t.Parallel()

	testQuery[bool](t)
	testQuery[string](t)
	testQuery[uint8](t) // byte
	testQuery[uint16](t)
	testQuery[uint32](t)
	testQuery[uint64](t) // uint
	testQuery[int8](t)
	testQuery[int16](t)
	testQuery[int32](t)
	testQuery[int64](t) // int
	testQuery[float32](t)
	testQuery[float64](t)
	testQuery[complex64](t)
	testQuery[complex128](t)
}

func TestDecodeQuerySlice(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v []string) bool {
		var req struct {
			Value []string `oas:"value,query"`
		}

		queries := make(url.Values)
		for i := range v {
			queries.Add("value", v[i])
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return slices.Equal(v, req.Value)
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeQueryByteSlice(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v string) bool {
		var req struct {
			Value []byte `oas:"value,query"`
		}

		queries := make(url.Values)
		queries.Set("value", v)

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return string(req.Value) == v
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeQueryImploded(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v []string) bool {
		var req struct {
			Value []string `oas:"value,query,implode"`
		}

		// remove all commas
		for i := range v {
			v[i] = strings.ReplaceAll(v[i], ",", "")
		}

		queries := make(url.Values)
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, ","))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return slices.Equal(v, req.Value)
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeQueryExploded(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v []string) bool {
		var req struct {
			Default []string `oas:"value,query"`
			Value   []string `oas:"value,query,explode"`
		}

		queries := make(url.Values)
		for i := range v {
			queries.Add("value", v[i])
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return slices.Equal(v, req.Value) && slices.Equal(v, req.Default)
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeInvalidTag(t *testing.T) {
	t.Parallel()

	var req struct {
		Value []string `oas:"value,query,expanded"`
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)

	err := Decode(r, &req)
	if err == nil {
		t.Error("want error, got no error")
	}
}

func TestDecodeQuerySliceSpace(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v []string) bool {
		var req struct {
			Value []string `oas:"value,query,spaceDelimited"`
		}

		// remove all delimiters
		for i := range v {
			v[i] = strings.ReplaceAll(v[i], " ", "")
		}

		queries := make(url.Values)
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, " "))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return slices.Equal(v, req.Value)
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeQuerySlicePipe(t *testing.T) {
	t.Parallel()

	err := quick.Check(func(v []string) bool {
		var req struct {
			Value []string `oas:"value,query,pipeDelimited"`
		}

		for i := range v {
			v[i] = strings.ReplaceAll(v[i], "|", "")
		}

		queries := make(url.Values)
		if len(v) > 0 {
			queries.Set("value", strings.Join(v, "|"))
		}

		r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return slices.Equal(v, req.Value)
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeQuerySliceEmpty(t *testing.T) {
	t.Parallel()

	var req struct {
		Fields []string
	}

	r := httptest.NewRequest(http.MethodGet, "/?fields=", nil)

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	want := []string{""}
	if !slices.Equal(want, req.Fields) {
		t.Errorf("want %v, got %v", want, req.Fields)
	}
}

func TestDecodeQueryOptional(t *testing.T) {
	t.Parallel()

	var req struct {
		Field bool `oas:"field,query"`
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Field {
		t.Error("want false, got true")
	}
}

func TestDecodeQueryRequired(t *testing.T) {
	t.Parallel()

	var req struct {
		Field bool `oas:"field,query,required"`
	}

	queries := make(url.Values)

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	want := "query param 'field' is required"

	err := Decode(r, &req)
	if err == nil || err.Error() != want {
		t.Errorf(`want "%s", got "%s"`, want, err)
	}
}

func TestDecodeQueryFieldName(t *testing.T) {
	t.Parallel()

	type req struct {
		FieldOne   string
		FieldTwo   string `oas:",query,required"`
		FieldThree []string
	}

	want := req{
		FieldOne:   "foo",
		FieldTwo:   "bar",
		FieldThree: []string{"bazz", "fuzz"}, // sorted
	}

	queries := make(url.Values)
	queries.Set("fIeLdOnE", want.FieldOne)
	queries.Set("fieldTwo", want.FieldTwo)
	queries.Add("fieldthree", "fuzz")
	queries.Add("FIELDTHREE", "bazz")

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	var got req

	err := Decode(r, &got)
	if err != nil {
		t.Error(err)
	}

	if want.FieldOne != got.FieldOne {
		t.Errorf("want %s, got %s", want.FieldOne, got.FieldOne)
	}

	if want.FieldTwo != got.FieldTwo {
		t.Errorf("want %s, got %s", want.FieldTwo, got.FieldTwo)
	}

	slices.Sort(got.FieldThree)

	if !slices.Equal(want.FieldThree, got.FieldThree) {
		t.Errorf("want %v, got %s", want.FieldThree, got.FieldThree)
	}
}

func TestDecodeQueryIgnore(t *testing.T) {
	t.Parallel()

	var req struct {
		Field string `oas:"-"`
	}

	queries := make(url.Values)
	queries.Set("field", "foobar")

	r := httptest.NewRequest(http.MethodGet, "/?"+queries.Encode(), nil)

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Field != "" {
		t.Errorf("want empty, got %s", req.Field)
	}
}

func TestDecodeQueryDeep(t *testing.T) {
	t.Parallel()

	type Filter struct {
		Search string `oas:"search"`
		Gt     byte
	}

	err := quick.Check(func(v Filter) bool {
		query := make(url.Values)
		query.Set("filter[search]", v.Search)
		query.Set("filter[gt]", strconv.Itoa(int(v.Gt)))

		r := httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)

		var req struct {
			Filter `oas:"filter,query,deepObject"`
		}

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
		}

		return v == req.Filter
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

type Sort struct {
	Name string
	Asc  bool
}

func (s *Sort) UnmarshalText(text []byte) error {
	words := strings.Split(string(text), ",")
	if len(words) > 2 {
		return fmt.Errorf("incorrectly formatted sort: %s", text)
	}

	s.Name = words[0]
	s.Asc = len(words) == 1 || strings.ToLower(words[1]) == "asc"

	return nil
}

func TestDecodeUnmarshalText(t *testing.T) {
	t.Parallel()

	var req struct {
		Sort
	}

	r := httptest.NewRequest(http.MethodGet, "/?sort=name", nil)

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Name != "name" {
		t.Errorf(`want "name", got %s`, req.Name)
	}

	if !req.Asc {
		t.Error("want true, got false")
	}
}

func TestDecodeJSONBody(t *testing.T) {
	t.Parallel()

	var req struct {
		Body struct {
			ID int `json:"id"`
		} `oas:",body,json"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"id":9}`))

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Body.ID != 9 {
		t.Errorf("want 9, got %d", req.Body.ID)
	}
}

func TestDecodeXMLBody(t *testing.T) {
	t.Parallel()

	var req struct {
		Body struct {
			ID int `xml:"Id"`
		} `oas:",body,xml"`
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`<Body><Id>1</Id></Body>`))

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Body.ID != 1 {
		t.Errorf("want 1, got %d", req.Body.ID)
	}
}

func TestDecoder_DecodePath(t *testing.T) {
	t.Parallel()

	dec := NewDecoder()

	err := quick.Check(func(id int) bool {
		var req struct {
			ClientID int `oas:"id,path"`
		}

		// Path has no impact on the test. Set path value manually.
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetPathValue("id", strconv.Itoa(id))

		err := dec.Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return id == req.ClientID
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeEmbeddedStructs(t *testing.T) {
	t.Parallel()

	type Range struct {
		Start int `oas:"rangeStart,query"`
		End   int `oas:"rangeEnd,query"`
	}

	err := quick.Check(func(rangeStart, rangeEnd int) bool {
		query := make(url.Values)
		query.Set("rangeStart", strconv.Itoa(rangeStart))
		query.Set("rangeEnd", strconv.Itoa(rangeEnd))
		query.Set("sort", "name")

		r := httptest.NewRequest(http.MethodGet, "/?"+query.Encode(), nil)

		var req struct {
			Sort
			Range
		}

		err := Decode(r, &req)
		if err != nil {
			t.Log(err)
			return false
		}

		return rangeStart == req.Start &&
			rangeEnd == req.End &&
			req.Name == "name" &&
			req.Asc
	}, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestDecodeImplodeLastValue(t *testing.T) {
	t.Parallel()

	// read the last value when expected imploded query, but received exploded

	var req struct {
		Value string `oas:"value,query,implode"`
	}

	r := httptest.NewRequest(http.MethodGet, "/?value=first&value=last", nil)

	err := Decode(r, &req)
	if err != nil {
		t.Error(err)
	}

	if req.Value != "last" {
		t.Errorf(`want "last", got "%s"`, req.Value)
	}
}

func BenchmarkDecode(b *testing.B) {
	var err error

	var req struct {
		Value []string `oas:"value,query"`
		Deep  struct {
			OK bool `oas:"ok"`
		} `oas:"deep,deepObject"`
	}

	r := httptest.NewRequest(http.MethodGet, "/?value=one,two,three&deep[ok]=1", nil)

	for range b.N {
		err = Decode(r, &req)
	}

	_ = err
}
