// Package request implements simple decoding of http request - url path, queries, headers and body - into golang struct
// for easier consumption, resulting in less code boilerplate.
//
// Implementation is based on OpenAPI 3 specification https://swagger.io/docs/specification/about/.
//
//	func (r *http.Request, w *http.Response) {
//		var req struct {
//			// path - requires Decoder.Path.Get to get value of path parameter
//			Id `path:"id"`
//
//			// query params
//			ExplodedIds []int `query:"id"`           // ?id=1&id=2&id=3
//			ImplodedIds []int `query:"ids,imploded"` // ?ids=1,2,3
//			Search string                            // ?search=foobar
//
//			// body
//			Client Client `body:"json"`
//		}
//
//		if err := request.Decode(r, &req); err != nil {
//			// ...
//		}
//	}
package request

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// List of supported delimiters.
const (
	QueryDelimiterPipe  = "|"
	QueryDelimiterSpace = " "
	QueryDelimiterComma = ","
)

// List of supported serialization styles.
const (
	QueryStyleForm  = "form"  // ?id=3,4,5
	QueryStyleSpace = "space" // ?id=3%204%205
	QueryStylePipe  = "pipe"  // ?id=3|4|5
	QueryStyleDeep  = "deep"  // ?id[role]=admin&id[firstName]=Alex
)

type PathConf struct {
	// Get returns value of path parameter.
	//
	//	// chi
	//	PathConf{
	//		Get: chi.URLParam
	//	}
	Get func(r *http.Request, name string) string
}

type QueryConf struct {
	// true - "?id=1&id=2&id=3", false - "?id=1,2,3"
	Exploded bool
	// one of QueryStyleForm, QueryStyleSpace, QueryStylePipe or QueryStyleDeep
	Style string
	// one of QueryDelimiterPipe, QueryDelimiterSpace, QueryDelimiterComma
	Delimiter string
}

type Decoder struct {
	Path  PathConf
	Query QueryConf
}

func NewDecoder() Decoder {
	return Decoder{
		Query: QueryConf{
			Exploded:  true,
			Style:     QueryStyleForm,
			Delimiter: QueryDelimiterComma,
		},
	}
}

var defaultDecoder = NewDecoder()

// Decode decodes http request into golang struct using defaults of OpenAPI 3 specification.
func Decode(r *http.Request, i interface{}) error {
	return defaultDecoder.Decode(r, i)
}

// Decode decodes http request into golang struct.
//
// Decoding of query params follows specification of https://swagger.io/docs/specification/serialization/#query.
//
//	// required - decoding returns error if query param is not present
//	var req struct {
//		Name string `query:",required"`
//	}
//
//	// default - ?id=1&id=2&id=3
//	var req struct {
//		Id []int // case insensitive match of field name and query parameter
//	}
//
//	// comma delimited - ?id=1,2,3
//	var req struct {
//		Id []int  `query:",form"`      // implicitly imploded
//		Ids []int `query:"id,imploded` // form by default
//	}
//
//	// pipe delimited - ?id=1|2|3
//	var req struct {
//		Id []int `query:",pipe" // implicitly imploded
//	}
//
//	// space delimited - ?id=1%202%203
//	var req struct {
//		Id []int `query:",space"` // implicitly imploded
//	}
//
//	// set different name - ?id=1,2,3
//	var req struct {
//		FilterClientIds []int `query:"id,form"` // implicitly imploded
//	}
//
// Use encoding.TextUnmarshaler to implement custom decoding.
//
// Decoding of request headers is NOT yet implemented.
//
// Decoding of request body is simple - it uses either json or xml unmarshaller:
//
//	type Entity struct {
//		Id int
//	}
//
//	// If no field tag value specified, "accept" request header is used to determine decoding. Uses json by default.
//	var req struct {
//		Entity `body:""`
//	}
//
//	// Always use json umarshalling, ignore "accept" request header:
//	var req struct {
//		Entity `body:"json"`
//	}
//
//	// Always use xml unmarshalling, ignore "accept" request header:
//	var req struct {
//		Entity `body:"xml"`
//	}
func (d Decoder) Decode(r *http.Request, i interface{}) error {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return errors.New("call of Decode passes non-pointer as second argument")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("call of Decode passes pointer to non-struct as second argument")
	}

	// query values lookup by its original and lowercased name
	const doubleSize = 2
	query := make(map[string][]string, doubleSize*len(r.URL.Query()))

	for qk, qv := range r.URL.Query() {
		lower := strings.ToLower(qk)

		if existing, ok := query[lower]; ok {
			qv = append(qv, existing...)
		}

		query[qk] = qv
		query[lower] = qv
	}

	for _, field := range flattenFields(v) {
		tagValue, ok := field.Type.Tag.Lookup("body")
		if ok {
			err := decodeBody(r, tagValue, field.Value.Addr().Interface())
			if err != nil {
				return err
			}

			continue
		}

		_, ok = field.Type.Tag.Lookup("header")
		if ok {
			err := decodeHeaders()
			if err != nil {
				return err
			}

			continue
		}

		tagValue, ok = field.Type.Tag.Lookup("path")
		if ok {
			if d.Path.Get == nil {
				continue
			}

			err := setValue(field.Value, []string{d.Path.Get(r, tagValue)})
			if err != nil {
				return fmt.Errorf("path '%s': %w", tagValue, err)
			}
		}

		// query params
		err := decodeQuery(d.Query, field.Value, field.Type, query)
		if err != nil {
			return err
		}
	}

	return nil
}

