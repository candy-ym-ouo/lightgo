package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Source int

const (
	SourceAuto Source = iota
	SourceJSON
	SourceForm
	SourceQuery
	SourceParam
	SourceHeader
)

type Params interface{ Param(string) string }

func Bind(r *http.Request, dst any, source Source, params Params) error {
	if err := checkDestination(dst); err != nil {
		return err
	}
	if source == SourceAuto {
		mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		switch mediaType {
		case "application/json":
			source = SourceJSON
		case "application/x-www-form-urlencoded", "multipart/form-data":
			source = SourceForm
		default:
			source = SourceQuery
		}
	}
	var err error
	switch source {
	case SourceJSON:
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		err = dec.Decode(dst)
	case SourceForm:
		err = r.ParseMultipartForm(32 << 20)
		if err == http.ErrNotMultipart {
			err = r.ParseForm()
		}
		if err == nil {
			err = bindValues(reflect.ValueOf(dst).Elem(), r.Form, "form", nil)
		}
	case SourceQuery:
		err = bindValues(reflect.ValueOf(dst).Elem(), r.URL.Query(), "query", nil)
	case SourceParam:
		err = bindValues(reflect.ValueOf(dst).Elem(), nil, "param", params)
	case SourceHeader:
		err = bindHeader(reflect.ValueOf(dst).Elem(), r.Header)
	default:
		err = errors.New("unknown binding source")
	}
	if err != nil {
		return err
	}
	applyDefaults(reflect.ValueOf(dst).Elem())
	return Validate(dst)
}
func BindJSON(r *http.Request, dst any) error            { return Bind(r, dst, SourceJSON, nil) }
func BindForm(r *http.Request, dst any) error            { return Bind(r, dst, SourceForm, nil) }
func BindQuery(r *http.Request, dst any) error           { return Bind(r, dst, SourceQuery, nil) }
func BindHeader(r *http.Request, dst any) error          { return Bind(r, dst, SourceHeader, nil) }
func BindParam(r *http.Request, dst any, p Params) error { return Bind(r, dst, SourceParam, p) }
func checkDestination(dst any) error {
	if dst == nil {
		return errors.New("binding destination is nil")
	}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return errors.New("binding destination must be a non-nil struct pointer")
	}
	return nil
}
func bindValues(v reflect.Value, values url.Values, tagName string, params Params) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f, sf := v.Field(i), t.Field(i)
		if !sf.IsExported() || !f.CanSet() {
			continue
		}
		if f.Kind() == reflect.Struct {
			if err := bindValues(f, values, tagName, params); err != nil {
				return err
			}
			continue
		}
		name := fieldName(sf, tagName)
		if name == "-" {
			continue
		}
		var raw []string
		if tagName == "param" && params != nil {
			if value := params.Param(name); value != "" {
				raw = []string{value}
			}
		} else {
			raw = values[name]
		}
		if len(raw) == 0 {
			continue
		}
		if err := setField(f, raw); err != nil {
			return fmt.Errorf("bind %s: %w", sf.Name, err)
		}
	}
	return nil
}
func bindHeader(v reflect.Value, h http.Header) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f, sf := v.Field(i), t.Field(i)
		if !sf.IsExported() || !f.CanSet() {
			continue
		}
		name := sf.Tag.Get("header")
		if name == "" {
			name = sf.Name
		}
		if values := h.Values(name); len(values) > 0 {
			if err := setField(f, values); err != nil {
				return fmt.Errorf("bind %s: %w", sf.Name, err)
			}
		}
	}
	return nil
}
func fieldName(sf reflect.StructField, tag string) string {
	name := sf.Tag.Get(tag)
	if name == "" && tag != "json" {
		name = sf.Tag.Get("json")
	}
	if before, _, ok := strings.Cut(name, ","); ok {
		name = before
	}
	if name == "" {
		name = strings.ToLower(sf.Name[:1]) + sf.Name[1:]
	}
	return name
}
func setField(field reflect.Value, values []string) error {
	if field.Kind() == reflect.Slice {
		parts := values
		if len(values) == 1 {
			parts = strings.Split(values[0], ",")
		}
		slice := reflect.MakeSlice(field.Type(), 0, len(parts))
		for _, raw := range parts {
			item := reflect.New(field.Type().Elem()).Elem()
			if err := setScalar(item, strings.TrimSpace(raw)); err != nil {
				return err
			}
			slice = reflect.Append(slice, item)
		}
		field.Set(slice)
		return nil
	}
	return setScalar(field, values[0])
}
func setScalar(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(v)
	default:
		return fmt.Errorf("unsupported kind %s", field.Kind())
	}
	return nil
}
func applyDefaults(v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f, sf := v.Field(i), t.Field(i)
		if f.Kind() == reflect.Struct {
			applyDefaults(f)
			continue
		}
		if raw := sf.Tag.Get("default"); raw != "" && f.CanSet() && isEmpty(f) {
			_ = setField(f, []string{raw})
		}
	}
}
