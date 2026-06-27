package doct

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// helperFuncs is the curated set of template helpers available to report
// templates, on top of the load-time "xml" escaper. They are pure functions
// with no access to the template or zip state. List helpers use reflect only
// for structural slice operations (Len/Index/Slice) and element
// stringification, never for field-name access.
var helperFuncs = template.FuncMap{
	// List
	"head":    head,
	"tail":    tail,
	"first":   first,
	"last":    last,
	"reverse": reverse,
	// String
	"upper":    strings.ToUpper,
	"lower":    strings.ToLower,
	"title":    title,
	"trim":     strings.TrimSpace,
	"replace":  replace,
	"truncate": truncate,
	"join":     join,
	"default":  defaultVal,
}

// head returns the first n elements of a slice; n is clamped to [0, len].
// A non-slice argument is returned unchanged.
func head(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	n = clamp(n, v.Len())
	return v.Slice(0, n).Interface()
}

// tail returns the last n elements of a slice; n is clamped to [0, len].
// A non-slice argument is returned unchanged.
func tail(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	n = clamp(n, v.Len())
	return v.Slice(v.Len()-n, v.Len()).Interface()
}

// first returns the first element of a slice, or nil if empty/non-slice.
func first(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}
	return v.Index(0).Interface()
}

// last returns the last element of a slice, or nil if empty/non-slice.
func last(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}
	return v.Index(v.Len() - 1).Interface()
}

// reverse returns a reversed copy of a slice. A non-slice argument is returned
// unchanged.
func reverse(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	for i := 0; i < v.Len(); i++ {
		out.Index(v.Len() - 1 - i).Set(v.Index(i))
	}
	return out.Interface()
}

// title upper-cases the first rune of each whitespace-separated word.
func title(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[size:]
	}
	return strings.Join(words, " ")
}

// replace returns s with all instances of old replaced by repl. Argument order
// reads well in a pipeline: {{ .X | replace "a" "b" }}.
func replace(old, repl, s string) string {
	return strings.ReplaceAll(s, old, repl)
}

// truncate caps s to n runes (not bytes), appending an ellipsis only when the
// string was actually shortened.
func truncate(n int, s string) string {
	if n < 0 {
		n = 0
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// join stringifies each element of slice xs via fmt.Sprint and joins them with
// sep. A non-slice argument is stringified directly.
func join(sep string, xs any) string {
	v := reflect.ValueOf(xs)
	if v.Kind() != reflect.Slice {
		return fmt.Sprint(xs)
	}
	parts := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		parts[i] = fmt.Sprint(v.Index(i).Interface())
	}
	return strings.Join(parts, sep)
}

// defaultVal returns d when v is its zero value (nil, empty string, 0, empty
// slice/map), else v.
func defaultVal(d, v any) any {
	if v == nil {
		return d
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		if rv.Len() == 0 {
			return d
		}
		return v
	}
	if rv.IsZero() {
		return d
	}
	return v
}

// clamp bounds n to [0, max].
func clamp(n, max int) int {
	if n < 0 {
		return 0
	}
	if n > max {
		return max
	}
	return n
}
