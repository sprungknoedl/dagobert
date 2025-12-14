package fp

import "iter"

func Apply[A any, B any](in []A, fn func(A) B) []B {
	out := make([]B, len(in))
	for i, a := range in {
		out[i] = fn(a)
	}
	return out
}

func ApplyS[A any, B any](in iter.Seq[A], fn func(A) B) []B {
	out := make([]B, 0)
	for a := range in {
		out = append(out, fn(a))
	}
	return out
}

func ApplyM[A any, B any, K comparable](in map[K]A, fn func(A) B) map[K]B {
	out := make(map[K]B, len(in))
	for k, a := range in {
		out[k] = fn(a)
	}
	return out
}

func Filter[A any](in []A, fn func(A) bool) []A {
	out := make([]A, 0, len(in))
	for _, a := range in {
		if fn(a) {
			out = append(out, a)
		}
	}
	return out
}

func FilterS[A any](in iter.Seq[A], fn func(A) bool) []A {
	out := make([]A, 0)
	for a := range in {
		if fn(a) {
			out = append(out, a)
		}
	}
	return out
}

func FilterM[A any, K comparable](in map[K]A, fn func(A) bool) map[K]A {
	out := make(map[K]A, len(in))
	for k, a := range in {
		if fn(a) {
			out[k] = a
		}
	}
	return out
}

func If[T any](cond bool, a T, b T) T {
	if cond {
		return a
	}
	return b
}

func ToMap[A any, K comparable](in []A, fn func(A) K) map[K]A {
	out := make(map[K]A, len(in))
	for _, a := range in {
		out[fn(a)] = a
	}
	return out
}

func ToList[A any, K comparable](in map[K]A) []A {
	i := 0
	out := make([]A, len(in))
	for _, a := range in {
		out[i] = a
		i++
	}
	return out
}
