package tty

import "fmt"

const (
	blue    = "\033[97;44m"
	cyan    = "\033[97;46m"
	green   = "\033[97;42m"
	magenta = "\033[97;45m"
	red     = "\033[97;41m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	reset   = "\033[0m"
)

type Fn func(string) string

func Blue(str string) string {
	return fmt.Sprintf("%s%s%s", blue, str, reset)
}

func Cyan(str string) string {
	return fmt.Sprintf("%s%s%s", cyan, str, reset)
}

func Green(str string) string {
	return fmt.Sprintf("%s%s%s", green, str, reset)
}

func Magenta(str string) string {
	return fmt.Sprintf("%s%s%s", magenta, str, reset)
}

func Red(str string) string {
	return fmt.Sprintf("%s%s%s", red, str, reset)
}

func White(str string) string {
	return fmt.Sprintf("%s%s%s", white, str, reset)
}

func Yellow(str string) string {
	return fmt.Sprintf("%s%s%s", yellow, str, reset)
}
