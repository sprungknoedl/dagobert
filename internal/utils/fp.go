package utils

func Apply[A any, B any](in []A, fn func(A) B) []B {
	out := make([]B, len(in))
	for i, a := range in {
		out[i] = fn(a)
	}
	return out
}

func ApplyM[A any, B any, C comparable](in map[C]A, fn func(A) B) []B {
	i := 0
	out := make([]B, len(in))
	for _, a := range in {
		out[i] = fn(a)
		i += 1
	}
	return out
}

func If[T any](cond bool, a T, b T) T {
	if cond {
		return a
	}
	return b
}
