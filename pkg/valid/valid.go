package valid

import (
	"strings"
)

type Condition struct {
	Name    string
	Message string
	Missing bool
	Invalid bool
}

func (c Condition) String() string {
	if c.Missing {
		return "Missing required field."
	} else if c.Invalid {
		return c.Message
	} else {
		return ""
	}
}

type ValidationError map[string]Condition

func Check(conds []Condition) ValidationError {
	r := ValidationError{}
	for _, c := range conds {
		if c.Missing || c.Invalid {
			r[c.Name] = c
		}
	}

	if len(r) > 0 {
		return r
	} else {
		return nil
	}
}

func (r ValidationError) Error() string {
	parts := []string{}
	for _, v := range r {
		parts = append(parts, v.String())
	}

	return strings.Join(parts, "\n")
}

func (r ValidationError) Valid() bool {
	return len(r) == 0
}
