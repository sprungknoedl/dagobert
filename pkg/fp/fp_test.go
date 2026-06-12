package fp

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
	t.Run("Maps Values", func(t *testing.T) {
		got := Apply([]int{1, 2, 3}, strconv.Itoa)
		assert.Equal(t, []string{"1", "2", "3"}, got)
	})

	t.Run("Preserves Order", func(t *testing.T) {
		got := Apply([]string{"c", "a", "b"}, strings.ToUpper)
		assert.Equal(t, []string{"C", "A", "B"}, got)
	})

	t.Run("Empty Slice", func(t *testing.T) {
		got := Apply([]int{}, strconv.Itoa)
		assert.Equal(t, []string{}, got)
	})

	t.Run("Nil Slice", func(t *testing.T) {
		got := Apply(nil, strconv.Itoa)
		assert.Equal(t, []string{}, got)
	})
}

func TestApplyS(t *testing.T) {
	t.Run("Maps Values", func(t *testing.T) {
		got := ApplyS(slices.Values([]int{1, 2, 3}), strconv.Itoa)
		assert.Equal(t, []string{"1", "2", "3"}, got)
	})

	t.Run("Empty Sequence", func(t *testing.T) {
		got := ApplyS(slices.Values([]int{}), strconv.Itoa)
		assert.Equal(t, []string{}, got)
	})
}

func TestApplyM(t *testing.T) {
	t.Run("Maps Values Keeps Keys", func(t *testing.T) {
		got := ApplyM(map[string]int{"a": 1, "b": 2}, strconv.Itoa)
		assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)
	})

	t.Run("Empty Map", func(t *testing.T) {
		got := ApplyM(map[string]int{}, strconv.Itoa)
		assert.Equal(t, map[string]string{}, got)
	})

	t.Run("Nil Map", func(t *testing.T) {
		got := ApplyM[int, string, string](nil, strconv.Itoa)
		assert.Equal(t, map[string]string{}, got)
	})
}

func TestFilter(t *testing.T) {
	even := func(i int) bool { return i%2 == 0 }

	t.Run("Keeps Matching Elements In Order", func(t *testing.T) {
		got := Filter([]int{1, 2, 3, 4, 5, 6}, even)
		assert.Equal(t, []int{2, 4, 6}, got)
	})

	t.Run("No Matches", func(t *testing.T) {
		got := Filter([]int{1, 3, 5}, even)
		assert.Equal(t, []int{}, got)
	})

	t.Run("Nil Slice", func(t *testing.T) {
		got := Filter(nil, even)
		assert.Equal(t, []int{}, got)
	})
}

func TestFilterS(t *testing.T) {
	even := func(i int) bool { return i%2 == 0 }

	t.Run("Keeps Matching Elements In Order", func(t *testing.T) {
		got := FilterS(slices.Values([]int{1, 2, 3, 4}), even)
		assert.Equal(t, []int{2, 4}, got)
	})

	t.Run("Empty Sequence", func(t *testing.T) {
		got := FilterS(slices.Values([]int{}), even)
		assert.Equal(t, []int{}, got)
	})
}

func TestFilterM(t *testing.T) {
	even := func(i int) bool { return i%2 == 0 }

	t.Run("Keeps Matching Entries", func(t *testing.T) {
		got := FilterM(map[string]int{"a": 1, "b": 2, "c": 4}, even)
		assert.Equal(t, map[string]int{"b": 2, "c": 4}, got)
	})

	t.Run("No Matches", func(t *testing.T) {
		got := FilterM(map[string]int{"a": 1, "b": 3}, even)
		assert.Equal(t, map[string]int{}, got)
	})

	t.Run("Nil Map", func(t *testing.T) {
		got := FilterM[int, string](nil, even)
		assert.Equal(t, map[string]int{}, got)
	})
}

func TestIf(t *testing.T) {
	t.Run("True Branch", func(t *testing.T) {
		assert.Equal(t, "a", If(true, "a", "b"))
	})

	t.Run("False Branch", func(t *testing.T) {
		assert.Equal(t, "b", If(false, "a", "b"))
	})
}

func TestToMap(t *testing.T) {
	type item struct {
		ID   string
		Name string
	}
	key := func(i item) string { return i.ID }

	t.Run("Keys By Function", func(t *testing.T) {
		got := ToMap([]item{{"x", "one"}, {"y", "two"}}, key)
		assert.Equal(t, map[string]item{
			"x": {"x", "one"},
			"y": {"y", "two"},
		}, got)
	})

	t.Run("Duplicate Keys Last Wins", func(t *testing.T) {
		got := ToMap([]item{{"x", "first"}, {"x", "second"}}, key)
		assert.Equal(t, map[string]item{"x": {"x", "second"}}, got)
	})

	t.Run("Nil Slice", func(t *testing.T) {
		got := ToMap(nil, key)
		assert.Equal(t, map[string]item{}, got)
	})
}

func TestToList(t *testing.T) {
	t.Run("Returns All Values", func(t *testing.T) {
		got := ToList(map[string]int{"a": 1, "b": 2, "c": 3})
		assert.ElementsMatch(t, []int{1, 2, 3}, got)
	})

	t.Run("Nil Map", func(t *testing.T) {
		got := ToList[int, string](nil)
		assert.Equal(t, []int{}, got)
	})
}

func TestValues(t *testing.T) {
	got := Values(map[string]int{"a": 1, "b": 2})
	assert.ElementsMatch(t, []int{1, 2}, got)
}

func TestKeys(t *testing.T) {
	t.Run("Returns All Keys", func(t *testing.T) {
		got := Keys(map[string]int{"a": 1, "b": 2})
		assert.ElementsMatch(t, []string{"a", "b"}, got)
	})

	t.Run("Nil Map", func(t *testing.T) {
		got := Keys[int, string](nil)
		assert.Equal(t, []string{}, got)
	})
}