type field struct {
	Value reflect.Value
	Type  reflect.StructField
}

// flattenFields flattens all fields of struct, the following fields are not flattened:
// - fields having "body" field tag;
// - fields having "query" field tag with "deep" serialization;
// - fields having encoding.TextUnmarshaler interface.
func flattenFields(v reflect.Value) []field {
	ft := v.Type()

	fields := make([]field, 0, ft.NumField())

	for i := 0; i < ft.NumField(); i++ {
		sfv := v.Field(i)
		sft := ft.Field(i)

		// NOTE: ignore unexported fields in struct.
		if !sft.IsExported() {
			continue
		}

		if _, ok := sfv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			fields = append(fields, field{Value: sfv, Type: sft})
			continue
		}

		if sfv.Kind() == reflect.Struct {
			deepQueryOrBody := func() bool {
				for _, s := range strings.Split(sft.Tag.Get("query"), ",") {
					if s == "deep" {
						return true
					}
				}

				_, ok := sft.Tag.Lookup("body")

				return ok
			}()

			if deepQueryOrBody {
				fields = append(fields, field{Value: sfv, Type: sft})
			} else {
				fields = append(fields, flattenFields(sfv)...)
			}
		} else {
			fields = append(fields, field{Value: sfv, Type: sft})
		}
	}

	return fields
}

type fieldConf struct {
	name     string // query name
	style    string // serialization style
	exploded bool   // whether exploded values
	required bool
}

func parseFieldTag(queryConf QueryConf, s string) fieldConf {
	conf := fieldConf{exploded: queryConf.Exploded, style: queryConf.Style}
	confString := strings.SplitN(s, ",", 2) // nolint: gomnd

	conf.name = strings.TrimSpace(confString[0])

	if len(confString) == 1 {
		return conf
	}

	for _, v := range strings.Split(confString[1], ",") {
		switch s := strings.TrimSpace(v); s {
		case "required":
			conf.required = true
		case "exploded":
			conf.exploded = true
		case "imploded":
			conf.exploded = false
		case QueryStyleForm, QueryStylePipe, QueryStyleSpace:
			conf.style = s
			// implicitly implode if style is specified
			conf.exploded = false
		case QueryStyleDeep:
			conf.style = s
		}
	}

	return conf
}

func parseQueryValuesDeep(name string, query map[string][]string) map[string][]string {
	values := map[string][]string{}

	for k := range query {
		if strings.HasPrefix(k, name+"[") {
			values[k[len(name)+1:len(k)-1]] = query[k]
		}
	}

	return values
}

