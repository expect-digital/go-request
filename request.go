// Package request simplifies the decoding of HTTP requests (REST API) into Go structs for easier consumption.
// It implements decoding based on the [OpenAPI 3.1] specification.
//
// In general, it is better to use code generation from the API specification,
//
// Key Features:
//   - Decodes path parameters, query parameters, request headers (not yet implemented), and request body.
//   - Supports different query parameter styles: form, space-delimited, pipe-delimited,
//     and deep (nested) objects.
//   - Allows customization of field names, required parameters, and decoding behavior through struct tags.
//   - Handles different body content types (JSON, XML) based on the Accept header or a specified field tag.
//
// When using Go standard packages, the code might look something like:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		var (
//			err error
//			req struct {
//				ID     int     // path value
//				Expand *string // query param
//			}
//		)
//
//		if req.ID, err = strconv.Atoi(r.PathValue("id")); err != nil {
//			// handle error
//			return
//		}
//
//		if expand := r.URL.Query().Get("expand"); expand != "" {
//			req.Expand = &expand
//		}
//	}
//
// The request package allows to bind data using field tags. Effectively,
// reducing the boilerplate code significantly.
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		var req struct {
//			ID     int     `oas:"id,path"`      // path value
//			Expand *string `oas:"expand,query"` // query param
//		}
//
//		err := Decode(r, &req)
//		if err != nil {
//			// handle error
//			return
//		}
//	}
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
	"slices"
	"strconv"
	"strings"
)

const fieldTagName = "oas"

const (
	originQuery  = "query"
	originBody   = "body"
	originPath   = "path"
	originHeader = "header"
)

// List of supported serialization styles.
const (
	QueryStyleForm           = "form"           // imploded "?id=3,4,5" or exploded "?id=3&id=4&id=5"
	QueryStyleSpaceDelimited = "spaceDelimited" // imploded "?id=3%204%205" or exploded "?id=3&id=4&id=5"
	QueryStylePipeDelimited  = "pipeDelimited"  // imploded "?id=3|4|5" or exploded "?id=3&id=4&=5"
	QueryStyleDeepObject     = "deepObject"     // exploded "?id[role]=admin&id[firstName]=Alex"
)

// queryConf contains default configuration for Decoder.
type queryConf struct {
	// one of QueryStyleForm, QueryStyleSpace, QueryStylePipe or QueryStyleDeep
	style string
	// true - "?id=1&id=2&id=3", false - "?id=1,2,3"
	exploded bool
}

// Decoder decodes (binds) [net/http.Request] data into Go struct.
type Decoder struct {
	pathValue func(r *http.Request, name string) string
	query     queryConf
}

// Opt allows to override default [request.Decoder] options.
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

// PathValue allows to override default path parameter getter in [request.NewDecoder].
func PathValue(pathValue func(r *http.Request, name string) string) Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.pathValue = pathValue
	})
}

// QueryStyle allows to override default query parameter style:
//   - [request.QueryStyleForm]
//   - [request.QueryStyleSpaceDelimited]
//   - [request.QueryStylePipeDelimited]
//   - [request.QueryStyleDeepObject]
func QueryStyle(style string) Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.style = style
	})
}

// QueryExplode sets each value in a separate query parameter (e.g "?id=1&id=2"). The query delimiter is ignored.
func QueryExplode() Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.exploded = true
	})
}

// QueryImplode sets all values in a single query parameter and all values are
// separated by a delimiter (e.g "?id=1,2").
func QueryImplode() Opt { //nolint:ireturn
	return newOpt(func(d *Decoder) {
		d.query.exploded = false
	})
}

// NewDecoder returns a new decoder to decode [net/http.Request] data into Go struct.
//
// By default:
//   - the decoder reads path value using
//     https://pkg.go.dev/net/http#Request.PathValue. Override with [request.PathValue] option.
//   - the decoder uses exploded query parameters. Override with [request.QueryImplode]
//     or [request.QueryExplode] option.
//   - the decoder uses [request.QueryStyleForm] query parameter style. Override with [request.QueryStyle] option.
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
func Decode(r *http.Request, i any) error {
	return defaultDecoder.Decode(r, i)
}

