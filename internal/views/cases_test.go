package views

import (
	"strconv"
	"testing"

	"github.com/sprungknoedl/dagobert/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSplitAssignees(t *testing.T) {
	users := make([]model.User, 7)
	for i := range users {
		users[i] = model.User{ID: strconv.Itoa(i)}
	}

	t.Run("caps at max, returning the rest as hidden", func(t *testing.T) {
		shown, hidden := splitAssignees(users, 5)
		assert.Len(t, shown, 5)
		assert.Len(t, hidden, 2)
	})

	t.Run("max <= 0 means no cap", func(t *testing.T) {
		shown, hidden := splitAssignees(users, 0)
		assert.Len(t, shown, 7)
		assert.Empty(t, hidden)
	})

	t.Run("fewer assignees than max returns none hidden", func(t *testing.T) {
		shown, hidden := splitAssignees(users[:3], 5)
		assert.Len(t, shown, 3)
		assert.Empty(t, hidden)
	})
}

func TestAssigneeInitials(t *testing.T) {
	t.Run("first and last word", func(t *testing.T) {
		assert.Equal(t, "TK", assigneeInitials(model.User{Name: "Thomas Kastner"}))
	})

	t.Run("single-word name gives one letter", func(t *testing.T) {
		assert.Equal(t, "T", assigneeInitials(model.User{Name: "Thomas"}))
	})

	t.Run("empty name falls back to login", func(t *testing.T) {
		assert.Equal(t, "J", assigneeInitials(model.User{Name: "", Login: "jdoe"}))
	})
}
