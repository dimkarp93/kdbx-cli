package utils

import (
	"cmp"
)

func Filter[T any](items []T, keep func(T) bool) []T {
	panic("not implemented")
}
func Map[T, R any](items []T, fn func(T) R) []R {
	panic("not implemented")
}
func FlatMap[T, R any](items []T, fn func(T) []R) []R {
	panic("not implemented")
}
func Unique[T comparable](items []T) []T {
	panic("not implemented")
}
func UniqueBy[T any, K comparable](items []T, key func(T) K) []T {
	panic("not implemented")
}
func GroupBy[T any, K comparable](items []T, key func(T) K) map[K][]T {
	panic("not implemented")
}
func ToSet[T comparable](items []T) map[T]bool {
	panic("not implemented")
}
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	panic("not implemented")
}
func SortedUnique[T cmp.Ordered](items []T) []T {
	panic("not implemented")
}