// parseQueryValues parses query parameters as defined in field tag.
func parseQueryValues(conf fieldConf, query map[string][]string) ([]string, bool) {
	values, ok := query[conf.name]
	if !ok {
		return nil, false
	}

	if conf.exploded {
		return values, true
	}

	// imploded - take first value, ignore remaining
	first := values[0]

	var del string

	switch conf.style {
	default:
		del = QueryDelimiterComma
	case QueryStyleSpace:
		del = QueryDelimiterSpace
	case QueryStylePipe:
		del = QueryDelimiterPipe
	}

	return strings.Split(first, del), true
}

func decodeBody(r *http.Request, fieldTag string, i interface{}) error {
	accept := r.Header.Get("accept")

	if fieldTag == "json" || fieldTag == "" && strings.HasPrefix(accept, "application/json") {
		err := json.NewDecoder(r.Body).Decode(i)
		if err != nil {
			return err
		}
	} else if fieldTag == "xml" || fieldTag == "" && strings.HasPrefix(accept, "application/xml") {
		err := xml.NewDecoder(r.Body).Decode(i)
		if err != nil {
			return err
		}
	}

	return nil
}

func decodeHeaders() error {
	return errors.New("unmarshaling header is not implemented")
}

func decodeQuery(queryConf QueryConf, fv reflect.Value, ft reflect.StructField, query map[string][]string) error {
	conf := parseFieldTag(queryConf, ft.Tag.Get("query"))

	if conf.name == "" {
		// use lowercased field name
		conf.name = strings.ToLower(ft.Name)
	}

	switch {
	default:
		qv, ok := parseQueryValues(conf, query)
		if !ok {
			if conf.required {
				return fmt.Errorf("query param '%s' is required", conf.name)
			}

			if len(qv) == 0 {
				return nil
			}
		}

		if err := setValue(fv, qv); err != nil {
			return fmt.Errorf("query param '%s': %w", conf.name, err)
		}
	case conf.name == "-":
		return nil
	case conf.style == QueryStyleDeep:
		qv := parseQueryValuesDeep(conf.name, query)
		if conf.required && len(qv) == 0 {
			return fmt.Errorf("query param '%s' is required", conf.name)
		}

		if err := setDeepValue(queryConf, fv, qv); err != nil {
			return fmt.Errorf("query param '%s': %w", conf.name, err)
		}
	}

	return nil
}

func setValue(rv reflect.Value, values []string) error {
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}

		rv = rv.Elem()
	}

	const bitsPerByte = 8

	bitSize := func() int { return int(rv.Type().Size()) * bitsPerByte }

	value := values[0]

	if e, ok := rv.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return e.UnmarshalText([]byte(value))
	}

	switch kind := rv.Kind(); kind {
	default:
		return fmt.Errorf("unknown type: %s", kind)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}

		rv.SetBool(v)
	case reflect.String:
		rv.SetString(value)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v, err := strconv.ParseUint(value, 10, bitSize()) // nolint: gomnd
		if err != nil {
			return err
		}

		rv.SetUint(v)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		v, err := strconv.ParseInt(value, 10, bitSize()) // nolint: gomnd
		if err != nil {
			return err
		}

		rv.SetInt(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, bitSize())
		if err != nil {
			return err
		}

		rv.SetFloat(v)
	case reflect.Complex64, reflect.Complex128:
		v, err := strconv.ParseComplex(value, bitSize())
		if err != nil {
			return err
		}

		rv.SetComplex(v)
	case reflect.Slice:
		t := rv.Type()

		if t.Elem().Kind() == reflect.Uint8 {
			rv.SetBytes([]byte(value))
			break
		}

		slice := reflect.MakeSlice(t, 0, len(values))

		if len(values) > 0 {
			for _, value := range values {
				v := reflect.New(t.Elem()).Elem()
				if err := setValue(v, []string{value}); err != nil {
					return err
				}

				slice = reflect.Append(slice, v)
			}
		}

		rv.Set(slice)
	}

	return nil
}

func setDeepValue(queryConf QueryConf, rv reflect.Value, values map[string][]string) error {
	rt := rv.Type()

	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}

		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return errors.New("expected struct for deep style")
	}

	for i := 0; i < rv.NumField(); i++ {
		sfv := rv.Field(i)
		sft := rt.Field(i)

		err := decodeQuery(queryConf, sfv, sft, values)
		if err != nil {
			return err
		}
	}

	return nil
}
