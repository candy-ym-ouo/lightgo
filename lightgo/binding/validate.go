package binding

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message,omitempty"`
}
type ValidationErrors []FieldError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s %s", e[0].Field, e[0].Rule)
}

var (
	emailPattern   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	numericPattern = regexp.MustCompile(`^[+-]?(?:\d+\.?\d*|\.\d+)$`)
	alphaPattern   = regexp.MustCompile(`^[A-Za-z]+$`)
)

func Validate(value any) error {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	return validateStruct(v, "")
}
func validateStruct(v reflect.Value, prefix string) error {
	t := v.Type()
	var out ValidationErrors
	for i := 0; i < v.NumField(); i++ {
		field, sf := v.Field(i), t.Field(i)
		if !sf.IsExported() {
			continue
		}
		name := sf.Name
		if prefix != "" {
			name = prefix + "." + name
		}
		if field.Kind() == reflect.Struct && field.Type() != reflect.TypeOf(time.Time{}) {
			if err := validateStruct(field, name); err != nil {
				out = append(out, err.(ValidationErrors)...)
			}
		}
		rules := splitRules(sf.Tag.Get("validate"))
		for _, rule := range rules {
			key, arg, _ := strings.Cut(rule, "=")
			if !rulePass(field, key, arg) {
				out = append(out, FieldError{Field: name, Rule: rule, Message: validationMessage(name, key, arg)})
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	return nil
}
func splitRules(tag string) []string {
	if strings.TrimSpace(tag) == "" {
		return nil
	}
	return strings.Split(tag, "|")
}
func rulePass(v reflect.Value, rule, arg string) bool {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return rule != "required"
		}
		v = v.Elem()
	}
	if rule != "required" && isEmpty(v) {
		return true
	}
	s := valueString(v)
	switch rule {
	case "required":
		return !isEmpty(v)
	case "min":
		n, err := strconv.Atoi(arg)
		return err == nil && measure(v) >= n
	case "max":
		n, err := strconv.Atoi(arg)
		return err == nil && measure(v) <= n
	case "len":
		n, err := strconv.Atoi(arg)
		return err == nil && measure(v) == n
	case "email":
		return emailPattern.MatchString(s)
	case "numeric":
		return numericPattern.MatchString(s)
	case "alpha":
		return alphaPattern.MatchString(s)
	case "oneof":
		for _, allowed := range strings.Fields(arg) {
			if s == allowed {
				return true
			}
		}
		return false
	case "regexp":
		rx, err := regexp.Compile(arg)
		return err == nil && rx.MatchString(s)
	case "datetime":
		_, err := time.Parse(arg, s)
		return err == nil
	default:
		return true
	}
}
func isEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return v.IsZero()
}
func measure(v reflect.Value) int {
	switch v.Kind() {
	case reflect.String:
		return len([]rune(v.String()))
	case reflect.Array, reflect.Slice, reflect.Map:
		return v.Len()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int(v.Float())
	}
	return 0
}
func valueString(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	}
	return fmt.Sprint(v.Interface())
}
func validationMessage(field, rule, arg string) string {
	switch rule {
	case "required":
		return field + " is required"
	case "min":
		return field + " must be at least " + arg
	case "max":
		return field + " must be at most " + arg
	case "oneof":
		return field + " must be one of " + arg
	default:
		return field + " failed " + rule
	}
}
