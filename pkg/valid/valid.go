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

type Result map[string]Condition

func Check(conds []Condition) Result {
	r := Result{}
	for _, c := range conds {
		if c.Missing || c.Invalid {
			r[c.Name] = c
		}
	}

	return r
}

func (r Result) Error() string {
	parts := []string{}
	for _, v := range r {
		parts = append(parts, v.String())
	}

	return strings.Join(parts, "\n")
}

func (r Result) Valid() bool {
	return len(r) == 0
}
