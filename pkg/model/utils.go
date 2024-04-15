package model

import (
	"os"
	"strings"
)

func FromEnv(name string, defaults []string) []string {
	list := strings.Split(os.Getenv(name), ";")
	if len(list) > 1 {
		return list
	}
	return defaults
}
