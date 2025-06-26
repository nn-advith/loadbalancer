package common

import (
	"golang.org/x/exp/constraints"
)

func GCD[T constraints.Integer](a, b T) T {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a // gcd is always positive irrespective of sign of numbers
	}
	return a
}

func IndexAll[T comparable](S []T, V T) []int {
	var indexes []int
	for i, v := range S {
		if v == V {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