// Decode decodes an HTTP request into Go struct.
//
// Decoding of query params conforms to the [Query Serialization] spec.
//
//	// required - decoding returns error if query param is not present
//	var req struct {
//		Name string `oas:",query,required"`
//	}
//
//	// default - ?id=1&id=2&id=3
//	var req struct {
//		ID []int // case insensitive match of field name and query parameter
//	}
//
//	// comma delimited - ?id=1,2,3
//	var req struct {
//		ID []int  `oas:",query,form"`     // implicitly implode
//		IDs []int `oas:"id,query,implode` // form by default
//	}
//
//	// pipe delimited - ?id=1|2|3
//	var req struct {
//		ID []int `oas:",query,pipeDelimited" // implicitly implode
//	}
//
//	// space delimited - ?id=1%202%203
//	var req struct {
//		ID []int `oas:",query,spaceDelimited"` // implicitly imploded
//	}
//
//	// set different name - ?id=1,2,3
//	var req struct {
//		FilterClientIDs []int `oas:"id,query,form"` // implicitly imploded
//	}
//
// Use [encoding.TextUnmarshaler] to implement custom decoding.
//
// Decoding of request headers is NOT yet implemented.
//
// Decoding of request body is simple - it uses either json or xml unmarshaller:
//
//	type Entity struct {
//		ID int
//	}
//
//	// If no field tag value specified, "Accept" request header is used to determine decoding. Uses json by default.
//	var req struct {
//		Entity `oas:",body"`
//	}
//
//	// Always use JSON umarshalling, ignore "Accept" request header:
//	var req struct {
//		Entity `oas:",body,json"`
//	}
//
//	// Always use XML unmarshalling, ignore "Accept" request header:
//	var req struct {
//		Entity `oas:",body,xml"`
//	}
//
// [Query Serialization]: https://swagger.io/docs/specification/serialization/#query
func (d Decoder) Decode(r *http.Request, i any) error {
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return errors.New("call of Decode passes non-pointer as second argument")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("call of Decode passes pointer to non-struct as second argument")
	}

	// query values lookup by its original and lowercased name
	// TODO(jhorsts): why lowercase? investigate and apply the correct solution
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
		origin, conf := parseFieldConf(field.Type)

		// ignore
		if conf.name == "-" {
			continue
		}

		switch origin {
		case originQuery:
			err := d.decodeQuery(field.Value, conf, query)
			if err != nil {
				return err
			}
		case originBody:
			err := decodeBody(r, field.Value.Addr().Interface(), conf)
			if err != nil {
				return err
			}
		case originPath:
			err := setValue(field.Value, []string{d.pathValue(r, conf.name)})
			if err != nil {
				return fmt.Errorf("path '%s': %w", conf, err)
			}
		case originHeader:
			err := decodeHeaders()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// fieldConf contains extracted field tag values - <name>,<comma separated conf>.
type fieldConf struct {
	name string
	conf []string
}

func parseFieldConf(sf reflect.StructField) (origin string, conf fieldConf) {
	origin = "query"

	tagValue, ok := sf.Tag.Lookup(fieldTagName)
	if !ok {
		conf.name = strings.ToLower(sf.Name)
		return origin, conf
	}

	values := strings.Split(tagValue, ",")
	conf.name, conf.conf = values[0], values[1:]

	if conf.name == "" {
		conf.name = strings.ToLower(sf.Name)
	}

	found := -1

	for i, v := range conf.conf {
		conf.conf[i] = strings.TrimSpace(v)

		switch v {
		case originBody, originHeader, originPath, originQuery:
			origin = v
			found = i
		}
	}

	// remove origin from settings, order does not matter
	if found >= 0 {
		n := len(conf.conf) - 1
		conf.conf[found] = conf.conf[n]
		conf.conf = conf.conf[:n]
	}

	return origin, conf
}

type field struct {
	origin string
	conf   fieldConf
	Value  reflect.Value
	Type   reflect.StructField
}

// flattenFields flattens all fields of struct, the following fields are not flattened:
// - field tags having origin "body";
// - field tags having origin "query" with "deepObject" style;
// - fields having encoding.TextUnmarshaler interface.
func flattenFields(v reflect.Value) []field {
	ft := v.Type()

	fields := make([]field, 0, ft.NumField())

	for i := range ft.NumField() {
		f := field{Type: ft.Field(i)}

		// NOTE: ignore unexported fields in struct.
		if !f.Type.IsExported() {
			continue
		}

		f.Value = v.Field(i)
		f.origin, f.conf = parseFieldConf(f.Type)

		if _, ok := f.Value.Addr().Interface().(encoding.TextUnmarshaler); ok {
			fields = append(fields, f)
			continue
		}

		if f.Value.Kind() == reflect.Struct &&
			f.origin == originQuery && !slices.Contains(f.conf.conf, QueryStyleDeepObject) {
			fields = append(fields, flattenFields(f.Value)...)

			continue
		}

		fields = append(fields, f)
	}

	return fields
}

type fieldQueryConf struct {
	name     string // query name
	style    string // serialization style
	exploded bool   // whether exploded values
	required bool
}

func (d Decoder) parseQueryFieldConf(tagConf fieldConf) (fieldQueryConf, error) {
	conf := fieldQueryConf{
		exploded: d.query.exploded,
		style:    d.query.style,
		name:     tagConf.name,
	}

	if len(tagConf.conf) == 0 {
		return conf, nil
	}

	for _, setting := range tagConf.conf {
		switch setting {
		default:
			return fieldQueryConf{}, fmt.Errorf("invalid part '%s'", setting)
		case "required":
			conf.required = true
		case "explode":
			conf.exploded = true
		case "implode":
			conf.exploded = false
		case QueryStyleForm, QueryStylePipeDelimited, QueryStyleSpaceDelimited:
			conf.style = setting
			conf.exploded = false // TODO(jhorsts): should I remove it? OAS stipulates "explode" by default.
		case QueryStyleDeepObject:
			conf.style = setting
		}
	}

	// deepObject allows only exploded
	if conf.style == QueryStyleDeepObject {
		conf.exploded = true
	}

	return conf, nil
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
func parseQueryValues(queryConf fieldQueryConf, query map[string][]string) ([]string, bool) {
	values, ok := query[queryConf.name]
	if !ok {
		return nil, false
	}

	if queryConf.exploded || len(values) == 0 {
		return values, true
	}

	// Query is imploded. Always read the last value when expected imploded query, but received exploded - "?v=1&v=2".
	last := values[len(values)-1]

	delimiter := ","

	switch queryConf.style {
	case QueryStyleSpaceDelimited:
		delimiter = " "
	case QueryStylePipeDelimited:
		delimiter = "|"
	}

	return strings.Split(last, delimiter), true
}

func decodeBody(r *http.Request, i any, conf fieldConf) error {
	var format string

	switch {
	default:
		accept := strings.ToLower(r.Header.Get("Accept"))

		if strings.HasPrefix(accept, "application/json") {
			format = "json"
		} else if strings.HasPrefix(accept, "application/xml") {
			format = "xml"
		}
	case slices.Contains(conf.conf, "json"):
		format = "json"
	case slices.Contains(conf.conf, "xml"):
		format = "xml"
	}

	switch format {
	default:
		return fmt.Errorf(`want "xml" or "json", got unsupported "%s"`, fieldTagName)
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

func (d Decoder) decodeQuery(fv reflect.Value, conf fieldConf, query map[string][]string) error {
	queryConf, err := d.parseQueryFieldConf(conf)
	if err != nil {
		return err
	}

	// deep object
	if queryConf.style == QueryStyleDeepObject {
		qv := parseQueryValuesDeep(queryConf.name, query)
		if queryConf.required && len(qv) == 0 {
			return fmt.Errorf("query param '%s' is required", queryConf.name)
		}

		err = d.setDeepValue(fv, qv)
		if err != nil {
			return fmt.Errorf("query param '%s': %w", queryConf.name, err)
		}

		return nil
	}

	// normal query
	qv, ok := parseQueryValues(queryConf, query)
	if !ok {
		if queryConf.required {
			return fmt.Errorf("query param '%s' is required", queryConf.name)
		}

		if len(qv) == 0 {
			return nil
		}
	}

	err = setValue(fv, qv)
	if err != nil {
		return fmt.Errorf("query param '%s': %w", queryConf.name, err)
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
		err := e.UnmarshalText([]byte(value))
		if err != nil {
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

				err := setValue(v, []string{value})
				if err != nil {
					return err
				}

				slice = reflect.Append(slice, v)
			}
		}

		rv.Set(slice)
	}

	return nil
}

func (d Decoder) setDeepValue(rv reflect.Value, query map[string][]string) error {
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}

		rv = rv.Elem()
	}

	if kind := rv.Kind(); kind != reflect.Struct {
		return fmt.Errorf("want struct for deepObject, got %s", kind)
	}

	for i := range rv.NumField() {
		origin, conf := parseFieldConf(rv.Type().Field(i))

		// ignore
		if origin != originQuery && conf.name == "-" {
			continue
		}

		err := setValue(rv.Field(i), query[conf.name])
		if err != nil {
			return err
		}
	}

	return nil
}
