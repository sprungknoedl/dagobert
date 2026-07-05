package doct

import (
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHead(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   any
		want any
	}{
		{"middle", 2, []int{1, 2, 3, 4}, []int{1, 2}},
		{"clamp high", 10, []int{1, 2, 3}, []int{1, 2, 3}},
		{"clamp negative", -1, []int{1, 2, 3}, []int{}},
		{"empty", 2, []int{}, []int{}},
		{"non-slice", 2, "abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, head(tt.n, tt.in))
		})
	}
}

func TestTail(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   any
		want any
	}{
		{"middle", 2, []int{1, 2, 3, 4}, []int{3, 4}},
		{"clamp high", 10, []int{1, 2, 3}, []int{1, 2, 3}},
		{"clamp negative", -1, []int{1, 2, 3}, []int{}},
		{"empty", 2, []int{}, []int{}},
		{"non-slice", 2, "abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tail(tt.n, tt.in))
		})
	}
}

func TestFirst(t *testing.T) {
	assert.Equal(t, 1, first([]int{1, 2, 3}))
	assert.Nil(t, first([]int{}))
	assert.Nil(t, first("not a slice"))
}

func TestLast(t *testing.T) {
	assert.Equal(t, 3, last([]int{1, 2, 3}))
	assert.Nil(t, last([]int{}))
	assert.Nil(t, last("not a slice"))
}

func TestReverse(t *testing.T) {
	assert.Equal(t, []int{3, 2, 1}, reverse([]int{1, 2, 3}))
	assert.Equal(t, []int{}, reverse([]int{}))
	assert.Equal(t, "abc", reverse("abc"))
}

func TestTitle(t *testing.T) {
	assert.Equal(t, "Hello World", title("hello world"))
	assert.Equal(t, "Hello World", title("  hello   world  "))
	assert.Equal(t, "Über Café", title("über café"))
	assert.Equal(t, "", title(""))
}

func TestReplace(t *testing.T) {
	assert.Equal(t, "b-b-b", replace("a", "b", "a-a-a"))
	assert.Equal(t, "abc", replace("x", "y", "abc"))
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   string
		want string
	}{
		{"shorter", 10, "hello", "hello"},
		{"exact", 5, "hello", "hello"},
		{"truncated", 3, "hello", "hel…"},
		{"runes not bytes", 2, "café", "ca…"},
		{"keeps multibyte", 4, "café", "café"},
		{"negative", -1, "hello", "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.n, tt.in))
		})
	}
}

func TestJoin(t *testing.T) {
	assert.Equal(t, "a, b, c", join(", ", []string{"a", "b", "c"}))
	assert.Equal(t, "a, b, c", join(", ", model.Strings{"a", "b", "c"}))
	assert.Equal(t, "1-2-3", join("-", []any{1, 2, 3}))
	assert.Equal(t, "", join(", ", []string{}))
	assert.Equal(t, "x", join(", ", "x"))
}

func TestDefault(t *testing.T) {
	assert.Equal(t, "fallback", defaultVal("fallback", ""))
	assert.Equal(t, "fallback", defaultVal("fallback", nil))
	assert.Equal(t, 7, defaultVal(7, 0))
	assert.Equal(t, "fallback", defaultVal("fallback", []string{}))
	assert.Equal(t, "value", defaultVal("fallback", "value"))
	assert.Equal(t, 3, defaultVal(7, 3))
	assert.Equal(t, []string{"a"}, defaultVal("fallback", []string{"a"}))
}
