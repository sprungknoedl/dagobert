package model

import (
	"fmt"
	"hash/fnv"
	"time"
)

func HashFields(fields ...any) string {
	h := fnv.New32a()

	for _, f := range fields {
		switch v := f.(type) {
		case string:
			h.Write([]byte(v))
		case int64:
			fmt.Fprintf(h, "%d", v)
		case bool:
			fmt.Fprintf(h, "%v", v)
		case time.Time:
			h.Write([]byte(v.Format(time.RFC3339)))
		}
	}

	return string(h.Sum(nil))
}
