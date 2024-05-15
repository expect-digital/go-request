// Package request simplifies decoding of HTTP requests (REST APIs) into Go structs for easier consumption.
// It implements decoding based on the [OpenAPI 3.1] specification.
//
// Key Features:
//   - Decodes path parameters, query parameters, request headers (not yet implemented), and request body.
//   - Supports different query parameter styles: form (imploded/exploded), space-delimited, pipe-delimited,
//     and deep (nested objects).
//   - Allows customization of field names, required parameters, and decoding behavior through struct tags.
//   - Handles different body content types (JSON, XML) based on the Accept header or a specified field tag.
//
// [OpenAPI 3.1]: https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md
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

const (
	queryDelimiterPipe  = "|"
	queryDelimiterSpace = " "
	queryDelimiterComma = ","
)

// List of supported serialization styles.
const (
	QueryStyleForm  = "form"  // imploded "?id=3,4,5" or exploded "?id=3&id=4&id=5"
	QueryStyleSpace = "space" // imploded "?id=3%204%205" or exploded "?id=3&id=4&id=5"
	QueryStylePipe  = "pipe"  // imploded "?id=3|4|5" or exploded "?id=3&id=4&=5"
	QueryStyleDeep  = "deep"  // exploded "?id[role]=admin&id[firstName]=Alex"
)

type queryConf struct {
	// one of QueryStyleForm, QueryStyleSpace, QueryStylePipe or QueryStyleDeep
	style string
	// true - "?id=1&id=2&id=3", false - "?id=1,2,3"
	exploded bool
}

type Decoder struct {
	pathValue func(r *http.Request, name string) string
	query     queryConf
}

type Opt interface {
	apply(d *Decoder)
}

type decoderOpt struct {
	f func(d *Decoder)
}

func (o decoderOpt) apply(d *Decoder) {
	o.f(d)
}

func newOpt(f func(d *Decoder)) Opt { //nolint:ireturn
	return decoderOpt{f: f}
}

// PathValue sets a path parameter getter in [request.NewDecoder].
func PathValue(pathValue func(r *http.Request, name string) string) Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.pathValue = pathValue
	})
}

// QueryStyle lets you set query parameter style:
//   - [request.QueryStyleForm]
//   - [request.QueryStyleSpace]
//   - [request.QueryStylePipe]
//   - [request.QueryStyleDeep]
func QueryStyle(style string) Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.style = style
	})
}

// QueryExploded sets each value in a separate query parameter (e.g "?id=1&id=2"). The query delimiter is ignored.
func QueryExploded() Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.exploded = true
	})
}

// QueryImploded sets all values in a single query parameter and all values are
// separated by a delimiter (e.g "?id=1,2").
func QueryImploded() Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.exploded = false
	})
}

func NewDecoder(opts ...Opt) Decoder {
	decoder := Decoder{
		pathValue: func(r *http.Request, name string) string { return r.PathValue(name) },
		query: queryConf{
			exploded: true,
			style:    QueryStyleForm,
		},
	}

	for _, opt := range opts {
		opt.apply(&decoder)
	}

	return decoder
}

var defaultDecoder = NewDecoder()

// Decode decodes an HTTP request into a Go struct according to OpenAPI 3 specification.
func Decode(r *http.Request, i interface{}) error {
	return defaultDecoder.Decode(r, i)
}

