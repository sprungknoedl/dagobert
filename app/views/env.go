package views

import (
	"github.com/sprungknoedl/dagobert/app/model"
)

// Env is the core view-model injected into template renders via middleware.
// It carries the current case, user, customizable enums, active route, and the
// ACL predicate used to gate links and actions.
type Env struct {
	Case    model.Case
	User    model.User
	Enums   model.Enums
	Route   string
	Allowed func(method, url string) (string, bool)
}
