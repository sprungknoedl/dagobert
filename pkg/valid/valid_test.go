package valid

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionString(t *testing.T) {
	t.Run("Missing", func(t *testing.T) {
		c := Condition{Name: "Name", Missing: true}
		assert.Equal(t, "Missing required field.", c.String())
	})

	t.Run("Invalid", func(t *testing.T) {
		c := Condition{Name: "Type", Message: "Invalid type.", Invalid: true}
		assert.Equal(t, "Invalid type.", c.String())
	})

	t.Run("Missing Takes Precedence Over Invalid", func(t *testing.T) {
		c := Condition{Name: "Type", Message: "Invalid type.", Missing: true, Invalid: true}
		assert.Equal(t, "Missing required field.", c.String())
	})

	t.Run("Invalid Without Message", func(t *testing.T) {
		c := Condition{Name: "Type", Invalid: true}
		assert.Equal(t, "", c.String())
	})

	t.Run("Satisfied", func(t *testing.T) {
		c := Condition{Name: "Name", Message: "Invalid value."}
		assert.Equal(t, "", c.String())
	})
}

func TestCheck(t *testing.T) {
	t.Run("All Satisfied Returns Nil", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Name"},
			{Name: "Type", Message: "Invalid type."},
		})
		assert.Nil(t, vr)
	})

	t.Run("Empty Conditions Returns Nil", func(t *testing.T) {
		assert.Nil(t, Check([]Condition{}))
	})

	t.Run("Nil Conditions Returns Nil", func(t *testing.T) {
		assert.Nil(t, Check(nil))
	})

	t.Run("Collects Missing", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Name", Missing: true},
			{Name: "Type"},
		})
		assert.Len(t, vr, 1)
		assert.True(t, vr["Name"].Missing)
	})

	t.Run("Collects Invalid", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Name"},
			{Name: "Type", Message: "Invalid type.", Invalid: true},
		})
		assert.Len(t, vr, 1)
		assert.Equal(t, "Invalid type.", vr["Type"].Message)
	})

	t.Run("Keyed By Name", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Name", Missing: true},
			{Name: "Type", Invalid: true},
			{Name: "Status"},
		})
		assert.Len(t, vr, 2)
		assert.Contains(t, vr, "Name")
		assert.Contains(t, vr, "Type")
		assert.NotContains(t, vr, "Status")
	})
}

func TestValidationErrorError(t *testing.T) {
	t.Run("Single Condition", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Type", Message: "Invalid type.", Invalid: true},
		})
		assert.EqualError(t, vr, "Invalid type.")
	})

	t.Run("Multiple Conditions Joined By Newline", func(t *testing.T) {
		vr := Check([]Condition{
			{Name: "Name", Missing: true},
			{Name: "Type", Message: "Invalid type.", Invalid: true},
		})

		// map iteration order is not deterministic
		parts := strings.Split(vr.Error(), "\n")
		sort.Strings(parts)
		assert.Equal(t, []string{"Invalid type.", "Missing required field."}, parts)
	})

	t.Run("Implements Error Interface", func(t *testing.T) {
		var err error = Check([]Condition{{Name: "Name", Missing: true}})
		verr, ok := err.(ValidationError)
		assert.True(t, ok)
		assert.False(t, verr.Valid())
	})
}

func TestValidationErrorValid(t *testing.T) {
	t.Run("Nil Is Valid", func(t *testing.T) {
		var vr ValidationError
		assert.True(t, vr.Valid())
	})

	t.Run("Nil Check Result Is Valid", func(t *testing.T) {
		vr := Check([]Condition{{Name: "Name"}})
		assert.True(t, vr.Valid())
	})

	t.Run("Empty Is Valid", func(t *testing.T) {
		assert.True(t, ValidationError{}.Valid())
	})

	t.Run("Non-Empty Is Invalid", func(t *testing.T) {
		vr := Check([]Condition{{Name: "Name", Missing: true}})
		assert.False(t, vr.Valid())
	})
}