// Decode decodes an HTTP request into Go struct.
//
// Decoding of query params follows [Query Serialization] spec.
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
//	// If no field tag value specified, "Accept" request header is used to determine decoding. Uses json by default.
//	var req struct {
//		Entity `body:""`
//	}
//
//	// Always use JSON umarshalling, ignore "Accept" request header:
//	var req struct {
//		Entity `body:"json"`
//	}
//
//	// Always use XML unmarshalling, ignore "Accept" request header:
//	var req struct {
//		Entity `body:"xml"`
//	}
//
// [Query Serialization]: https://swagger.io/docs/specification/serialization/#query
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
			err := setValue(field.Value, []string{d.pathValue(r, tagValue)})
			if err != nil {
				return fmt.Errorf("path '%s': %w", tagValue, err)
			}
		}

		// query params
		err := decodeQuery(d.query, field.Value, field.Type, query)
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

	for i := range ft.NumField() {
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

func parseFieldTag(queryConf queryConf, tag string) fieldConf {
	tag = strings.TrimSpace(tag)
	parts := strings.Split(tag, ",")

	if len(parts) <= 1 {
		return fieldConf{
			exploded: queryConf.exploded,
			style:    queryConf.style,
			name:     tag,
		}
	}

	conf := fieldConf{
		exploded: queryConf.exploded,
		style:    queryConf.style,
		name:     strings.TrimSpace(parts[0]),
	}

	for _, part := range parts[1:] {
		switch v := strings.TrimSpace(part); v {
		case "required":
			conf.required = true
		case "exploded":
			conf.exploded = true
		case "imploded":
			conf.exploded = false
		case QueryStyleForm, QueryStylePipe, QueryStyleSpace:
			conf.style = v
			// implicitly implode if style is specified
			conf.exploded = false
		case QueryStyleDeep:
			conf.style = v
		}
	}

	return conf
}

func parseQueryValuesDeep(name string, query map[string][]string) map[string][]string {
	values := map[string][]string{}

	for k := range query {
		propName, ok := strings.CutPrefix(k, name+"[")
		if !ok {
			continue
		}

		propName, ok = strings.CutSuffix(propName, "]")
		if !ok {
			continue
		}

		values[propName] = query[k]
	}

	return values
}

// parseQueryValues parses query parameters as defined in field tag.
func parseQueryValues(conf fieldConf, query map[string][]string) ([]string, bool) {
	values, ok := query[conf.name]
	if !ok {
		return nil, false
	}

	if conf.exploded || len(values) == 0 {
		return values, true
	}

	// imploded - take first value, ignore remaining
	//
	// TODO(jhorsts): OpenAPI does not specify what should happen in given instance. However,
	// picking up the last value conforms more likely with developer expectations.
	first := values[0]

	delimiter := queryDelimiterComma

	switch conf.style {
	case QueryStyleSpace:
		delimiter = queryDelimiterSpace
	case QueryStylePipe:
		delimiter = queryDelimiterPipe
	}

	return strings.Split(first, delimiter), true
}

func decodeBody(r *http.Request, fieldTag string, i interface{}) error {
	if fieldTag == "" {
		accept := strings.ToLower(r.Header.Get("Accept"))

		if strings.HasPrefix(accept, "application/json") {
			fieldTag = "json"
		} else if strings.HasPrefix(accept, "application/xml") {
			fieldTag = "xml"
		}
	}

	switch fieldTag {
	default:
		return fmt.Errorf(`want "xml" or "json", got unsupported "%s"`, fieldTag)
	case "json":
		err := json.NewDecoder(r.Body).Decode(i)
		if err != nil {
			return fmt.Errorf("decode JSON body: %w", err)
		}

		return nil
	case "xml":
		err := xml.NewDecoder(r.Body).Decode(i)
		if err != nil {
			return fmt.Errorf("decode XML body: %w", err)
		}

		return nil
	}
}

func decodeHeaders() error {
	return errors.New("unmarshaling header is not implemented")
}

func decodeQuery(queryConf queryConf, fv reflect.Value, ft reflect.StructField, query map[string][]string) error {
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
	if len(values) == 0 {
		return nil
	}

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
		if err := e.UnmarshalText([]byte(value)); err != nil {
			return fmt.Errorf("set values %v: %w", values, err)
		}

		return nil
	}

	switch kind := rv.Kind(); kind { //nolint:exhaustive
	default:
		return fmt.Errorf("unknown type: %s", kind)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err //nolint:wrapcheck
		}

		rv.SetBool(v)
	case reflect.String:
		rv.SetString(value)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		v, err := strconv.ParseUint(value, 10, bitSize())
		if err != nil {
			return err //nolint:wrapcheck
		}

		rv.SetUint(v)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		v, err := strconv.ParseInt(value, 10, bitSize())
		if err != nil {
			return err //nolint:wrapcheck
		}

		rv.SetInt(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, bitSize())
		if err != nil {
			return err //nolint:wrapcheck
		}

		rv.SetFloat(v)
	case reflect.Complex64, reflect.Complex128:
		v, err := strconv.ParseComplex(value, bitSize())
		if err != nil {
			return err //nolint:wrapcheck
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

func setDeepValue(queryConf queryConf, rv reflect.Value, values map[string][]string) error {
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

	for i := range rv.NumField() {
		sfv := rv.Field(i)
		sft := rt.Field(i)

		err := decodeQuery(queryConf, sfv, sft, values)
		if err != nil {
			return err
		}
	}

	return nil
}
